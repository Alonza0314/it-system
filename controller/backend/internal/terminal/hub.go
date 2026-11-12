// Package terminal relays interactive shell sessions between an admin's
// browser and a runner.
//
// A runner that enables the terminal keeps one outbound control WebSocket to
// the controller. When an admin opens a terminal, the browser gets a one-time
// ticket over REST, connects its own WebSocket with that ticket, the
// controller asks the runner (over the control channel) to open a PTY, and the
// runner dials back a data WebSocket for that session. The controller then
// copies frames between the two data connections. The controller never
// connects to a runner and holds no runner credentials.
//
// See design/runner-terminal.md.
package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/coder/websocket"
)

const (
	controlReadLimit = 4 << 10
	dataReadLimit    = 64 << 10

	maxTerminalSize = 1000
)

// Authorizer answers the questions the hub asks the rest of the controller.
type Authorizer interface {
	RunnerExists(name string) bool
	// RunnerTokenCurrent reports whether a runner token's registration id
	// belongs to the runner's current registration.
	RunnerTokenCurrent(name, registrationID string) bool
	UserIsAdmin(user string) bool
}

// Logger is the subset of the controller logger the hub uses.
type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Options struct {
	TicketTTL            time.Duration
	IdleTimeout          time.Duration
	MaxDuration          time.Duration
	MaxSessionsPerRunner int
	MaxSessions          int

	// MaxTicketsPerUser and MaxTickets limit tickets issued but not used yet.
	MaxTicketsPerUser int
	MaxTickets        int

	// AllowedUsers limits who may open a terminal; empty means every admin.
	AllowedUsers []string
	// AllowedClientNets limits where the browser may connect from, by
	// Client.IP; empty means anywhere.
	AllowedClientNets []*net.IPNet
	// RequireHTTPS refuses browsers that did not reach the controller over
	// HTTPS (Client.HTTPS).
	RequireHTTPS bool
	// TrustedProxies are the reverse proxies whose X-Forwarded-Proto
	// header is believed; see ClientFor.
	TrustedProxies []netip.Prefix
	// OriginPatterns are extra browser origins accepted besides the
	// controller's own host (coder/websocket host patterns).
	OriginPatterns []string

	// OpenTimeout is how long to wait for the runner to dial back.
	OpenTimeout time.Duration
	// CheckInterval is how often idle time, the time limit and the admin's
	// access are checked.
	CheckInterval time.Duration
	// RecheckInterval is how often the admin's access is looked up again.
	RecheckInterval time.Duration
	// PingInterval keeps NAT entries alive and detects dead peers; a peer
	// that does not answer within PingTimeout is dropped.
	PingInterval time.Duration
	PingTimeout  time.Duration
}

func (o *Options) setDefaults() {
	if o.TicketTTL <= 0 {
		o.TicketTTL = 30 * time.Second
	}
	if o.IdleTimeout <= 0 {
		o.IdleTimeout = 15 * time.Minute
	}
	if o.MaxDuration <= 0 {
		o.MaxDuration = 2 * time.Hour
	}
	if o.MaxSessionsPerRunner <= 0 {
		o.MaxSessionsPerRunner = 2
	}
	if o.MaxSessions <= 0 {
		o.MaxSessions = 6
	}
	if o.MaxTicketsPerUser <= 0 {
		o.MaxTicketsPerUser = 10
	}
	if o.MaxTickets <= 0 {
		o.MaxTickets = 100
	}
	if o.OpenTimeout <= 0 {
		o.OpenTimeout = 10 * time.Second
	}
	if o.CheckInterval <= 0 {
		o.CheckInterval = 5 * time.Second
	}
	if o.RecheckInterval <= 0 {
		o.RecheckInterval = time.Minute
	}
	if o.PingInterval <= 0 {
		o.PingInterval = 30 * time.Second
	}
	if o.PingTimeout <= 0 {
		o.PingTimeout = 10 * time.Second
	}
}

// Error is a refused request and the HTTP status to answer it with.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

type agent struct {
	runner string
	remote string
	conn   *websocket.Conn
}

type session struct {
	id       string
	user     string
	runner   string
	remote   string
	cols     uint16
	rows     uint16
	started  time.Time
	deadline time.Time

	runnerConn chan *websocket.Conn
	rejected   chan string
	done       chan struct{}
}

type Hub struct {
	opts    Options
	auth    Authorizer
	log     Logger
	tickets *ticketStore
	now     func() time.Time

	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	agents   map[string]*agent
	sessions map[string]*session
}

func NewHub(opts Options, auth Authorizer, log Logger) *Hub {
	opts.setDefaults()
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		opts:    opts,
		auth:    auth,
		log:     log,
		tickets: newTicketStore(opts.TicketTTL, opts.MaxTicketsPerUser, opts.MaxTickets, time.Now),
		now:     time.Now,

		ctx:    ctx,
		cancel: cancel,

		agents:   make(map[string]*agent),
		sessions: make(map[string]*session),
	}
}

// Close ends every session and control connection.
func (h *Hub) Close() {
	h.cancel()

	h.mu.Lock()
	agents := make([]*agent, 0, len(h.agents))
	for _, a := range h.agents {
		agents = append(agents, a)
	}
	h.mu.Unlock()

	for _, a := range agents {
		_ = a.conn.Close(websocket.StatusGoingAway, "controller shutting down")
	}
}

// Client is the browser as the controller sees it.
type Client struct {
	// IP is the browser's address: the TCP peer, or what a trusted proxy
	// put in X-Forwarded-For.
	IP string
	// HTTPS reports whether the browser reached the controller over HTTPS.
	HTTPS bool
}

// ClientFor describes the browser that sent r. ip must be the client
// address worked out from X-Forwarded-For by trusting only the configured
// proxies, as gin's Context.ClientIP does.
//
// The controller itself serves plain HTTP, so a request counts as HTTPS only
// when a trusted proxy says so in X-Forwarded-Proto. Anyone else can send
// that header.
func (h *Hub) ClientFor(r *http.Request, ip string) Client {
	return Client{IP: ip, HTTPS: r.TLS != nil || h.fromTrustedHTTPSProxy(r)}
}

func (h *Hub) fromTrustedHTTPSProxy(r *http.Request) bool {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	addr := peer.Addr().Unmap()
	if !slices.ContainsFunc(h.opts.TrustedProxies, func(p netip.Prefix) bool { return p.Contains(addr) }) {
		return false
	}

	proto := r.Header.Values("X-Forwarded-Proto")

	return len(proto) == 1 && strings.EqualFold(strings.TrimSpace(proto[0]), "https")
}

// checkClient answers the browser-side checks shared by IssueTicket and
// ServeBrowser.
func (h *Hub) checkClient(client Client) *Error {
	if h.opts.RequireHTTPS && !client.HTTPS {
		return &Error{http.StatusForbidden, "Terminal requires HTTPS; open the controller through its https:// address"}
	}
	if !h.ClientAllowed(client.IP) {
		return &Error{http.StatusForbidden, "Terminal is not allowed from this address"}
	}

	return nil
}

// ClientAllowed reports whether ip is in AllowedClientNets.
func (h *Hub) ClientAllowed(clientIP string) bool {
	if len(h.opts.AllowedClientNets) == 0 {
		return true
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, n := range h.opts.AllowedClientNets {
		if n.Contains(ip) {
			return true
		}
	}

	return false
}

func (h *Hub) userAllowed(user string) bool {
	if user == "" {
		return false
	}

	return len(h.opts.AllowedUsers) == 0 || slices.Contains(h.opts.AllowedUsers, user)
}

// IssueTicket is called by the admin-only REST handler. user comes from the
// caller's JWT, tokenExpires is that JWT's expiry.
func (h *Hub) IssueTicket(user, runner string, client Client, cols, rows uint16, tokenExpires time.Time) (string, time.Duration, *Error) {
	if err := h.checkClient(client); err != nil {
		return "", 0, err
	}
	if !h.userAllowed(user) {
		return "", 0, &Error{http.StatusForbidden, "This account is not allowed to open a terminal"}
	}
	if !h.auth.RunnerExists(runner) {
		return "", 0, &Error{http.StatusNotFound, "Runner not found"}
	}
	if h.agent(runner) == nil {
		return "", 0, &Error{http.StatusConflict, fmt.Sprintf("Runner %s is not connected for terminal (offline, or terminal is not enabled on it)", runner)}
	}
	if err := h.checkLimits(runner); err != nil {
		return "", 0, err
	}

	id, err := h.tickets.issue(ticket{
		user:         user,
		runner:       runner,
		remoteIP:     client.IP,
		cols:         clampSize(cols, 80),
		rows:         clampSize(rows, 24),
		tokenExpires: tokenExpires,
	})
	if err == errTooManyTickets {
		return "", 0, &Error{http.StatusTooManyRequests, "Too many unused terminal tickets; wait a moment and retry"}
	}
	if err != nil {
		return "", 0, &Error{http.StatusInternalServerError, "Failed to create ticket"}
	}

	h.log.Infof("terminal ticket issued: user=%q runner=%q remote=%s", user, runner, client.IP)

	return id, h.opts.TicketTTL, nil
}

func (h *Hub) checkLimits(runner string) *Error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.checkLimitsLocked(runner)
}

func (h *Hub) checkLimitsLocked(runner string) *Error {
	if len(h.sessions) >= h.opts.MaxSessions {
		return &Error{http.StatusTooManyRequests, "Terminal session limit reached on the controller"}
	}

	n := 0
	for _, s := range h.sessions {
		if s.runner == runner {
			n++
		}
	}
	if n >= h.opts.MaxSessionsPerRunner {
		return &Error{http.StatusTooManyRequests, fmt.Sprintf("Terminal session limit reached for runner %s", runner)}
	}

	return nil
}

func (h *Hub) agent(runner string) *agent {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.agents[runner]
}

// ServeBrowser handles the browser's data WebSocket; client comes from
// ClientFor. It checks the ticket before the upgrade, so a bad ticket gets a
// plain 401.
func (h *Hub) ServeBrowser(w http.ResponseWriter, r *http.Request, client Client) {
	if err := h.checkClient(client); err != nil {
		http.Error(w, err.Message, err.Status)
		return
	}
	if r.Header.Get("Origin") == "" {
		http.Error(w, "Origin header is required", http.StatusForbidden)
		return
	}

	protocols := requestedSubprotocols(r)
	if !slices.Contains(protocols, model.TerminalSubprotocol) {
		http.Error(w, "Unsupported terminal protocol", http.StatusBadRequest)
		return
	}

	var ticketID string
	for _, p := range protocols {
		if after, ok := strings.CutPrefix(p, model.TerminalTicketProtocolPrefix); ok {
			ticketID = after
		}
	}

	t, ok := h.tickets.consume(ticketID)
	if !ok || t.remoteIP != client.IP {
		h.log.Warnf("terminal: rejected browser connection with an invalid ticket from %s", client.IP)
		http.Error(w, "Invalid or expired ticket", http.StatusUnauthorized)
		return
	}
	if !h.userAllowed(t.user) || !h.auth.UserIsAdmin(t.user) {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:   []string{model.TerminalSubprotocol},
		OriginPatterns: h.opts.OriginPatterns,
	})
	if err != nil {
		h.log.Warnf("terminal: browser upgrade from %s failed: %v", client.IP, err)
		return
	}
	conn.SetReadLimit(dataReadLimit)

	s, serr := h.startSession(t, client.IP)
	if serr != nil {
		code := websocket.StatusCode(model.TerminalCloseLimitReached)
		if serr.Status == http.StatusInternalServerError {
			code = websocket.StatusInternalError
		}
		closeWithError(conn, code, serr.Message)
		return
	}
	defer h.endSession(s)

	h.log.Infof("terminal session %s opened: user=%q runner=%q remote=%s", s.id, s.user, s.runner, s.remote)

	a := h.agent(s.runner)
	if a == nil {
		h.log.Infof("terminal session %s closed: runner not connected", s.id)
		closeWithError(conn, model.TerminalCloseRunnerGone, "Runner is not connected for terminal")
		return
	}

	open := model.TerminalMessage{
		Type:    model.TerminalMsgOpen,
		Session: s.id,
		Cols:    s.cols,
		Rows:    s.rows,
		User:    s.user,
	}
	if err := writeJSON(h.ctx, a.conn, open); err != nil {
		h.log.Infof("terminal session %s closed: runner control channel failed: %v", s.id, err)
		closeWithError(conn, model.TerminalCloseRunnerGone, "Runner disconnected")
		return
	}

	timer := time.NewTimer(h.opts.OpenTimeout)
	defer timer.Stop()

	select {
	case rc := <-s.runnerConn:
		res := h.bridge(s, conn, rc)
		h.log.Infof("terminal session %s closed: user=%q runner=%q duration=%s in=%dB out=%dB reason=%q",
			s.id, s.user, s.runner, h.now().Sub(s.started).Round(time.Second), res.in, res.out, res.reason)
	case reason := <-s.rejected:
		h.log.Infof("terminal session %s closed: runner refused: %s", s.id, reason)
		closeWithError(conn, model.TerminalCloseForbidden, "Runner refused the session: "+reason)
	case <-timer.C:
		h.log.Infof("terminal session %s closed: runner did not respond", s.id)
		closeWithError(conn, model.TerminalCloseRunnerNoResponse, "Runner did not respond")
	case <-h.ctx.Done():
		closeWithError(conn, websocket.StatusGoingAway, "Controller shutting down")
	}
}

func (h *Hub) startSession(t ticket, remote string) (*session, *Error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, &Error{http.StatusInternalServerError, "Failed to create session"}
	}

	now := h.now()
	deadline := now.Add(h.opts.MaxDuration)
	if !t.tokenExpires.IsZero() && t.tokenExpires.Before(deadline) {
		deadline = t.tokenExpires
	}

	s := &session{
		id:       hex.EncodeToString(idBytes),
		user:     t.user,
		runner:   t.runner,
		remote:   remote,
		cols:     t.cols,
		rows:     t.rows,
		started:  now,
		deadline: deadline,

		runnerConn: make(chan *websocket.Conn, 1),
		rejected:   make(chan string, 1),
		done:       make(chan struct{}),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.checkLimitsLocked(t.runner); err != nil {
		return nil, err
	}
	h.sessions[s.id] = s

	return s, nil
}

func (h *Hub) endSession(s *session) {
	h.mu.Lock()
	delete(h.sessions, s.id)
	h.mu.Unlock()

	close(s.done)
}

func (h *Hub) session(id string) *session {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.sessions[id]
}

// ServeControl handles a runner's control WebSocket. runner and
// registrationID must come from the runner's JWT claims, not from a header;
// remote is the runner's address for the logs.
func (h *Hub) ServeControl(w http.ResponseWriter, r *http.Request, runner, registrationID, remote string) {
	if runner == "" || !h.auth.RunnerExists(runner) {
		http.Error(w, "Runner is not registered", http.StatusForbidden)
		return
	}
	if !h.auth.RunnerTokenCurrent(runner, registrationID) {
		h.log.Warnf("terminal: refused control channel for runner %q from %s: token is not from its current registration", runner, remote)
		http.Error(w, "Runner token is not from this runner's current registration; register the runner again to use the terminal", http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	if err != nil {
		h.log.Warnf("terminal: control upgrade for runner %s failed: %v", runner, err)
		return
	}
	conn.SetReadLimit(controlReadLimit)

	a := &agent{runner: runner, remote: remote, conn: conn}

	h.mu.Lock()
	old := h.agents[runner]
	h.agents[runner] = a
	h.mu.Unlock()

	if old != nil {
		// normal after a network blip, but also what an impostor holding
		// this runner's token would cause
		h.log.Warnf("terminal control channel of runner %q replaced: old remote=%s new remote=%s", runner, old.remote, remote)
		_ = old.conn.Close(websocket.StatusPolicyViolation, "replaced by a newer connection")
	}

	h.log.Infof("terminal control channel connected: runner=%q remote=%s", runner, remote)

	ctx, cancel := context.WithCancel(h.ctx)
	defer func() {
		cancel()
		_ = conn.CloseNow()

		h.mu.Lock()
		if h.agents[runner] == a {
			delete(h.agents, runner)
		}
		h.mu.Unlock()

		h.log.Infof("terminal control channel disconnected: runner=%q remote=%s", runner, remote)
	}()

	go keepAlive(ctx, conn, h.opts.PingInterval, h.opts.PingTimeout)

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			continue
		}

		var msg model.TerminalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case model.TerminalMsgHello:
			h.log.Debugf("terminal: runner %s speaks protocol v%d, max %d sessions", runner, msg.Version, msg.MaxSessions)
		case model.TerminalMsgRejected:
			if s := h.session(msg.Session); s != nil && s.runner == runner {
				select {
				case s.rejected <- msg.Message:
				default:
				}
			}
		}
	}
}

// ServeRunnerSession handles the data WebSocket a runner dials for a session.
// runner and registrationID must come from the runner's JWT claims.
func (h *Hub) ServeRunnerSession(w http.ResponseWriter, r *http.Request, runner, registrationID, id string) {
	if !h.auth.RunnerTokenCurrent(runner, registrationID) {
		http.Error(w, "Runner token is not from this runner's current registration", http.StatusForbidden)
		return
	}

	s := h.session(id)
	if s == nil || s.runner != runner {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	if err != nil {
		h.log.Warnf("terminal: session upgrade for runner %s failed: %v", runner, err)
		return
	}
	conn.SetReadLimit(dataReadLimit)
	defer func() {
		_ = conn.CloseNow()
	}()

	select {
	case s.runnerConn <- conn:
	default:
		_ = conn.Close(websocket.StatusPolicyViolation, "session already attached")
		return
	}

	<-s.done
}

type bridgeResult struct {
	reason      string
	browserCode websocket.StatusCode
	// a peer that stopped answering pings is dropped without a close handshake
	browserDead bool
	runnerDead  bool
	in          int64
	out         int64
}

// bridge copies frames between the browser and the runner until one side
// closes or a limit is hit. Only binary frames (keystrokes) from the browser
// count as activity for the idle timeout.
func (h *Hub) bridge(s *session, browser, runner *websocket.Conn) bridgeResult {
	ctx, cancel := context.WithCancel(h.ctx)
	defer cancel()

	var (
		mu        sync.Mutex
		lastInput = h.now()
		in, out   int64
	)

	// the first result ends the bridge; later ones are dropped
	results := make(chan bridgeResult, 1)
	report := func(r bridgeResult) {
		select {
		case results <- r:
		default:
		}
	}

	_ = writeJSON(ctx, browser, model.TerminalMessage{Type: model.TerminalMsgReady, Session: s.id})

	go func() {
		for {
			typ, data, err := browser.Read(ctx)
			if err != nil {
				report(bridgeResult{reason: "browser closed", browserCode: websocket.StatusNormalClosure})
				return
			}

			switch typ {
			case websocket.MessageBinary:
				mu.Lock()
				lastInput = h.now()
				in += int64(len(data))
				mu.Unlock()

				if err := runner.Write(ctx, websocket.MessageBinary, data); err != nil {
					report(bridgeResult{reason: "runner disconnected", browserCode: model.TerminalCloseRunnerGone})
					return
				}
			case websocket.MessageText:
				var msg model.TerminalMessage
				if err := json.Unmarshal(data, &msg); err != nil || msg.Type != model.TerminalMsgResize {
					continue
				}
				if msg.Cols == 0 || msg.Rows == 0 || msg.Cols > maxTerminalSize || msg.Rows > maxTerminalSize {
					continue
				}

				resize := model.TerminalMessage{Type: model.TerminalMsgResize, Cols: msg.Cols, Rows: msg.Rows}
				if err := writeJSON(ctx, runner, resize); err != nil {
					report(bridgeResult{reason: "runner disconnected", browserCode: model.TerminalCloseRunnerGone})
					return
				}
			}
		}
	}()

	go func() {
		for {
			typ, data, err := runner.Read(ctx)
			if err != nil {
				report(bridgeResult{reason: "runner disconnected", browserCode: model.TerminalCloseRunnerGone})
				return
			}

			switch typ {
			case websocket.MessageBinary:
				mu.Lock()
				out += int64(len(data))
				mu.Unlock()

				if err := browser.Write(ctx, websocket.MessageBinary, data); err != nil {
					report(bridgeResult{reason: "browser closed", browserCode: websocket.StatusNormalClosure})
					return
				}
			case websocket.MessageText:
				var msg model.TerminalMessage
				if err := json.Unmarshal(data, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case model.TerminalMsgExit:
					_ = writeJSON(ctx, browser, model.TerminalMessage{Type: model.TerminalMsgExit, Code: msg.Code})
					report(bridgeResult{reason: "shell exited", browserCode: websocket.StatusNormalClosure})
					return
				case model.TerminalMsgError:
					_ = writeJSON(ctx, browser, model.TerminalMessage{Type: model.TerminalMsgError, Message: msg.Message})
				}
			}
		}
	}()

	check := time.NewTicker(h.opts.CheckInterval)
	defer check.Stop()
	ping := time.NewTicker(h.opts.PingInterval)
	defer ping.Stop()
	lastRecheck := h.now()

	var res bridgeResult
loop:
	for {
		select {
		case res = <-results:
			break loop
		case <-h.ctx.Done():
			res = bridgeResult{reason: "controller shutting down", browserCode: websocket.StatusGoingAway}
			break loop
		case <-ping.C:
			go func() {
				if err := pingOnce(ctx, browser, h.opts.PingTimeout); err != nil && ctx.Err() == nil {
					report(bridgeResult{reason: "browser stopped responding", browserCode: websocket.StatusGoingAway, browserDead: true})
				}
			}()
			go func() {
				if err := pingOnce(ctx, runner, h.opts.PingTimeout); err != nil && ctx.Err() == nil {
					report(bridgeResult{reason: "runner stopped responding", browserCode: model.TerminalCloseRunnerGone, runnerDead: true})
				}
			}()
		case <-check.C:
			now := h.now()

			mu.Lock()
			idle := now.Sub(lastInput)
			mu.Unlock()

			if idle >= h.opts.IdleTimeout {
				res = bridgeResult{reason: "idle timeout", browserCode: model.TerminalCloseTimeout}
				break loop
			}
			if !now.Before(s.deadline) {
				res = bridgeResult{reason: "session time limit", browserCode: model.TerminalCloseTimeout}
				break loop
			}
			if now.Sub(lastRecheck) >= h.opts.RecheckInterval {
				lastRecheck = now
				if !h.auth.UserIsAdmin(s.user) {
					res = bridgeResult{reason: "admin access revoked", browserCode: model.TerminalCloseForbidden}
					break loop
				}
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if res.browserDead {
			_ = browser.CloseNow()
			return
		}
		_ = browser.Close(res.browserCode, closeReason(res.reason))
	}()
	go func() {
		defer wg.Done()
		if res.runnerDead {
			_ = runner.CloseNow()
			return
		}
		_ = runner.Close(websocket.StatusNormalClosure, "session closed")
	}()
	wg.Wait()

	mu.Lock()
	res.in, res.out = in, out
	mu.Unlock()

	return res
}

func keepAlive(ctx context.Context, conn *websocket.Conn, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := pingOnce(ctx, conn, timeout); err != nil {
				_ = conn.CloseNow()
				return
			}
		}
	}
}

func pingOnce(ctx context.Context, conn *websocket.Conn, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return conn.Ping(ctx)
}

func writeJSON(ctx context.Context, conn *websocket.Conn, msg model.TerminalMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return conn.Write(ctx, websocket.MessageText, data)
}

// closeWithError tells the browser why, both as a message (shown in the
// terminal window) and as the close reason.
func closeWithError(conn *websocket.Conn, code websocket.StatusCode, message string) {
	_ = writeJSON(context.Background(), conn, model.TerminalMessage{Type: model.TerminalMsgError, Message: message})
	_ = conn.Close(code, closeReason(message))
}

// closeReason keeps a close reason within the 123 bytes a close frame allows.
func closeReason(reason string) string {
	if len(reason) > 120 {
		return reason[:120]
	}

	return reason
}

func requestedSubprotocols(r *http.Request) []string {
	var protocols []string
	for _, v := range r.Header.Values("Sec-WebSocket-Protocol") {
		for p := range strings.SplitSeq(v, ",") {
			if p = strings.TrimSpace(p); p != "" {
				protocols = append(protocols, p)
			}
		}
	}

	return protocols
}

func clampSize(v, fallback uint16) uint16 {
	if v == 0 || v > maxTerminalSize {
		return fallback
	}

	return v
}

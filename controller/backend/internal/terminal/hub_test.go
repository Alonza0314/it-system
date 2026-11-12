package terminal

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/coder/websocket"
)

type nopLogger struct{}

func (nopLogger) Infof(string, ...interface{})  {}
func (nopLogger) Warnf(string, ...interface{})  {}
func (nopLogger) Errorf(string, ...interface{}) {}
func (nopLogger) Debugf(string, ...interface{}) {}

type fakeAuth struct {
	mu      sync.Mutex
	runners map[string]bool
	admins  map[string]bool
}

func newFakeAuth() *fakeAuth {
	return &fakeAuth{
		runners: map[string]bool{"runner-1": true, "runner-2": true},
		admins:  map[string]bool{"admin": true},
	}
}

func (a *fakeAuth) RunnerExists(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runners[name]
}

// RunnerTokenCurrent accepts "rid-<runner>" as the current registration.
func (a *fakeAuth) RunnerTokenCurrent(name, registrationID string) bool {
	return a.RunnerExists(name) && registrationID == "rid-"+name
}

func (a *fakeAuth) UserIsAdmin(user string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.admins[user]
}

func (a *fakeAuth) setAdmin(user string, admin bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.admins[user] = admin
}

type testEnv struct {
	t    *testing.T
	hub  *Hub
	auth *fakeAuth
	srv  *httptest.Server
	ws   string
}

// newTestEnv serves the hub like the controller does, except that the runner
// name comes from a test header instead of the runner JWT.
func newTestEnv(t *testing.T, opts Options) *testEnv {
	t.Helper()

	auth := newFakeAuth()
	hub := NewHub(opts, auth, nopLogger{})

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeBrowser(w, r, hub.ClientFor(r, remoteIP(r.RemoteAddr)))
	})
	mux.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeControl(w, r, r.Header.Get("X-Test-Runner"), r.Header.Get("X-Test-Rid"), remoteIP(r.RemoteAddr))
	})
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeRunnerSession(w, r, r.Header.Get("X-Test-Runner"), r.Header.Get("X-Test-Rid"), r.URL.Query().Get("id"))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		hub.Close()
		srv.Close()
	})

	return &testEnv{t: t, hub: hub, auth: auth, srv: srv, ws: "ws" + strings.TrimPrefix(srv.URL, "http")}
}

type runnerBehavior int

const (
	runnerEcho runnerBehavior = iota
	runnerReject
	runnerIgnore
)

type fakeRunner struct {
	resizes chan model.TerminalMessage
	opens   chan model.TerminalMessage
	// stop drops the runner's control and data connections
	stop context.CancelFunc
}

// startRunner connects a fake runner. With runnerEcho it dials back every
// session, echoes input and answers "exit\n" with an exit message.
func (e *testEnv) startRunner(name string, behavior runnerBehavior) *fakeRunner {
	e.t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	e.t.Cleanup(cancel)

	header := http.Header{"X-Test-Runner": {name}, "X-Test-Rid": {"rid-" + name}}
	conn, _, err := websocket.Dial(ctx, e.ws+"/control", &websocket.DialOptions{
		HTTPHeader:   header,
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	if err != nil {
		e.t.Fatalf("control dial: %v", err)
	}
	e.t.Cleanup(func() { _ = conn.CloseNow() })

	fr := &fakeRunner{
		resizes: make(chan model.TerminalMessage, 10),
		opens:   make(chan model.TerminalMessage, 10),
		stop:    cancel,
	}

	go func() {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var msg model.TerminalMessage
			if json.Unmarshal(data, &msg) != nil || msg.Type != model.TerminalMsgOpen {
				continue
			}
			fr.opens <- msg

			switch behavior {
			case runnerReject:
				_ = writeJSON(ctx, conn, model.TerminalMessage{Type: model.TerminalMsgRejected, Session: msg.Session, Message: "session limit reached on runner"})
			case runnerEcho:
				go fr.echoSession(ctx, e.ws, header, msg.Session)
			}
		}
	}()

	waitFor(e.t, func() bool { return e.hub.agent(name) != nil })

	return fr
}

func (fr *fakeRunner) echoSession(ctx context.Context, ws string, header http.Header, id string) {
	conn, _, err := websocket.Dial(ctx, ws+"/session?id="+id, &websocket.DialOptions{
		HTTPHeader:   header,
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}

		if typ == websocket.MessageText {
			var msg model.TerminalMessage
			if json.Unmarshal(data, &msg) == nil && msg.Type == model.TerminalMsgResize {
				fr.resizes <- msg
			}
			continue
		}

		if string(data) == "exit\n" {
			_ = writeJSON(ctx, conn, model.TerminalMessage{Type: model.TerminalMsgExit, Code: 3})
			continue
		}
		_ = conn.Write(ctx, websocket.MessageBinary, append([]byte("echo:"), data...))
	}
}

func (e *testEnv) ticket(user, runner string, tokenExpires time.Time) string {
	e.t.Helper()

	id, _, err := e.hub.IssueTicket(user, runner, localClient, 100, 30, tokenExpires)
	if err != nil {
		e.t.Fatalf("IssueTicket: %v", err.Message)
	}

	return id
}

func (e *testEnv) dialBrowser(ticketID string, header http.Header) (*websocket.Conn, *http.Response, error) {
	if header == nil {
		header = http.Header{"Origin": {e.srv.URL}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return websocket.Dial(ctx, e.ws+"/ws", &websocket.DialOptions{
		HTTPHeader:   header,
		Subprotocols: []string{model.TerminalSubprotocol, model.TerminalTicketProtocolPrefix + ticketID},
	})
}

func (e *testEnv) openBrowser(ticketID string) *websocket.Conn {
	e.t.Helper()

	conn, _, err := e.dialBrowser(ticketID, nil)
	if err != nil {
		e.t.Fatalf("browser dial: %v", err)
	}
	e.t.Cleanup(func() { _ = conn.CloseNow() })

	return conn
}

func readText(t *testing.T, conn *websocket.Conn) model.TerminalMessage {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if typ != websocket.MessageText {
			continue
		}

		var msg model.TerminalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("bad message %q: %v", data, err)
		}
		return msg
	}
}

// readUntilClosed drains the connection and returns its close status.
func readUntilClosed(t *testing.T, conn *websocket.Conn) (websocket.StatusCode, []model.TerminalMessage) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msgs []model.TerminalMessage
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return websocket.CloseStatus(err), msgs
		}
		if typ == websocket.MessageText {
			var msg model.TerminalMessage
			if json.Unmarshal(data, &msg) == nil {
				msgs = append(msgs, msg)
			}
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTicketStoreOneTimeAndExpiry(t *testing.T) {
	now := time.Now()
	s := newTicketStore(30*time.Second, 10, 100, func() time.Time { return now })

	id, err := s.issue(ticket{user: "admin", runner: "runner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 43 {
		t.Fatalf("ticket id length = %d, want 43", len(id))
	}

	if got, ok := s.consume(id); !ok || got.user != "admin" || got.runner != "runner-1" {
		t.Fatalf("first consume = %+v, %v", got, ok)
	}
	if _, ok := s.consume(id); ok {
		t.Fatal("ticket worked twice")
	}

	id, _ = s.issue(ticket{user: "admin", runner: "runner-1"})
	now = now.Add(30 * time.Second)
	if _, ok := s.consume(id); ok {
		t.Fatal("expired ticket worked")
	}

	if _, ok := s.consume(""); ok {
		t.Fatal("empty ticket worked")
	}
}

func TestClientChecks(t *testing.T) {
	_, vpn, _ := net.ParseCIDR("10.8.0.0/16")
	h := NewHub(Options{AllowedClientNets: []*net.IPNet{vpn}}, newFakeAuth(), nopLogger{})
	defer h.Close()

	if !h.ClientAllowed("10.8.1.4") {
		t.Error("VPN address refused")
	}
	if h.ClientAllowed("140.113.1.1") || h.ClientAllowed("") {
		t.Error("public or empty address allowed")
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://example")
	rec := httptest.NewRecorder()
	h.ServeBrowser(rec, req, Client{IP: "140.113.1.1"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("ServeBrowser from a public address: status %d, want 403", rec.Code)
	}
	if _, _, err := h.IssueTicket("admin", "runner-1", Client{IP: "140.113.1.1"}, 0, 0, time.Time{}); err == nil || err.Status != http.StatusForbidden {
		t.Errorf("IssueTicket from a public address = %v, want 403", err)
	}

	secure := NewHub(Options{RequireHTTPS: true}, newFakeAuth(), nopLogger{})
	defer secure.Close()

	if _, _, err := secure.IssueTicket("admin", "runner-1", Client{IP: "10.8.1.4"}, 0, 0, time.Time{}); err == nil || !strings.Contains(err.Message, "HTTPS") {
		t.Errorf("IssueTicket over plain HTTP = %v, want an HTTPS error", err)
	}
	// over HTTPS it gets past the client checks to the runner lookup
	if _, _, err := secure.IssueTicket("admin", "no-such-runner", Client{IP: "10.8.1.4", HTTPS: true}, 0, 0, time.Time{}); err == nil || err.Status != http.StatusNotFound {
		t.Errorf("IssueTicket over HTTPS = %v, want 404 for the unknown runner", err)
	}
}

func TestClientForTrustsForwardedProtoOnlyFromProxies(t *testing.T) {
	h := NewHub(Options{TrustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}}, newFakeAuth(), nopLogger{})
	defer h.Close()

	cases := []struct {
		name   string
		remote string
		proto  []string
		want   bool
	}{
		{"proxy says https", "127.0.0.1:5000", []string{"https"}, true},
		{"proxy, IPv4-mapped peer", "[::ffff:127.0.0.1]:5000", []string{"HTTPS"}, true},
		{"proxy says http", "127.0.0.1:5000", []string{"http"}, false},
		{"proxy without the header", "127.0.0.1:5000", nil, false},
		{"proxy passes two values", "127.0.0.1:5000", []string{"https", "http"}, false},
		{"anyone else says https", "140.113.1.1:5000", []string{"https"}, false},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.RemoteAddr = c.remote
		for _, v := range c.proto {
			req.Header.Add("X-Forwarded-Proto", v)
		}
		if got := h.ClientFor(req, "10.8.1.4"); got.HTTPS != c.want || got.IP != "10.8.1.4" {
			t.Errorf("%s: ClientFor = %+v, want HTTPS %v", c.name, got, c.want)
		}
	}
}

func TestBrowserNeedsHTTPSWhenRequired(t *testing.T) {
	e := newTestEnv(t, Options{RequireHTTPS: true, TrustedProxies: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}})
	e.startRunner("runner-1", runnerEcho)

	id, _, err := e.hub.IssueTicket("admin", "runner-1", Client{IP: "127.0.0.1", HTTPS: true}, 0, 0, time.Time{})
	if err != nil {
		t.Fatal(err.Message)
	}

	_, resp, dialErr := e.dialBrowser(id, nil)
	if dialErr == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("plain HTTP browser: err=%v resp=%v, want 403", dialErr, resp)
	}

	browser, _, dialErr := e.dialBrowser(id, http.Header{"Origin": {e.srv.URL}, "X-Forwarded-Proto": {"https"}})
	if dialErr != nil {
		t.Fatalf("browser through the HTTPS proxy: %v", dialErr)
	}
	defer func() { _ = browser.CloseNow() }()
	if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
		t.Errorf("first message = %+v, want ready", msg)
	}
}

func TestIssueTicketErrors(t *testing.T) {
	e := newTestEnv(t, Options{AllowedUsers: []string{"admin"}, MaxSessionsPerRunner: 1})

	cases := []struct {
		name   string
		user   string
		runner string
		want   int
	}{
		{"user not in allowedUsers", "tenant-admin", "runner-1", http.StatusForbidden},
		{"unknown runner", "admin", "runner-9", http.StatusNotFound},
		{"runner not connected", "admin", "runner-1", http.StatusConflict},
	}
	for _, c := range cases {
		_, _, err := e.hub.IssueTicket(c.user, c.runner, localClient, 0, 0, time.Time{})
		if err == nil || err.Status != c.want {
			t.Errorf("%s: got %v, want %d", c.name, err, c.want)
		}
	}

	e.startRunner("runner-1", runnerEcho)
	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
		t.Fatalf("first message = %+v, want ready", msg)
	}

	if _, _, err := e.hub.IssueTicket("admin", "runner-1", localClient, 0, 0, time.Time{}); err == nil || err.Status != http.StatusTooManyRequests {
		t.Errorf("over the per-runner limit: got %v, want 429", err)
	}
}

func TestBrowserHandshakeRejections(t *testing.T) {
	e := newTestEnv(t, Options{})
	e.startRunner("runner-1", runnerEcho)

	status := func(ticketID string, header http.Header) int {
		conn, resp, err := e.dialBrowser(ticketID, header)
		if err == nil {
			_ = conn.CloseNow()
			return http.StatusSwitchingProtocols
		}
		if resp == nil {
			t.Fatalf("dial failed without a response: %v", err)
		}
		return resp.StatusCode
	}

	if got := status("", nil); got != http.StatusUnauthorized {
		t.Errorf("no ticket: %d, want 401", got)
	}
	if got := status("not-a-ticket", nil); got != http.StatusUnauthorized {
		t.Errorf("unknown ticket: %d, want 401", got)
	}

	used := e.ticket("admin", "runner-1", time.Time{})
	if got := status(used, nil); got != http.StatusSwitchingProtocols {
		t.Fatalf("valid ticket: %d, want 101", got)
	}
	if got := status(used, nil); got != http.StatusUnauthorized {
		t.Errorf("reused ticket: %d, want 401", got)
	}

	if got := status(e.ticket("admin", "runner-1", time.Time{}), http.Header{}); got != http.StatusForbidden {
		t.Errorf("no Origin: %d, want 403", got)
	}
	if got := status(e.ticket("admin", "runner-1", time.Time{}), http.Header{"Origin": {"http://evil.example"}}); got != http.StatusForbidden {
		t.Errorf("foreign Origin: %d, want 403", got)
	}

	revoked := e.ticket("admin", "runner-1", time.Time{})
	e.auth.setAdmin("admin", false)
	if got := status(revoked, nil); got != http.StatusForbidden {
		t.Errorf("no longer admin: %d, want 403", got)
	}
}

func TestBrowserTicketBoundToAddress(t *testing.T) {
	e := newTestEnv(t, Options{})
	e.startRunner("runner-1", runnerEcho)

	id, _, err := e.hub.IssueTicket("admin", "runner-1", Client{IP: "10.8.1.4"}, 0, 0, time.Time{})
	if err != nil {
		t.Fatal(err.Message)
	}

	_, resp, dialErr := e.dialBrowser(id, nil)
	if dialErr == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ticket used from another address: err=%v resp=%v, want 401", dialErr, resp)
	}
}

func TestRelayResizeAndExit(t *testing.T) {
	e := newTestEnv(t, Options{})
	fr := e.startRunner("runner-1", runnerEcho)

	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))

	open := <-fr.opens
	if open.User != "admin" || open.Cols != 100 || open.Rows != 30 || len(open.Session) != 32 {
		t.Fatalf("open message = %+v", open)
	}
	if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
		t.Fatalf("first message = %+v, want ready", msg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := browser.Write(ctx, websocket.MessageBinary, []byte("hostname\n")); err != nil {
		t.Fatal(err)
	}
	typ, data, err := browser.Read(ctx)
	if err != nil || typ != websocket.MessageBinary || string(data) != "echo:hostname\n" {
		t.Fatalf("echo = %v %q %v", typ, data, err)
	}

	// invalid sizes are dropped, valid ones are forwarded
	_ = browser.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":0,"rows":10}`))
	_ = browser.Write(ctx, websocket.MessageText, []byte(`{"type":"open","session":"x"}`))
	_ = browser.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":120,"rows":40}`))
	select {
	case r := <-fr.resizes:
		if r.Cols != 120 || r.Rows != 40 {
			t.Fatalf("resize = %+v, want 120x40", r)
		}
	case <-ctx.Done():
		t.Fatal("resize not forwarded")
	}

	if err := browser.Write(ctx, websocket.MessageBinary, []byte("exit\n")); err != nil {
		t.Fatal(err)
	}
	code, msgs := readUntilClosed(t, browser)
	if code != websocket.StatusNormalClosure {
		t.Errorf("close code = %d, want 1000", code)
	}
	if len(msgs) == 0 || msgs[0].Type != model.TerminalMsgExit || msgs[0].Code != 3 {
		t.Errorf("messages before close = %+v, want exit code 3", msgs)
	}

	waitFor(t, func() bool { return e.hub.session(open.Session) == nil })
}

func TestRunnerCannotAttachAnotherRunnersSession(t *testing.T) {
	e := newTestEnv(t, Options{OpenTimeout: 300 * time.Millisecond})
	fr := e.startRunner("runner-1", runnerIgnore)

	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	open := <-fr.opens

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, e.ws+"/session?id="+open.Session, &websocket.DialOptions{
		HTTPHeader: http.Header{"X-Test-Runner": {"runner-2"}, "X-Test-Rid": {"rid-runner-2"}},
	})
	if err == nil || resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("runner-2 attaching runner-1's session: err=%v resp=%v, want 404", err, resp)
	}

	code, _ := readUntilClosed(t, browser)
	if code != model.TerminalCloseRunnerNoResponse {
		t.Errorf("close code = %d, want %d", code, model.TerminalCloseRunnerNoResponse)
	}
}

func TestUnregisteredRunnerControlRefused(t *testing.T) {
	e := newTestEnv(t, Options{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, c := range []struct{ name, rid string }{
		{"", ""},
		{"deleted-runner", "rid-deleted-runner"},
		// a token from before the runner was registered again
		{"runner-1", "rid-old"},
		// a token from before registration ids existed
		{"runner-1", ""},
	} {
		_, resp, err := websocket.Dial(ctx, e.ws+"/control", &websocket.DialOptions{
			HTTPHeader: http.Header{"X-Test-Runner": {c.name}, "X-Test-Rid": {c.rid}},
		})
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Errorf("runner %q rid %q: err=%v resp=%v, want 403", c.name, c.rid, err, resp)
		}
	}
	if e.hub.agent("runner-1") != nil {
		t.Error("a stale token registered a control channel")
	}
}

func TestNewerControlConnectionReplacesOlder(t *testing.T) {
	e := newTestEnv(t, Options{})

	e.startRunner("runner-1", runnerIgnore)
	first := e.hub.agent("runner-1")
	e.startRunner("runner-1", runnerEcho)
	waitFor(t, func() bool { return e.hub.agent("runner-1") != first })

	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
		t.Fatalf("first message = %+v, want ready from the newer runner connection", msg)
	}
}

func TestRunnerRejectsSession(t *testing.T) {
	e := newTestEnv(t, Options{})
	e.startRunner("runner-1", runnerReject)

	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	code, msgs := readUntilClosed(t, browser)
	if code != model.TerminalCloseForbidden {
		t.Errorf("close code = %d, want %d", code, model.TerminalCloseForbidden)
	}
	if len(msgs) == 0 || msgs[0].Type != model.TerminalMsgError || !strings.Contains(msgs[0].Message, "session limit reached on runner") {
		t.Errorf("messages = %+v, want the runner's reason", msgs)
	}
}

func TestSessionEndsOn(t *testing.T) {
	fast := Options{IdleTimeout: 300 * time.Millisecond, CheckInterval: 50 * time.Millisecond, RecheckInterval: 50 * time.Millisecond}

	t.Run("idle timeout", func(t *testing.T) {
		e := newTestEnv(t, fast)
		e.startRunner("runner-1", runnerEcho)
		browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))

		// resizes are not activity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		go func() {
			for range 10 {
				_ = browser.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":90,"rows":20}`))
				time.Sleep(50 * time.Millisecond)
			}
		}()

		code, _ := readUntilClosed(t, browser)
		if code != model.TerminalCloseTimeout {
			t.Errorf("close code = %d, want %d", code, model.TerminalCloseTimeout)
		}
	})

	t.Run("token expiry", func(t *testing.T) {
		opts := fast
		opts.IdleTimeout = time.Hour
		e := newTestEnv(t, opts)
		e.startRunner("runner-1", runnerEcho)
		browser := e.openBrowser(e.ticket("admin", "runner-1", time.Now().Add(300*time.Millisecond)))

		code, _ := readUntilClosed(t, browser)
		if code != model.TerminalCloseTimeout {
			t.Errorf("close code = %d, want %d", code, model.TerminalCloseTimeout)
		}
	})

	t.Run("admin revoked", func(t *testing.T) {
		opts := fast
		opts.IdleTimeout = time.Hour
		e := newTestEnv(t, opts)
		e.startRunner("runner-1", runnerEcho)
		browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
		if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
			t.Fatalf("first message = %+v, want ready", msg)
		}

		e.auth.setAdmin("admin", false)
		code, _ := readUntilClosed(t, browser)
		if code != model.TerminalCloseForbidden {
			t.Errorf("close code = %d, want %d", code, model.TerminalCloseForbidden)
		}
	})

	t.Run("runner gone", func(t *testing.T) {
		e := newTestEnv(t, Options{})
		fr := e.startRunner("runner-1", runnerEcho)
		browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
		if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
			t.Fatalf("first message = %+v, want ready", msg)
		}

		fr.stop()

		code, _ := readUntilClosed(t, browser)
		if code != model.TerminalCloseRunnerGone {
			t.Errorf("close code = %d, want %d", code, model.TerminalCloseRunnerGone)
		}
		waitFor(t, func() bool { return e.hub.agent("runner-1") == nil })
		if _, _, err := e.hub.IssueTicket("admin", "runner-1", localClient, 0, 0, time.Time{}); err == nil || err.Status != http.StatusConflict {
			t.Errorf("ticket for a disconnected runner: got %v, want 409", err)
		}
	})
}

func TestStaleRunnerTokenCannotAttachSession(t *testing.T) {
	e := newTestEnv(t, Options{OpenTimeout: 300 * time.Millisecond})
	fr := e.startRunner("runner-1", runnerIgnore)

	e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	open := <-fr.opens

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, e.ws+"/session?id="+open.Session, &websocket.DialOptions{
		HTTPHeader: http.Header{"X-Test-Runner": {"runner-1"}, "X-Test-Rid": {"rid-old"}},
	})
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("stale token attaching a session: err=%v resp=%v, want 403", err, resp)
	}
}

func TestTicketLimits(t *testing.T) {
	e := newTestEnv(t, Options{MaxTicketsPerUser: 2, MaxTickets: 3})
	e.startRunner("runner-1", runnerEcho)
	e.auth.setAdmin("other", true)

	issue := func(user string) *Error {
		_, _, err := e.hub.IssueTicket(user, "runner-1", localClient, 0, 0, time.Time{})
		return err
	}

	for i := range 2 {
		if err := issue("admin"); err != nil {
			t.Fatalf("ticket %d: %v", i+1, err.Message)
		}
	}
	if err := issue("admin"); err == nil || err.Status != http.StatusTooManyRequests {
		t.Errorf("third unused ticket of one user: %v, want 429", err)
	}
	if err := issue("other"); err != nil {
		t.Errorf("another user's ticket: %v", err.Message)
	}
	if err := issue("other"); err == nil || err.Status != http.StatusTooManyRequests {
		t.Errorf("over the global limit: %v, want 429", err)
	}
}

func TestBridgeEndsWhenBrowserStopsAnsweringPings(t *testing.T) {
	e := newTestEnv(t, Options{PingInterval: 100 * time.Millisecond, PingTimeout: 200 * time.Millisecond})
	fr := e.startRunner("runner-1", runnerEcho)

	browser := e.openBrowser(e.ticket("admin", "runner-1", time.Time{}))
	open := <-fr.opens
	if msg := readText(t, browser); msg.Type != model.TerminalMsgReady {
		t.Fatalf("first message = %+v, want ready", msg)
	}

	// a browser that stops reading (a laptop gone to sleep) never answers a ping
	waitFor(t, func() bool { return e.hub.session(open.Session) == nil })
}

func TestCloseReasonFitsFrame(t *testing.T) {
	if got := closeReason(strings.Repeat("x", 200)); len(got) > 123 {
		t.Errorf("close reason is %d bytes", len(got))
	}
}

var localClient = Client{IP: "127.0.0.1"}

func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}

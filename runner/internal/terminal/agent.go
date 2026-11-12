// Package terminal lets admins open a shell on this runner from the
// controller's Runner page. The runner keeps one outbound control WebSocket
// to the controller; for each session the controller asks for, the runner
// dials a data WebSocket and attaches it to a new PTY. The runner never
// listens on a port. See design/runner-terminal.md.
package terminal

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/config"
	"github.com/Alonza0314/it-system/runner/constant"

	"github.com/coder/websocket"
)

const (
	controlReadLimit = 4 << 10
	dataReadLimit    = 64 << 10

	maxBackoff   = 30 * time.Second
	pingInterval = 30 * time.Second
	stopTimeout  = 15 * time.Second
)

type Logger interface {
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
}

type Agent struct {
	runnerName string
	token      string

	controlURL string
	sessionURL string

	shell *shellSpec

	maxSessions   int
	idleTimeout   time.Duration
	maxDuration   time.Duration
	checkInterval time.Duration
	procPath      string

	log Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	active int
}

// NewAgent checks the terminal configuration; an error means the runner
// should not start.
// controllerURL is the controller's base URL, such as https://it.example.org;
// with https the WebSocket connections use TLS (wss).
func NewAgent(runnerName, controllerURL, token string, cfg config.TerminalIE, log Logger) (*Agent, error) {
	nice := 10
	if cfg.Nice != nil {
		nice = *cfg.Nice
	}

	shell, err := resolveShell(cfg.User, cfg.Shell, nice)
	if err != nil {
		return nil, err
	}

	a := &Agent{
		runnerName: runnerName,
		token:      strings.TrimSpace(token),

		controlURL: controllerURL + constant.API_RUNNER_TERMINAL_CONTROL,
		sessionURL: controllerURL + constant.API_RUNNER_TERMINAL_SESSION,

		shell: shell,

		maxSessions:   cfg.MaxSessions,
		idleTimeout:   cfg.IdleTimeout,
		maxDuration:   cfg.MaxDuration,
		checkInterval: 5 * time.Second,
		procPath:      "/proc",

		log: log,
	}
	if a.maxSessions <= 0 {
		a.maxSessions = 2
	}
	if a.idleTimeout <= 0 {
		a.idleTimeout = 15 * time.Minute
	}
	if a.maxDuration <= 0 {
		a.maxDuration = 2 * time.Hour
	}
	a.ctx, a.cancel = context.WithCancel(context.Background())

	return a, nil
}

// ShellUser is the account terminal sessions run as.
func (a *Agent) ShellUser() string {
	return a.shell.user
}

func (a *Agent) Start() {
	a.wg.Add(1)
	go a.controlLoop()
}

// Stop closes the control channel and ends every session. Ending a session
// is bounded, but Stop does not wait longer than stopTimeout in any case.
func (a *Agent) Stop() {
	a.cancel()

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(stopTimeout):
		a.log.Warnf("Terminal: sessions did not end within %s", stopTimeout)
	}
}

func (a *Agent) header() http.Header {
	return http.Header{
		constant.AUTHENTICATION_HEADER_KEY: {fmt.Sprintf(constant.AUTHENTICATION_HEADER_VALUE, a.token)},
		constant.USER_HEADER_KEY:           {a.runnerName},
	}
}

func (a *Agent) controlLoop() {
	defer a.wg.Done()

	backoff := time.Second
	for {
		connected, err := a.runControl()
		if a.ctx.Err() != nil {
			return
		}
		if connected {
			backoff = time.Second
		}

		wait := backoff/2 + rand.N(backoff/2+1)
		a.log.Warnf("Terminal control channel: %v; reconnecting in %s", err, wait.Round(100*time.Millisecond))

		select {
		case <-a.ctx.Done():
			return
		case <-time.After(wait):
		}

		backoff = min(backoff*2, maxBackoff)
	}
}

// runControl keeps one control connection until it fails. connected reports
// whether the dial succeeded.
func (a *Agent) runControl() (connected bool, err error) {
	dialCtx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	conn, resp, err := websocket.Dial(dialCtx, a.controlURL, &websocket.DialOptions{
		HTTPHeader:   a.header(),
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	cancel()
	if err != nil {
		if resp != nil {
			return false, fmt.Errorf("controller answered %s", resp.Status)
		}
		return false, err
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(controlReadLimit)

	a.log.Infof("Terminal control channel connected to %s", a.controlURL)

	ctx, stop := context.WithCancel(a.ctx)
	defer stop()

	if err := writeJSON(ctx, conn, model.TerminalMessage{
		Type:        model.TerminalMsgHello,
		Version:     model.TerminalProtocolVersion,
		MaxSessions: a.maxSessions,
	}); err != nil {
		return true, err
	}

	go keepAlive(ctx, conn)

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return true, err
		}
		if typ != websocket.MessageText {
			continue
		}

		var msg model.TerminalMessage
		if err := json.Unmarshal(data, &msg); err != nil || msg.Type != model.TerminalMsgOpen {
			continue
		}
		if !validSessionID(msg.Session) {
			a.log.Warnf("Terminal: ignored an open request with a malformed session id")
			continue
		}

		if !a.reserve() {
			a.log.Warnf("Terminal: refused session %s from %s: limit of %d sessions reached", msg.Session, msg.User, a.maxSessions)
			_ = writeJSON(ctx, conn, model.TerminalMessage{
				Type:    model.TerminalMsgRejected,
				Session: msg.Session,
				Message: "session limit reached on runner",
			})
			continue
		}

		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			defer a.release()
			a.runSession(msg)
		}()
	}
}

func (a *Agent) reserve() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.active >= a.maxSessions {
		return false
	}
	a.active++

	return true
}

func (a *Agent) release() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.active--
}

func (a *Agent) dialSession(id string) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	conn, resp, err := websocket.Dial(ctx, a.sessionURL+"?id="+url.QueryEscape(id), &websocket.DialOptions{
		HTTPHeader:   a.header(),
		Subprotocols: []string{model.TerminalSubprotocol},
	})
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("controller answered %s", resp.Status)
		}
		return nil, err
	}
	conn.SetReadLimit(dataReadLimit)

	return conn, nil
}

func validSessionID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)

	return err == nil
}

func keepAlive(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pctx)
			cancel()
			if err != nil {
				_ = conn.CloseNow()
				return
			}
		}
	}
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

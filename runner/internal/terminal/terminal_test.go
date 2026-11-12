package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/config"

	"github.com/coder/websocket"
)

type nopLogger struct{}

func (nopLogger) Infof(string, ...interface{})  {}
func (nopLogger) Warnf(string, ...interface{})  {}
func (nopLogger) Errorf(string, ...interface{}) {}
func (nopLogger) Debugf(string, ...interface{}) {}

func currentUser(t *testing.T) string {
	t.Helper()

	u, err := user.Current()
	if err != nil {
		t.Skipf("no current user: %v", err)
	}

	return u.Username
}

func testShell(t *testing.T) *shellSpec {
	t.Helper()

	s, err := resolveShell(currentUser(t), "/bin/sh", 0)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

// readUntil reads from the PTY master until re matches the output so far.
func readUntil(t *testing.T, f *os.File, re *regexp.Regexp) []string {
	t.Helper()

	var out bytes.Buffer
	buf := make([]byte, 4096)
	deadline := time.Now().Add(5 * time.Second)
	for {
		_ = f.SetReadDeadline(deadline)
		n, err := f.Read(buf)
		out.Write(buf[:n])
		if m := re.FindStringSubmatch(out.String()); m != nil {
			return m
		}
		if err != nil {
			t.Fatalf("waiting for %q: %v; output so far:\n%s", re, err, out.String())
		}
	}
}

// processesWith lists live processes whose command line contains marker.
func processesWith(marker string) []int {
	entries, _ := os.ReadDir("/proc")
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil || !strings.Contains(string(bytes.ReplaceAll(cmdline, []byte{0}, []byte{' '})), marker) {
			continue
		}
		if state, _ := processState("/proc", pid); state != "" && state != "Z" {
			pids = append(pids, pid)
		}
	}

	return pids
}

func TestResolveShell(t *testing.T) {
	if _, err := resolveShell("", "", 0); err == nil {
		t.Error("an empty terminal.user is accepted")
	}
	if _, err := resolveShell("no-such-user-it-system", "", 0); err == nil {
		t.Error("an unknown terminal.user is accepted")
	}
	if _, err := resolveShell(currentUser(t), "sh", 0); err == nil {
		t.Error("a relative terminal.shell is accepted")
	}

	// an account whose login shell is nologin needs an explicit shell
	if shell := loginShell("/etc/passwd", "nobody"); isNoLoginShell(shell) {
		if _, err := resolveShell("nobody", "", 0); err == nil || !strings.Contains(err.Error(), "no login shell") {
			t.Errorf("nobody (%s) without terminal.shell: %v, want a no-login-shell error", shell, err)
		}
		if _, err := resolveShell("nobody", "/bin/sh", 0); err != nil {
			t.Errorf("nobody with terminal.shell: %v", err)
		}
	}
}

func TestShellSizeAndResize(t *testing.T) {
	ptmx, _, cmd, err := testShell(t).start(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		_ = ptmx.Close()
	}()

	if _, err := ptmx.Write([]byte("printf 'SZ=%s.\\n' \"$(stty size)\"\n")); err != nil {
		t.Fatal(err)
	}
	if m := readUntil(t, ptmx, regexp.MustCompile(`SZ=(\d+ \d+)\.`)); m[1] != "24 80" {
		t.Errorf("initial size = %q, want 24 80", m[1])
	}

	if err := setWinsize(ptmx, 100, 30); err != nil {
		t.Fatal(err)
	}
	if _, err := ptmx.Write([]byte("printf 'SZ2=%s.\\n' \"$(stty size)\"\n")); err != nil {
		t.Fatal(err)
	}
	if m := readUntil(t, ptmx, regexp.MustCompile(`SZ2=(\d+ \d+)\.`)); m[1] != "30 100" {
		t.Errorf("size after resize = %q, want 30 100", m[1])
	}
}

// A program that leaves the shell's session but keeps the terminal open must
// neither keep a session from ending nor outlive it.
func TestSessionEndsDespiteDetachedHolder(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid not found")
	}

	ptmx, ttyName, cmd, err := testShell(t).start(80, 24)
	if err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	marker := fmt.Sprintf("%d", 31000+os.Getpid()%1000)
	script := fmt.Sprintf("sleep %s1 & setsid -f sleep %s2; echo STARTED\n", marker, marker)
	if _, err := ptmx.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	readUntil(t, ptmx, regexp.MustCompile(`(?m)^STARTED`))
	if got := len(processesWith("sleep " + marker)); got != 2 {
		t.Fatalf("%d sleep processes started, want 2", got)
	}

	readerDone := make(chan error, 1)
	go func() {
		_ = ptmx.SetReadDeadline(time.Time{})
		buf := make([]byte, 1024)
		for {
			if _, err := ptmx.Read(buf); err != nil {
				readerDone <- err
				return
			}
		}
	}()

	// the shell exits; the setsid'd sleep still holds the terminal
	if err := cmd.Process.Signal(os.Kill); err != nil {
		t.Fatal(err)
	}
	<-waitErr

	// Close must end the pending Read even though the slave is still open
	_ = ptmx.Close()
	select {
	case err := <-readerDone:
		if !errors.Is(err, os.ErrClosed) {
			t.Logf("reader ended with %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Read on the PTY master was not interrupted by Close")
	}

	endProcesses("/proc", pid, ttyName)
	deadline := time.Now().Add(3 * time.Second)
	for len(processesWith("sleep "+marker)) > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("still running after endProcesses: %v", processesWith("sleep "+marker))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// fakeController accepts the runner's control and session connections.
type fakeController struct {
	t        *testing.T
	srv      *httptest.Server
	controls chan *websocket.Conn
	sessions chan *websocket.Conn
}

func newFakeController(t *testing.T) *fakeController {
	t.Helper()

	fc := &fakeController{t: t, controls: make(chan *websocket.Conn, 4), sessions: make(chan *websocket.Conn, 4)}
	mux := http.NewServeMux()
	accept := func(ch chan *websocket.Conn) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer runner-token" {
				http.Error(w, "no", http.StatusUnauthorized)
				return
			}
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{model.TerminalSubprotocol}})
			if err != nil {
				return
			}
			ch <- conn
			<-r.Context().Done()
		}
	}
	mux.HandleFunc("/api/run/runner/terminal/control", accept(fc.controls))
	mux.HandleFunc("/api/run/runner/terminal/session", accept(fc.sessions))
	fc.srv = httptest.NewServer(mux)
	t.Cleanup(fc.srv.Close)

	return fc
}

func (fc *fakeController) agent(maxSessions int) *Agent {
	fc.t.Helper()

	a, err := NewAgent("runner-1", fc.srv.URL, "runner-token\n", config.TerminalIE{
		Enabled: true, User: currentUser(fc.t), Shell: "/bin/sh", MaxSessions: maxSessions,
	}, nopLogger{})
	if err != nil {
		fc.t.Fatal(err)
	}
	a.Start()
	fc.t.Cleanup(a.Stop)

	return a
}

func receive(t *testing.T, ch chan *websocket.Conn) *websocket.Conn {
	t.Helper()

	select {
	case c := <-ch:
		t.Cleanup(func() { _ = c.CloseNow() })
		return c
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not connect")
		return nil
	}
}

func sendJSON(t *testing.T, conn *websocket.Conn, msg model.TerminalMessage) {
	t.Helper()

	data, _ := json.Marshal(msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
}

// readOutput reads binary frames until re matches.
func readOutput(t *testing.T, conn *websocket.Conn, re *regexp.Regexp) []string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out strings.Builder
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("waiting for %q: %v; output so far:\n%s", re, err, out.String())
		}
		if typ == websocket.MessageBinary {
			out.Write(data)
			if m := re.FindStringSubmatch(out.String()); m != nil {
				return m
			}
		}
	}
}

func sessionID(n int) string {
	return fmt.Sprintf("%032x", n)
}

func TestAgentSession(t *testing.T) {
	fc := newFakeController(t)
	fc.agent(2)

	control := receive(t, fc.controls)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, hello, err := control.Read(ctx)
	if err != nil || !strings.Contains(string(hello), `"type":"hello"`) {
		t.Fatalf("hello = %s, %v", hello, err)
	}

	sendJSON(t, control, model.TerminalMessage{Type: model.TerminalMsgOpen, Session: sessionID(1), Cols: 90, Rows: 20, User: "admin"})
	session := receive(t, fc.sessions)

	if err := session.Write(ctx, websocket.MessageBinary, []byte("echo PID=$$ SUM=$((40+2)) SZ=$(stty size)\n")); err != nil {
		t.Fatal(err)
	}
	m := readOutput(t, session, regexp.MustCompile(`PID=(\d+) SUM=42 SZ=(\d+ \d+)`))
	if m[2] != "20 90" {
		t.Errorf("size = %q, want 20 90 from the open message", m[2])
	}
	pid, _ := strconv.Atoi(m[1])

	sendJSON(t, session, model.TerminalMessage{Type: model.TerminalMsgResize, Cols: 120, Rows: 40})
	if err := session.Write(ctx, websocket.MessageBinary, []byte("echo SZ2=$(stty size)\n")); err != nil {
		t.Fatal(err)
	}
	if m := readOutput(t, session, regexp.MustCompile(`SZ2=(\d+ \d+)`)); m[1] != "40 120" {
		t.Errorf("size after resize = %q, want 40 120", m[1])
	}

	// the controller closes the session: the shell must go away
	_ = session.Close(websocket.StatusNormalClosure, "window closed")
	deadline := time.Now().Add(5 * time.Second)
	for {
		state, _ := processState("/proc", pid)
		if state == "" || state == "Z" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("shell pid %d still running (%s) after the session closed", pid, state)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestAgentRejectsOverLimitAndMalformedOpen(t *testing.T) {
	fc := newFakeController(t)
	fc.agent(1)

	control := receive(t, fc.controls)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, _ = control.Read(ctx) // hello

	sendJSON(t, control, model.TerminalMessage{Type: model.TerminalMsgOpen, Session: "../../etc/passwd"})
	sendJSON(t, control, model.TerminalMessage{Type: model.TerminalMsgOpen, Session: sessionID(1)})
	receive(t, fc.sessions)

	sendJSON(t, control, model.TerminalMessage{Type: model.TerminalMsgOpen, Session: sessionID(2)})
	_, data, err := control.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var msg model.TerminalMessage
	_ = json.Unmarshal(data, &msg)
	if msg.Type != model.TerminalMsgRejected || msg.Session != sessionID(2) {
		t.Errorf("reply to an open over the limit = %s, want rejected for session 2", data)
	}

	select {
	case <-fc.sessions:
		t.Error("the runner dialed back for a malformed or rejected session")
	case <-time.After(300 * time.Millisecond):
	}
}

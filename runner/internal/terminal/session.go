package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/coder/websocket"
)

const maxTerminalSize = 1000

// runSession attaches one PTY to one data connection. It returns after the
// shell and everything else in its session have been killed.
func (a *Agent) runSession(open model.TerminalMessage) {
	conn, err := a.dialSession(open.Session)
	if err != nil {
		a.log.Warnf("Terminal session %s: could not connect back to the controller: %v", open.Session, err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	cols, rows := sizeOr(open.Cols, 80), sizeOr(open.Rows, 24)
	ptmx, ttyName, cmd, err := a.shell.start(cols, rows)
	if err != nil {
		a.log.Errorf("Terminal session %s: failed to start shell %s as %s: %v", open.Session, a.shell.path, a.shell.user, err)
		_ = writeJSON(a.ctx, conn, model.TerminalMessage{Type: model.TerminalMsgError, Message: "Runner failed to start a shell"})
		_ = conn.Close(websocket.StatusInternalError, "failed to start shell")
		return
	}

	pid := cmd.Process.Pid
	started := time.Now()
	a.log.Infof("Terminal session %s opened by %s: shell %s as %s, pid %d", open.Session, open.User, a.shell.path, a.shell.user, pid)

	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()

	var (
		lastInput atomic.Int64
		in, out   atomic.Int64
	)
	lastInput.Store(started.UnixNano())

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	// PTY -> controller
	var outputDone sync.WaitGroup
	outputDone.Add(1)
	go func() {
		defer outputDone.Done()

		buf := make([]byte, 16<<10)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				out.Add(int64(n))
				if werr := conn.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// controller -> PTY
	inputClosed := make(chan struct{})
	go func() {
		defer close(inputClosed)

		for {
			typ, data, err := conn.Read(ctx)
			if err != nil {
				return
			}

			switch typ {
			case websocket.MessageBinary:
				lastInput.Store(time.Now().UnixNano())
				in.Add(int64(len(data)))
				if _, err := ptmx.Write(data); err != nil {
					return
				}
			case websocket.MessageText:
				var msg model.TerminalMessage
				if json.Unmarshal(data, &msg) != nil || msg.Type != model.TerminalMsgResize {
					continue
				}
				if msg.Cols == 0 || msg.Rows == 0 || msg.Cols > maxTerminalSize || msg.Rows > maxTerminalSize {
					continue
				}
				_ = setWinsize(ptmx, msg.Cols, msg.Rows)
			}
		}
	}()

	check := time.NewTicker(a.checkInterval)
	defer check.Stop()

	var (
		reason   string
		exitCode = -1
		exited   bool
	)
loop:
	for {
		select {
		case err := <-waitErr:
			exited, reason = true, "shell exited"
			exitCode = exitStatus(err)
			break loop
		case <-inputClosed:
			reason = "connection closed"
			break loop
		case <-a.ctx.Done():
			reason = "runner stopping"
			break loop
		case <-check.C:
			now := time.Now()
			if now.Sub(time.Unix(0, lastInput.Load())) >= a.idleTimeout {
				reason = "idle timeout"
				break loop
			}
			if now.Sub(started) >= a.maxDuration {
				reason = "session time limit"
				break loop
			}
		}
	}

	reaped := exited
	if !exited {
		// bash passes SIGHUP on to its jobs; then make sure
		_ = syscall.Kill(pid, syscall.SIGHUP)
		select {
		case <-waitErr:
			reaped = true
		case <-time.After(time.Second):
		}
	}

	// end background jobs left in the session and anything still holding the
	// terminal, then the shell if still alive
	endProcesses(a.procPath, pid, ttyName)
	if !reaped {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		select {
		case <-waitErr:
		case <-time.After(2 * time.Second):
			a.log.Warnf("Terminal session %s: shell pid %d did not exit", open.Session, pid)
		}
	}

	// let the last output through before the exit message
	drained := make(chan struct{})
	go func() {
		outputDone.Wait()
		close(drained)
	}()
	select {
	case <-drained:
	case <-time.After(500 * time.Millisecond):
	}
	// the master is non-blocking (pollable), so Close ends a pending Read; the
	// reader can still be stuck writing to a slow browser, which the
	// conn.Close below ends
	_ = ptmx.Close()
	select {
	case <-drained:
	case <-time.After(2 * time.Second):
		a.log.Warnf("Terminal session %s: output reader did not stop", open.Session)
	}

	if exited {
		_ = writeJSON(a.ctx, conn, model.TerminalMessage{Type: model.TerminalMsgExit, Code: exitCode})
	}
	_ = conn.Close(websocket.StatusNormalClosure, reason)

	a.log.Infof("Terminal session %s closed: opened by %s, duration %s, in %dB, out %dB, reason %q",
		open.Session, open.User, time.Since(started).Round(time.Second), in.Load(), out.Load(), reason)
}

func exitStatus(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

func sizeOr(v, fallback uint16) uint16 {
	if v == 0 || v > maxTerminalSize {
		return fallback
	}

	return v
}

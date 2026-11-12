package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
)

const shellPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

// shellSpec is the resolved terminal.user and terminal.shell.
type shellSpec struct {
	path string

	user   string
	uid    uint32
	gid    uint32
	groups []uint32
	home   string

	nice int

	// switchUser is false when the runner already runs as this user
	// (for example in tests); switching needs root.
	switchUser bool
}

func resolveShell(userName, shell string, nice int) (*shellSpec, error) {
	if userName == "" {
		return nil, fmt.Errorf("terminal.user is required when the terminal is enabled (write \"root\" to use root)")
	}

	u, err := user.Lookup(userName)
	if err != nil {
		return nil, fmt.Errorf("terminal.user %q: %v", userName, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("terminal.user %q: uid %q: %v", userName, u.Uid, err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("terminal.user %q: gid %q: %v", userName, u.Gid, err)
	}

	var groups []uint32
	if ids, err := u.GroupIds(); err == nil {
		for _, id := range ids {
			if g, err := strconv.ParseUint(id, 10, 32); err == nil {
				groups = append(groups, uint32(g))
			}
		}
	}

	if shell == "" {
		shell = loginShell("/etc/passwd", userName)
		if shell == "" || isNoLoginShell(shell) {
			return nil, fmt.Errorf("terminal.user %q has no login shell (%q); give it one, or set terminal.shell to use this account anyway", userName, shell)
		}
	}
	if !filepath.IsAbs(shell) {
		return nil, fmt.Errorf("terminal.shell %q must be an absolute path", shell)
	}
	if info, err := os.Stat(shell); err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("terminal.shell %q is not an executable file", shell)
	}

	home := u.HomeDir
	if info, err := os.Stat(home); err != nil || !info.IsDir() {
		home = "/"
	}

	return &shellSpec{
		path:       shell,
		user:       u.Username,
		uid:        uint32(uid),
		gid:        uint32(gid),
		groups:     groups,
		home:       home,
		nice:       nice,
		switchUser: uint64(os.Getuid()) != uid,
	}, nil
}

// loginShell reads the user's shell from a passwd file.
func loginShell(passwdPath, userName string) string {
	f, err := os.Open(passwdPath)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) == 7 && fields[0] == userName {
			return fields[6]
		}
	}

	return ""
}

func isNoLoginShell(shell string) bool {
	base := filepath.Base(shell)
	return base == "nologin" || base == "false"
}

// start runs a login shell on a new PTY, in its own session and process
// group, so the task server's kill(-pgid) never reaches it and closing the
// terminal can end everything it started. It returns the PTY master, the
// path of the slave (to find every process still holding it at the end),
// and the running command.
func (s *shellSpec) start(cols, rows uint16) (*os.File, string, *exec.Cmd, error) {
	cmd := exec.Command(s.path, "-l")
	cmd.Dir = s.home
	cmd.Env = []string{
		"HOME=" + s.home,
		"USER=" + s.user,
		"LOGNAME=" + s.user,
		"SHELL=" + s.path,
		"PATH=" + shellPath,
		"TERM=xterm-256color",
		"LANG=C.UTF-8",
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if s.switchUser {
		cmd.SysProcAttr.Credential = &syscall.Credential{Uid: s.uid, Gid: s.gid, Groups: s.groups}
	}

	ptmx, tty, err := pty.Open()
	if err != nil {
		return nil, "", nil, err
	}
	ttyName := tty.Name()
	defer func() { _ = tty.Close() }()

	if err := pty.Setsize(ptmx, &pty.Winsize{Cols: cols, Rows: rows}); err != nil {
		_ = ptmx.Close()
		return nil, "", nil, err
	}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	if err := cmd.Start(); err != nil {
		_ = ptmx.Close()
		return nil, "", nil, err
	}

	if s.nice != 0 {
		_ = syscall.Setpriority(syscall.PRIO_PROCESS, cmd.Process.Pid, s.nice)
	}

	master, err := pollable(ptmx)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, "", nil, err
	}

	return master, ttyName, cmd, nil
}

// pollable returns a non-blocking copy of f and closes f. creack/pty works on
// f.Fd(), which switches the PTY master to blocking mode; a blocking Read is
// not interrupted by Close, so a session whose slave is still held open by
// some process could never finish. Reads on the returned file go through the
// Go poller and end when it is closed.
func pollable(f *os.File) (*os.File, error) {
	fd, err := syscall.Dup(int(f.Fd()))
	if err != nil {
		return nil, err
	}
	syscall.CloseOnExec(fd)
	if err := syscall.SetNonblock(fd, true); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}
	_ = f.Close()

	return os.NewFile(uintptr(fd), f.Name()), nil
}

// setWinsize resizes the terminal without f.Fd(), which would make the
// master blocking again (see pollable).
func setWinsize(f *os.File, cols, rows uint16) error {
	raw, err := f.SyscallConn()
	if err != nil {
		return err
	}

	ws := struct{ Row, Col, X, Y uint16 }{Row: rows, Col: cols}
	var errno syscall.Errno
	if err := raw.Control(func(fd uintptr) {
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&ws))) //nolint:gosec // ioctl takes a pointer
	}); err != nil {
		return err
	}
	if errno != 0 {
		return errno
	}

	return nil
}

// endProcesses SIGKILLs every process in session sid (background jobs
// included, though they have their own process group) and every process that
// still has the terminal ttyName open, such as a program started with setsid
// from the shell. It scans again until it finds none, because a process can
// fork between a scan and the kill, and gives up after a few rounds. A
// program that both leaves the session and closes the terminal (a daemon,
// tmux) survives; so do containers started through docker.
func endProcesses(procPath string, sid int, ttyName string) int {
	killed := 0
	for range 20 {
		n := 0
		for _, pid := range sessionProcesses(procPath, sid, ttyName) {
			if syscall.Kill(pid, syscall.SIGKILL) == nil {
				n++
			}
		}
		if n == 0 {
			return killed
		}
		killed += n
		time.Sleep(50 * time.Millisecond)
	}

	return killed
}

// sessionProcesses lists the live (not zombie) processes in session sid or
// with ttyName open.
func sessionProcesses(procPath string, sid int, ttyName string) []int {
	entries, err := os.ReadDir(procPath)
	if err != nil {
		return nil
	}

	self := os.Getpid()
	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}

		state, psid := processState(procPath, pid)
		if state == "" || state == "Z" {
			continue
		}
		if psid == sid || (ttyName != "" && holdsFile(procPath, pid, ttyName)) {
			pids = append(pids, pid)
		}
	}

	return pids
}

// processState returns the state and session id of pid from
// /proc/<pid>/stat, or "" and -1.
func processState(procPath string, pid int) (string, int) {
	data, err := os.ReadFile(filepath.Join(procPath, strconv.Itoa(pid), "stat"))
	if err != nil {
		return "", -1
	}

	// the command name is in parentheses and may contain spaces
	i := strings.LastIndexByte(string(data), ')')
	if i < 0 {
		return "", -1
	}

	// after ")": state ppid pgrp session ...
	fields := strings.Fields(string(data[i+1:]))
	if len(fields) < 4 {
		return "", -1
	}

	sid, err := strconv.Atoi(fields[3])
	if err != nil {
		return "", -1
	}

	return fields[0], sid
}

// holdsFile reports whether pid has path open.
func holdsFile(procPath string, pid int, path string) bool {
	dir := filepath.Join(procPath, strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		// once the master is closed, the pts node is removed and the link
		// reads "<path> (deleted)"
		if target, err := os.Readlink(filepath.Join(dir, e.Name())); err == nil && (target == path || target == path+" (deleted)") {
			return true
		}
	}

	return false
}

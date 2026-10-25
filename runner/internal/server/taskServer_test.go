package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, output)
	}
}

func newGitRepo(t *testing.T, dir, subject string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init", "--quiet")
	runGit(t, dir, "commit", "--quiet", "--allow-empty", "-m", subject)
}

func TestListCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	// Same layout as a task workspace: ci-test with free5gc cloned inside it
	root := t.TempDir()
	repoDir := filepath.Join(root, constant.CI_TEST_REPO)
	f5gcDir := filepath.Join(repoDir, constant.FREE5GC_REPO)

	newGitRepo(t, filepath.Join(root, "amf"), "amf default branch")
	newGitRepo(t, filepath.Join(root, "nrf"), "nrf pull request")
	newGitRepo(t, repoDir, "ci-test main")
	newGitRepo(t, f5gcDir, "free5gc main")
	runGit(t, f5gcDir, "-c", "protocol.file.allow=always", "submodule", "--quiet", "add", filepath.Join(root, "amf"), "NFs/amf")
	runGit(t, f5gcDir, "-c", "protocol.file.allow=always", "submodule", "--quiet", "add", filepath.Join(root, "nrf"), "NFs/nrf")

	s := &taskServer{RunnerLogger: logger.NewRunnerLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true)}
	output := s.listCommits(f5gcDir, []model.NfPr{{NfName: "nrf", PR: 95}})

	for _, want := range []string{
		`(?m)^ci-test +[0-9a-f]{7,} ci-test main$`,
		`(?m)^free5gc +[0-9a-f]{7,} free5gc main$`,
		`(?m)^NFs/amf +[0-9a-f]{7,} amf default branch$`,
		`(?m)^NFs/nrf +[0-9a-f]{7,} nrf pull request \(PR #95\)$`,
	} {
		if !regexp.MustCompile(want).MatchString(output) {
			t.Errorf("output does not match %q:\n%s", want, output)
		}
	}
}

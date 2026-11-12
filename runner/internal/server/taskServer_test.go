package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

// newOriginWithPR makes a bare repository that, like GitHub, has a pull
// request under refs/pull/<n>/head, and returns its path and the commit.
func newOriginWithPR(t *testing.T, dir, name string, pr int) string {
	t.Helper()

	work := filepath.Join(dir, name+"-work")
	newGitRepo(t, work, name+" default branch")
	runGit(t, work, "checkout", "--quiet", "-b", "feature")
	runGit(t, work, "commit", "--quiet", "--allow-empty", "-m", name+" pull request")

	origin := filepath.Join(dir, name+".git")
	runGit(t, dir, "clone", "--quiet", "--bare", work, origin)
	runGit(t, origin, "update-ref", fmt.Sprintf("refs/pull/%d/head", pr), "refs/heads/feature")

	return origin
}

func gitHead(t *testing.T, dir string) string {
	t.Helper()

	cmd := exec.Command("git", "log", "-1", "--format=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log in %s: %v", dir, err)
	}

	return strings.TrimSpace(string(out))
}

func TestFetchNfPrCheckoutPaths(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	// free5gc keeps the NFs under NFs/ and webconsole at the top level
	root := t.TempDir()
	repoDir := filepath.Join(root, constant.CI_TEST_REPO)
	f5gcDir := filepath.Join(repoDir, constant.FREE5GC_REPO)
	newGitRepo(t, repoDir, "ci-test main")
	newGitRepo(t, f5gcDir, "free5gc main")

	for _, sub := range []struct {
		name, path string
		pr         int
	}{
		{"amf", "NFs/amf", 242},
		{"webconsole", "webconsole", 7},
	} {
		origin := newOriginWithPR(t, root, sub.name, sub.pr)
		runGit(t, f5gcDir, "-c", "protocol.file.allow=always", "submodule", "--quiet", "add", origin, sub.path)
	}

	s := &taskServer{RunnerLogger: logger.NewRunnerLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true)}
	output, err := s.fetchNfPr(context.Background(), []model.NfPr{{NfName: "amf", PR: 242}, {NfName: "webconsole", PR: 7}}, f5gcDir)
	if err != nil {
		t.Fatalf("fetchNfPr: %v\n%s", err, output)
	}

	for _, c := range []struct{ path, want string }{
		{"NFs/amf", "amf pull request"},
		{"webconsole", "webconsole pull request"},
	} {
		if got := gitHead(t, filepath.Join(f5gcDir, c.path)); got != c.want {
			t.Errorf("%s is at %q, want %q", c.path, got, c.want)
		}
	}

	if want := `(?m)^webconsole +[0-9a-f]{7,} webconsole pull request \(PR #7\)$`; !regexp.MustCompile(want).MatchString(output) {
		t.Errorf("output does not match %q:\n%s", want, output)
	}
}

package repowatch

import (
	"context"
	"os"
	"testing"

	"github.com/Cloud-Foundations/golib/pkg/log/testlogger"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

const testHttpGitTarget = "https://github.com/Cloud-Foundations/golib.git"

func TestIsHttpRepo(t *testing.T) {
	httpRepos := []string{
		testHttpGitTarget,
	}
	sshRepos := []string{
		"git@github.com:Cloud-Foundations/golib.git",
	}
	for _, url := range httpRepos {
		if !isHttpRepo(url) {
			t.Fatalf("Reported as NON http %s", url)
		}
	}
	for _, url := range sshRepos {
		if isHttpRepo(url) {
			t.Fatalf("Reported AS http %s", url)
		}
	}

}

func TestSetupGitRepositoryHttp(t *testing.T) {
	dir, err := os.MkdirTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up
	config := Config{
		LocalRepositoryDirectory: dir,
		RepositoryURL:            testHttpGitTarget,
	}
	params := Params{
		Logger: testlogger.New(t),
	}

	metrics := &gitMetricsType{
		latencyDistribution: tricorder.NewGeometricBucketer(1, 1e5).
			NewCumulativeDistribution(),
	}

	ctx := context.Background()
	_, _, err = setupGitRepository(ctx, config, params, metrics) //(*git.Worktree, string, error) {
	if err != nil {
		t.Logf("Error setting up git repo err=%s", err)
		t.Fatal(err)
	}

}

package repowatch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/Cloud-Foundations/golib/pkg/log/testlogger"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
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

func checkUrlBase(repoURL string) error {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return nil
	}
	probeTarget := parsedURL.Scheme + "://" + parsedURL.Host
	_, err = http.Get(probeTarget)
	if err != nil {
		return fmt.Errorf("could not fetch %s", probeTarget)
	}
	return nil
}

func TestHttpPullAndSetupGitRepository(t *testing.T) {
	err := checkUrlBase(testHttpGitTarget)
	if err != nil {
		t.Skipf("base test url not reachable... cannot test init or pull err=%s", err)
	}
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
	err = tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"git-pull-latency"), metrics.latencyDistribution,
		units.Millisecond, "latency of git pull calls")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()
	worktree, _, err := setupGitRepository(ctx, config, params, metrics)
	if err != nil {
		t.Logf("Error setting up git repo err=%s", err)
		t.Fatal(err)
	}
	_, err = gitPull(worktree, config.LocalRepositoryDirectory,
		metrics)
	if err != nil {
		t.Logf("Error pulling git repo err=%s", err)
		t.Fatal(err)
	}
}

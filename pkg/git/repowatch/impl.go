package repowatch

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/log"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

var (
	transportAuth     transport.AuthMethod
	transportAuthLock sync.Mutex
)

type gitMetricsType struct {
	lastAttemptedPullTime  time.Time
	lastCommitId           string
	lastSuccessfulPullTime time.Time
	lastNotificationTime   time.Time
	latencyDistribution    *tricorder.CumulativeDistribution
}

func checkDirectory(directory string) error {
	if fi, err := os.Stat(directory); err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("not a directory: %s", directory)
	}
	return nil
}

func gitPull(worktree *git.Worktree, repositoryDirectory string,
	metrics *gitMetricsType) (string, error) {
	metrics.lastAttemptedPullTime = time.Now()
	transportAuthLock.Lock()
	pullOptions := &git.PullOptions{Auth: transportAuth}
	transportAuthLock.Unlock()
	if err := worktree.Pull(pullOptions); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return "", err
		}
	}
	metrics.lastSuccessfulPullTime = time.Now()
	metrics.latencyDistribution.Add(
		metrics.lastSuccessfulPullTime.Sub(metrics.lastAttemptedPullTime))
	return readLatestCommitId(repositoryDirectory)
}

func readLatestCommitId(repositoryDirectory string) (string, error) {
	commitId, err := ioutil.ReadFile(
		filepath.Join(repositoryDirectory, ".git/refs/heads/master"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(commitId)), nil
}

func setupGitRepository(ctx context.Context, config Config, params Params,
	metrics *gitMetricsType) (*git.Worktree, string, error) {
	err := os.MkdirAll(config.LocalRepositoryDirectory, fsutil.DirPerms)
	if err != nil {
		return nil, "", err
	}
	gitSubdir := filepath.Join(config.LocalRepositoryDirectory, ".git")
	if _, err := os.Stat(gitSubdir); err != nil {
		if !os.IsNotExist(err) {
			return nil, "", err
		}
		metrics.lastAttemptedPullTime = time.Now()
		pubkeys, err := getAuth(ctx, params.SecretsClient, config.AwsSecretId,
			params.Logger)
		if err != nil {
			return nil, "", err
		}
		transportAuth = pubkeys
		repo, err := git.PlainClone(config.LocalRepositoryDirectory, false,
			&git.CloneOptions{
				Auth: transportAuth,
				URL:  config.RepositoryURL,
			})
		if err != nil {
			return nil, "", err
		}
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, "", err
		}
		metrics.lastSuccessfulPullTime = time.Now()
		lastCommitId, err := readLatestCommitId(config.LocalRepositoryDirectory)
		return worktree, lastCommitId, err
	} else {
		repo, err := git.PlainOpen(config.LocalRepositoryDirectory)
		if err != nil {
			return nil, "", err
		}
		worktree, err := repo.Worktree()
		if err != nil {
			return nil, "", err
		}
		go func() {
			ctx := context.Background()
			for {
				pubkeys, err := getAuth(ctx, params.SecretsClient,
					config.AwsSecretId, params.Logger)
				if err != nil {
					params.Logger.Println(err)
					time.Sleep(time.Minute * 5)
				} else {
					transportAuthLock.Lock()
					transportAuth = pubkeys
					transportAuthLock.Unlock()
					return
				}
			}
		}()
		// Try to be as fresh as possible.
		commitId, err := gitPull(worktree, config.LocalRepositoryDirectory,
			metrics)
		if err != nil {
			params.Logger.Println(err)
			lastCommitId, err := readLatestCommitId(
				config.LocalRepositoryDirectory)
			return worktree, lastCommitId, err
		} else {
			return worktree, commitId, nil
		}
	}
}

func watch(config Config, params Params) (<-chan string, error) {
	if config.Branch != "" && config.Branch != "master" {
		return nil, errors.New("non-master branch not supported")
	}
	if config.CheckInterval < time.Second {
		config.CheckInterval = time.Second
	}
	if config.RepositoryURL == "" {
		return watchLocal(config.LocalRepositoryDirectory, config.CheckInterval,
			params.MetricDirectory, params.Logger)
	}
	return watchGit(config, params)
}

func watchGit(config Config, params Params) (<-chan string, error) {
	notificationChannel := make(chan string, 1)
	metrics := &gitMetricsType{
		latencyDistribution: tricorder.NewGeometricBucketer(1, 1e5).
			NewCumulativeDistribution(),
	}
	err := tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"git-pull-latency"), metrics.latencyDistribution,
		units.Millisecond, "latency of git pull calls")
	if err != nil {
		return nil, err
	}
	ctx := context.TODO()
	if config.AwsSecretId != "" {
		if params.SecretsClient == nil {
			if params.AwsConfig == nil {
				awsConfig, err := awsconfig.LoadDefaultConfig(ctx,
					awsconfig.WithEC2IMDSRegion())
				if err != nil {
					return nil, err
				}
				params.AwsConfig = &awsConfig
			}
			params.SecretsClient = secretsmanager.NewFromConfig(
				*params.AwsConfig)
		}
	}
	var worktree *git.Worktree
	worktree, metrics.lastCommitId, err = setupGitRepository(ctx, config,
		params, metrics)
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"last-attempted-git-pull-time"), &metrics.lastAttemptedPullTime,
		units.None, "time of last attempted git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"last-commit-id"), &metrics.lastCommitId,
		units.None, "commit ID in master branch in  last successful git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"last-successful-git-pull-time"), &metrics.lastSuccessfulPullTime,
		units.None, "time of last successful git pull")
	if err != nil {
		return nil, err
	}
	err = tricorder.RegisterMetric(filepath.Join(params.MetricDirectory,
		"last-notification-time"), &metrics.lastNotificationTime, units.None,
		"time of last git change notification")
	if err != nil {
		return nil, err
	}
	metrics.lastNotificationTime = time.Now()
	notificationChannel <- config.LocalRepositoryDirectory
	go watchGitLoop(worktree, config, params, metrics, notificationChannel)
	return notificationChannel, nil
}

func watchGitLoop(worktree *git.Worktree, config Config, params Params,
	metrics *gitMetricsType,
	notificationChannel chan<- string) {
	for {
		time.Sleep(config.CheckInterval)
		commitId, err := gitPull(worktree, config.LocalRepositoryDirectory,
			metrics)
		if err != nil {
			params.Logger.Println(err)
		} else if commitId != metrics.lastCommitId {
			metrics.lastCommitId = commitId
			metrics.lastNotificationTime = time.Now()
			notificationChannel <- config.LocalRepositoryDirectory
		}
	}
}

func watchLocal(directory string, checkInterval time.Duration,
	metricDirectory string, logger log.DebugLogger) (<-chan string, error) {
	if err := checkDirectory(directory); err != nil {
		return nil, err
	}
	var lastNotificationTime time.Time
	err := tricorder.RegisterMetric(filepath.Join(metricDirectory,
		"last-notification-time"), &lastNotificationTime, units.None,
		"time of last notification")
	if err != nil {
		return nil, err
	}
	notificationChannel := make(chan string, 1)
	go watchLocalLoop(directory, checkInterval, &lastNotificationTime,
		notificationChannel, logger)
	return notificationChannel, nil
}

func watchLocalLoop(directory string, checkInterval time.Duration,
	lastNotificationTime *time.Time, notificationChannel chan<- string,
	logger log.DebugLogger) {
	for ; ; time.Sleep(checkInterval) {
		if err := checkDirectory(directory); err != nil {
			logger.Println(err)
		} else {
			*lastNotificationTime = time.Now()
			notificationChannel <- directory
		}
	}
}

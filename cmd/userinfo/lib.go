package main

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo/gitdb"
	"github.com/Cloud-Foundations/golib/pkg/git/repowatch"
	"github.com/Cloud-Foundations/golib/pkg/log"
)

// Get the database. If source is a directory, it specifies a local repository
// else it a remote repository.
func getDB(source string, logger log.DebugLogger) (*gitdb.UserInfo, error) {
	var tmpdir string
	fi, err := os.Stat(source)
	if err == nil && fi.IsDir() {
		tmpdir = source
		source = ""
	} else {
		tmpdir, err = ioutil.TempDir("", "userinfo")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tmpdir)
	}
	return gitdb.New2(gitdb.Config{Config: repowatch.Config{
		AwsSecretId:              *awsSecretId,
		CheckInterval:            time.Hour,
		LocalRepositoryDirectory: tmpdir,
		RepositoryURL:            source,
	}},
		gitdb.Params{Params: repowatch.Params{
			Logger: logger,
		}})
}

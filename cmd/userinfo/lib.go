package main

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/golib/pkg/auth/userinfo/gitdb"
)

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
	return gitdb.New(source, "", tmpdir, time.Hour, logger)
}

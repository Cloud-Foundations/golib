package gitdb

import (
	"testing"

	"github.com/Cloud-Foundations/golib/pkg/log/testlogger"
)

func TestFlat(t *testing.T) {
	uinfo := UserInfo{logger: testlogger.New(t)}
	if err := uinfo.loadDatabase("testdata/flat"); err != nil {
		t.Fatal(err)
	}
	uinfo.testDB(t)
}

func TestLoop(t *testing.T) {
	uinfo := UserInfo{logger: testlogger.New(t)}
	if err := uinfo.loadDatabase("testdata/loop"); err != nil {
		t.Fatal(err)
	}
	for user, groups := range uinfo.groupsPerUser {
		if len(groups) != 3 {
			return
		}
		uinfo.logger.Printf("%s: %v\n", user, groups)
	}
	t.Fatal("loop detection failed: all users are members of all groups")
}

func TestTree(t *testing.T) {
	uinfo := UserInfo{logger: testlogger.New(t)}
	if err := uinfo.loadDatabase("testdata/tree"); err != nil {
		t.Fatal(err)
	}
	uinfo.testDB(t)
}

func (uinfo *UserInfo) testDB(t *testing.T) {
	uinfo.testUserA(t)
	uinfo.testUserB(t)
	uinfo.testUserC(t)
}

func (uinfo *UserInfo) testUserA(t *testing.T) {
	if _, ok := uinfo.groupsPerUser["userA"]; ok {
		t.Fatal("userA has groups: DB not case-folded")
	} else if groups, ok := uinfo.groupsPerUser["usera"]; !ok {
		t.Fatal("no groups for usera")
	} else if _, ok := groups["project0"]; !ok {
		t.Fatal("usera not found in project0")
	} else if !uinfo.TestUserInGroup("userA", "project0") {
		t.Fatal("userA not found in project0 using TestUserInGroup")
	} else if !uinfo.TestUserInGroup("usera", "project0") {
		t.Fatal("usera not found in project0 using TestUserInGroup")
	} else if _, ok := groups["team0"]; !ok {
		t.Fatal("usera not found in team0")
	} else if _, ok := groups["unpermitted0"]; ok {
		t.Fatal("usera found in unpermitted team")
	} else if uinfo.TestUserInGroup("userA", "unpermitted0") {
		t.Fatal("userA found in unpermitted0 using TestUserInGroup")
	} else if uinfo.TestUserInGroup("usera", "unpermitted0") {
		t.Fatal("usera found in unpermitted0 using TestUserInGroup")
	} else if len(groups) != 2 {
		t.Fatalf("usera in %d groups, expected 2", len(groups))
	} else if g, err := uinfo.GetUserGroups("userA"); err != nil {
		t.Fatal(err)
	} else if len(g) != 2 {
		t.Fatalf("userA in %d groups using GetUserGroups, expected 2",
			len(groups))
	} else if g, err := uinfo.GetUserGroups("usera"); err != nil {
		t.Fatal(err)
	} else if len(g) != 2 {
		t.Fatalf("usera in %d groups using GetUserGroups, expected 2",
			len(groups))
	}
}

func (uinfo *UserInfo) testUserB(t *testing.T) {
	if groups, ok := uinfo.groupsPerUser["userb"]; !ok {
		t.Fatal("no groups for userb")
	} else if _, ok := groups["project1"]; !ok {
		t.Fatal("userb not found in project1")
	} else if _, ok := groups["team1"]; !ok {
		t.Fatal("userb not found in team1")
	} else if len(groups) != 2 {
		t.Fatalf("userb in %d groups, expected 2", len(groups))
	}
}

func (uinfo *UserInfo) testUserC(t *testing.T) {
	if groups, ok := uinfo.groupsPerUser["userc"]; !ok {
		t.Fatal("no groups for userc")
	} else if _, ok := groups["project0"]; !ok {
		t.Fatal("userc not found in project0")
	} else if !uinfo.TestUserInGroup("userc", "project0") {
		t.Fatal("userc not found in project0 using TestUserInGroup")
	} else if _, ok := groups["team0"]; !ok {
		t.Fatal("userc not found in team0")
	} else if len(groups) != 2 {
		t.Fatalf("userc in %d groups, expected 2", len(groups))
	} else if g, err := uinfo.GetUserGroups("userc"); err != nil {
		t.Fatal(err)
	} else if len(g) != 2 {
		t.Fatalf("userc in %d groups using GetUserGroups, expected 2",
			len(groups))
	}
}

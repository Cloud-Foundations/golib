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
	if sm, err := uinfo.GetGroupServiceMethods("project0"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length > 0 {
		t.Fatalf("project0 has %d ServiceMethods", length)
	}
	if sm, err := uinfo.GetGroupServiceMethods("project1"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 2 {
		t.Fatalf("project1 has %d ServiceMethods", length)
	} else if sm[0] != "BarServer.Reboot" {
		t.Fatalf("project1 has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "MultiServer.FuncOne" {
		t.Fatalf("project1 has bad ServiceMethod: %s", sm[1])
	}
	if sm, err := uinfo.GetGroupServiceMethods("team0"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 2 {
		t.Fatalf("team0 has %d ServiceMethods", length)
	} else if sm[0] != "FooServer.Shutdown" {
		t.Fatalf("team0 has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "MultiServer.*" {
		t.Fatalf("team0 has bad ServiceMethod: %s", sm[1])
	}
	if sm, err := uinfo.GetGroupServiceMethods("team1"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length > 0 {
		t.Fatalf("team1 has %d ServiceMethods", length)
	}
	uinfo.testUserA(t)
	uinfo.testUserB(t)
	uinfo.testUserC(t)
	uinfo.testUserD(t)
	uinfo.testUserE(t)
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
	} else if sm, err := uinfo.GetUserServiceMethods("usera"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 2 {
		t.Fatalf("usera has %d ServiceMethods", length)
	} else if sm[0] != "FooServer.Shutdown" {
		t.Fatalf("usera has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "MultiServer.*" {
		t.Fatalf("usera has bad ServiceMethod: %s", sm[1])
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
	} else if sm, err := uinfo.GetUserServiceMethods("userb"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 2 {
		t.Fatalf("userb has %d ServiceMethods", length)
	} else if sm[0] != "BarServer.Reboot" {
		t.Fatalf("userb has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "MultiServer.FuncOne" {
		t.Fatalf("userb has bad ServiceMethod: %s", sm[1])
	}
}

func (uinfo *UserInfo) testUserC(t *testing.T) {
	if groups, ok := uinfo.groupsPerUser["userc"]; !ok {
		t.Fatal("no groups for userc")
	} else if _, ok := groups["project0"]; !ok {
		t.Fatal("userc not found in project0")
	} else if !uinfo.TestUserInGroup("userc", "project0") {
		t.Fatal("userc not found in project0 using TestUserInGroup")
	} else if _, ok := groups["project1"]; !ok {
		t.Fatal("userc not found in project1")
	} else if !uinfo.TestUserInGroup("userc", "project1") {
		t.Fatal("userc not found in project1 using TestUserInGroup")
	} else if _, ok := groups["team0"]; !ok {
		t.Fatal("userc not found in team0")
	} else if _, ok := groups["team1"]; !ok {
		t.Fatal("userc not found in team1")
	} else if len(groups) != 4 {
		t.Fatalf("userc in %d groups, expected 4", len(groups))
	} else if g, err := uinfo.GetUserGroups("userc"); err != nil {
		t.Fatal(err)
	} else if len(g) != 4 {
		t.Fatalf("userc in %d groups using GetUserGroups, expected 4",
			len(groups))
	} else if sm, err := uinfo.GetUserServiceMethods("userc"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 3 {
		t.Fatalf("userc has %d ServiceMethods", length)
	} else if sm[0] != "BarServer.Reboot" {
		t.Fatalf("userc has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "FooServer.Shutdown" {
		t.Fatalf("userc has bad ServiceMethod: %s", sm[1])
	} else if sm[2] != "MultiServer.*" {
		t.Fatalf("userc has bad ServiceMethod: %s", sm[2])
	}
}

func (uinfo *UserInfo) testUserD(t *testing.T) {
	if sm, err := uinfo.GetUserServiceMethods("userd"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 1 {
		t.Fatalf("userd has %d ServiceMethods", length)
	} else if sm[0] != "*.*" {
		t.Fatalf("userd has bad ServiceMethod: %s", sm[0])
	}
}

func (uinfo *UserInfo) testUserE(t *testing.T) {
	if sm, err := uinfo.GetUserServiceMethods("usere"); err != nil {
		t.Fatal(err)
	} else if length := len(sm); length != 2 {
		t.Fatalf("usere has %d ServiceMethods", length)
	} else if sm[0] != "*.Reboot" {
		t.Fatalf("usere has bad ServiceMethod: %s", sm[0])
	} else if sm[1] != "MultiServer.FuncOne" {
		t.Fatalf("usere has bad ServiceMethod: %s", sm[1])
	}
}

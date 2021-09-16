package authinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testAuthInfo = AuthInfo{
		Username: "a-user",
		Groups:   []string{"company", "admin-team"},
	}
)

func TestCheckPermittedGroups(t *testing.T) {
	assert.False(t, testAuthInfo.CheckGroup("a-team"), "a-team is not a member")
	assert.True(t, testAuthInfo.CheckGroup("company"), "company is a member")
	assert.True(t, testAuthInfo.CheckGroup("admin-team"),
		"admin-team is a member")
}

package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilters(t *testing.T) {
	f := Filter{}
	assert.Nil(t, f.IdentityFilters())
	assert.Nil(t, f.ClassFilters())
	assert.Nil(t, f.AgentFilters())

	f.AddClassFilter("testing1")
	f.AddClassFilter("testing1")
	f.AddClassFilter("testing2")
	assert.Equal(t, f.ClassFilters(), []string{"testing1", "testing2"})

	f.AddAgentFilter("agent1")
	f.AddAgentFilter("agent1")
	f.AddAgentFilter("agent2")
	assert.Equal(t, f.AgentFilters(), []string{"agent1", "agent2"})

	f.AddIdentityFilter("id1")
	f.AddIdentityFilter("id1")
	f.AddIdentityFilter("id2")
	assert.Equal(t, f.IdentityFilters(), []string{"id1", "id2"})

	f.AddCompoundFilter("foo or bar")
	f.AddCompoundFilter("foo or bar")
	f.AddCompoundFilter("bar or foo")
	assert.Equal(t, f.CompoundFilters(), []string{"foo or bar", "bar or foo"})

	e := f.AddFactFilter("test1", ">=", "1")
	assert.Nil(t, e)
	e = f.AddFactFilter("test2", ">=", "2")
	assert.Nil(t, e)
	e = f.AddFactFilter("test3", "foo", "3")
	assert.Error(t, e)

	assert.Equal(t, f.FactFilters(), [][3]string{[3]string{"test1", ">=", "1"}, [3]string{"test2", ">=", "2"}})
}

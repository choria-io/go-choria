package choria

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tc struct {
	KeyOne string `confkey:"test.one" default:"one" environment:"ONE_OVERRIDE"`
	KeyTwo string `confkey:"test.two" default:"two"`
}

func TestTag(t *testing.T) {
	c := tc{}

	tag, _ := tag(c, "KeyOne", "default")

	assert.Equal(t, "one", tag)
}
func TestNewChoria(t *testing.T) {
	c := newChoria()
	assert.Equal(t, "puppet", c.DiscoveryHost)
	assert.Equal(t, 8085, c.DiscoveryPort)
	assert.Equal(t, true, c.UseSRVRecords)
}

func TestParseConfig(t *testing.T) {
	c, err := NewConfig("testdata/choria.cfg")

	assert.NoError(t, err, "should not be error")
	assert.Equal(t, "pdb.example.com", c.Choria.DiscoveryHost)
	assert.Equal(t, "Foo", c.Registration)
	assert.Equal(t, 10, c.RegisterInterval)
	assert.Equal(t, true, c.RegistrationSplay)
	assert.Equal(t, []string{"c_1", "c_2", "c_3"}, c.Collectives)
	assert.Equal(t, "c_1", c.MainCollective)
	assert.Equal(t, 5, c.KeepLogs)
	assert.Equal(t, []string{"/dir1", "/dir2", "/dir3", "/dir4"}, c.LibDir)
	assert.Equal(t, []string{"one", "two"}, c.DefaultDiscoveryOptions)
	assert.Equal(t, true, c.Choria.RandomizeMiddlewareHosts)
}
func TestSetDefaults(t *testing.T) {
	data := tc{}
	setDefaults(&data)

	assert.Equal(t, "one", data.KeyOne)
	assert.Equal(t, "two", data.KeyTwo)
}

func TestItemWithKey(t *testing.T) {
	data := tc{}

	k, err := itemWithKey(data, "test.one")
	assert.Equal(t, k, "KeyOne")
	assert.NoError(t, err, "should not be error")

	k, err = itemWithKey(data, "test.two")
	assert.Equal(t, k, "KeyTwo")
	assert.NoError(t, err, "should not be error")

	k, err = itemWithKey(data, "test.three")
	assert.Equal(t, k, "")
	assert.Error(t, err, "should be error")
}

func TestSetItemWithKey(t *testing.T) {
	data := tc{}

	setItemWithKey(&data, "test.one", "new value")
	assert.Equal(t, data.KeyOne, "new value")
	assert.Equal(t, data.KeyTwo, "")
}

func TestSetItemWithKeyEnvOverride(t *testing.T) {
	data := tc{}

	setItemWithKey(&data, "test.one", "new value")
	assert.Equal(t, data.KeyOne, "new value")

	os.Setenv("ONE_OVERRIDE", "OVERRIDE")
	setItemWithKey(&data, "test.one", "new value")
	assert.Equal(t, data.KeyOne, "OVERRIDE")

}

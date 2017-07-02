package federation

import (
	"testing"
	"time"

	"github.com/choria-io/go-choria/mcollective"
	"github.com/stretchr/testify/assert"
)

func TestNewFederationBroker(t *testing.T) {
	choria, err := mcollective.New("testdata/federation.cfg")
	assert.Nil(t, err)

	fb, err := NewFederationBroker("test_cluster", "test_instance", choria)
	assert.Nil(t, err)

	assert.Equal(t, "unknown", fb.Stats.Status)
	assert.Equal(t, "unknown", fb.Stats.CollectiveStats.ConnectedServer)
	assert.Equal(t, "unknown", fb.Stats.FederationStats.ConnectedServer)

	d, _ := time.ParseDuration("1s")
	assert.WithinDuration(t, fb.Stats.StartTime, time.Now(), d)
}

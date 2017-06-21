package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRequest(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	filter, filtered := request.Filter()

	request.SetMessage("hello world")

	j, _ := request.JSON()

	assert.Equal(t, "choria:request:1", gjson.Get(j, "protocol").String())
	assert.Equal(t, "hello world", request.Message())
	assert.Equal(t, 32, len(request.RequestID()))
	assert.Equal(t, "go.tests", request.SenderID())
	assert.Equal(t, "choria=test", request.CallerID())
	assert.Equal(t, "mcollective", request.Collective())
	assert.Equal(t, "test", request.Agent())
	assert.Equal(t, 120, request.TTL())
	assert.NotZero(t, request.Time())
	assert.False(t, filtered)
	assert.True(t, filter.Empty())

	filter.AddAgentFilter("rpcutil")
	filter, filtered = request.Filter()

	assert.True(t, filtered)
	assert.NotNil(t, filter)
}

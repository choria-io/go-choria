package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestNewReply(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "choria=test", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	reply, _ := NewReply(request)

	reply.SetMessage("hello world")

	j, _ := reply.JSON()

	assert.Equal(t, "choria:reply:1", gjson.Get(j, "protocol").String())
	assert.Equal(t, "hello world", reply.Message())
	assert.Equal(t, 32, len(reply.RequestID()))
	assert.Equal(t, "go.tests", reply.SenderID())
	assert.Equal(t, "test", reply.Agent())
	assert.NotZero(t, reply.Time())
}

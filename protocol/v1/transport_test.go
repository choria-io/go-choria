package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestTransportReply(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	request.SetMessage(`{"message":1}`)
	reply, _ := NewReply(request)
	sreply, _ := NewSecureReply(reply)
	treply, _ := NewTransportMessage("rip.mcollective")
	treply.SetReplyData(sreply)

	sj, err := sreply.JSON()
	assert.Nil(t, err)
	j, err := treply.JSON()
	assert.Nil(t, err)

	assert.Equal(t, "choria:transport:1", gjson.Get(j, "protocol").String())
	assert.Equal(t, "rip.mcollective", gjson.Get(j, "headers.mc_sender").String())

	d, err := treply.Message()
	assert.Nil(t, err)

	assert.Equal(t, sj, d)
}

func TestTransportRequest(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	request.SetMessage(`{"message":1}`)
	srequest, _ := NewSecureRequest(request, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
	trequest, _ := NewTransportMessage("rip.mcollective")
	trequest.SetRequestData(srequest)

	sj, _ := srequest.JSON()
	j, _ := trequest.JSON()

	assert.Equal(t, "choria:transport:1", gjson.Get(j, "protocol").String())
	assert.Equal(t, "rip.mcollective", gjson.Get(j, "headers.mc_sender").String())

	d, err := trequest.Message()
	assert.Nil(t, err)

	assert.Equal(t, sj, d)
}

func TestTransportFromJSON(t *testing.T) {
	request, _ := NewRequest("test", "go.tests", "rip.mcollective", 120, "a2f0ca717c694f2086cfa81b6c494648", "mcollective")
	srequest, _ := NewSecureRequest(request, "testdata/ssl/certs/rip.mcollective.pem", "testdata/ssl/private_keys/rip.mcollective.pem")
	trequest, _ := NewTransportMessage("rip.mcollective")
	trequest.SetRequestData(srequest)

	j, _ := trequest.JSON()

	_, err := NewTransportFromJSON(j)
	assert.Nil(t, err)

	_, err = NewTransportFromJSON(`{"protocol": 1}`)
	assert.Equal(t, "Supplied JSON document is not a valid Transport message: data: data is required, headers: headers is required, protocol: Invalid type. Expected: string, given: integer", err.Error())
}

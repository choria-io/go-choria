// generated code; DO NOT EDIT

package executorclient

import (
	"time"

	"github.com/choria-io/go-choria/providers/agent/mcorpc"
)

// ResultDetails is the details about a result
type ResultDetails struct {
	sender  string
	code    int
	message string
	ts      time.Time
}

// StatusCode is a reply status as defined by MCollective SimpleRPC - integers 0 to 5
//
// See the constants OK, RPCAborted, UnknownRPCAction, MissingRPCData, InvalidRPCData and UnknownRPCError
type StatusCode uint8

const (
	// OK is the reply status when all worked
	OK = StatusCode(iota)

	// Aborted is status for when the action could not run, most failures in an action should set this
	Aborted

	// UnknownAction is the status for unknown actions requested
	UnknownAction

	// MissingData is the status for missing input data
	MissingData

	// InvalidData is the status for invalid input data
	InvalidData

	// UnknownError is the status general failures in agents should set when things go bad
	UnknownError
)

// Sender is the identity of the remote that produced the message
func (d *ResultDetails) Sender() string {
	return d.sender
}

// OK determines if the request was successful
func (d *ResultDetails) OK() bool {
	return mcorpc.StatusCode(d.code) == mcorpc.OK
}

// StatusMessage is the status message produced by the remote
func (d *ResultDetails) StatusMessage() string {
	return d.message
}

// StatusCode is the status code produced by the remote
func (d *ResultDetails) StatusCode() StatusCode {
	return StatusCode(d.code)
}

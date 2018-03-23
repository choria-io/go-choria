package v1

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-protocol/protocol"
)

// SecureReply contains 1 serialized Reply hashed
type secureReply struct {
	Protocol    string `json:"protocol"`
	MessageBody string `json:"message"`
	Hash        string `json:"hash"`

	mu sync.Mutex
}

// SetMessage sets the message contained in the Reply and updates the hash
func (r *secureReply) SetMessage(reply protocol.Reply) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not JSON encode reply message to store it in the Secure Reply: %s", err.Error())
		return
	}

	hash := sha256.Sum256([]byte(j))
	r.MessageBody = string(j)
	r.Hash = base64.StdEncoding.EncodeToString(hash[:])

	return
}

// Message retrieves the stored message content
func (r *secureReply) Message() string {
	return r.MessageBody
}

// Validates the body of the message by comparing the recorded hash with the hash of the body
func (r *secureReply) Valid() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	hash := sha256.Sum256([]byte(r.MessageBody))
	if base64.StdEncoding.EncodeToString(hash[:]) == r.Hash {
		validCtr.Inc()
		return true
	}

	invalidCtr.Inc()
	return false
}

// JSON creates a JSON encoded reply
func (r *secureReply) JSON() (body string, err error) {
	j, err := json.Marshal(r)
	if err != nil {
		protocolErrorCtr.Inc()
		err = fmt.Errorf("Could not JSON Marshal: %s", err.Error())
		return
	}

	body = string(j)

	if err = r.IsValidJSON(body); err != nil {
		err = fmt.Errorf("JSON produced from the SecureRequest does not pass validation: %s", err.Error())
		return
	}

	return
}

// Version retreives the protocol version for this message
func (r *secureReply) Version() string {
	return r.Protocol
}

// IsValidJSON validates the given JSON data against the schema
func (r *secureReply) IsValidJSON(data string) (err error) {
	if !protocol.ClientStrictValidation {
		return nil
	}

	_, errors, err := schemas.Validate(schemas.SecureReplyV1, data)
	if err != nil {
		err = fmt.Errorf("Could not validate SecureReply JSON data: %s", err.Error())
		return
	}

	if len(errors) != 0 {
		err = fmt.Errorf("Supplied JSON document is not a valid SecureReply message: %s", strings.Join(errors, ", "))
		return
	}

	return
}

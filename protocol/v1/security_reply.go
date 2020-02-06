package v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/choria-io/go-choria/protocol"
)

// SecureReply contains 1 serialized Reply hashed
type secureReply struct {
	Protocol    string `json:"protocol"`
	MessageBody string `json:"message"`
	Hash        string `json:"hash"`

	security SecurityProvider

	mu sync.Mutex
}

// SetMessage sets the message contained in the Reply and updates the hash
func (r *secureReply) SetMessage(reply protocol.Reply) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	j, err := reply.JSON()
	if err != nil {
		protocolErrorCtr.Inc()
		return fmt.Errorf("could not JSON encode reply message to store it in the Secure Reply: %s", err)
	}

	hash := r.security.ChecksumString(j)
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

	hash := r.security.ChecksumString(r.MessageBody)
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
		return "", fmt.Errorf("could not JSON Marshal: %s", err)
	}

	body = string(j)

	if err = r.IsValidJSON(body); err != nil {
		return "", fmt.Errorf("reply JSON produced from the SecureRequest does not pass validation: %s", err)
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
		return fmt.Errorf("could not validate SecureReply JSON data: %s", err)
	}

	if len(errors) != 0 {
		return fmt.Errorf("supplied JSON document is not a valid SecureReply message: %s", strings.Join(errors, ", "))
	}

	return
}

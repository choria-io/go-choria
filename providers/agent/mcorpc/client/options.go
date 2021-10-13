package client

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"time"

	"github.com/choria-io/go-choria/inter"
	"github.com/choria-io/go-choria/protocol"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/ddl/agent"
)

// RequestOptions are options for a RPC request
type RequestOptions struct {
	BatchSize        int
	BatchSleep       time.Duration
	Collective       string
	ConnectionName   string
	DiscoveryTimeout time.Duration
	Filter           *protocol.Filter
	Handler          Handler
	ProcessReplies   bool
	ProtocolVersion  string
	Replies          chan inter.ConnectorMessage
	ReplyTo          string
	RequestID        string
	RequestType      string
	Targets          []string
	Timeout          time.Duration
	Workers          int
	LimitSeed        int64
	LimitMethod      string
	LimitSize        string
	ReplyExprFilter  string
	DiscoveryStartCB DiscoveryStartFunc
	DiscoveryEndCB   DiscoveryEndFunc

	// merged of all batches
	totalStats *Stats

	// per batch
	stats *Stats

	fw inter.Framework
}

// DiscoveryStartFunc gets called before discovery starts
type DiscoveryStartFunc func()

// DiscoveryEndFunc gets called after discovery ends and include the discovered node count
// and what count of nodes will be targeted after limits were applied should this return
// error the RPC call will terminate
type DiscoveryEndFunc func(discovered int, limited int) error

// RequestOption is a function capable of setting an option
type RequestOption func(*RequestOptions)

// NewRequestOptions creates a initialized request options
func NewRequestOptions(fw inter.Framework, ddl *agent.DDL) (*RequestOptions, error) {
	rid, err := fw.NewRequestID()
	if err != nil {
		return nil, err
	}

	cfg := fw.Configuration()

	return &RequestOptions{
		fw:              fw,
		ProtocolVersion: protocol.RequestV1,
		RequestType:     inter.DirectRequestMessageType,
		Collective:      cfg.MainCollective,
		ProcessReplies:  true,
		Workers:         3,
		ConnectionName:  fmt.Sprintf("%s-mcorpc-%s", fw.Certname(), rid),
		stats:           NewStats(),
		totalStats:      NewStats(),
		LimitMethod:     cfg.RPCLimitMethod,
		LimitSeed:       time.Now().UnixNano(),
		Filter:          protocol.NewFilter(),

		// add discovery timeout to the agent timeout as that's basically an indication of
		// network overhead, discovery being the smallest possible RPC request it's an indication
		// of what peoples network behavior is like assuming discovery works
		Timeout:          (time.Duration(cfg.DiscoveryTimeout) * time.Second) + ddl.Timeout(),
		DiscoveryTimeout: time.Duration(cfg.DiscoveryTimeout) * time.Second,
	}, nil
}

// ConfigureMessage configures a pre-made message object based on the settings contained
func (o *RequestOptions) ConfigureMessage(msg inter.Message) (err error) {
	o.totalStats.RequestID = msg.RequestID()
	o.RequestID = msg.RequestID()

	switch o.RequestType {
	case inter.RequestMessageType, inter.DirectRequestMessageType:
		if o.RequestType == inter.RequestMessageType && o.BatchSize > 0 {
			return fmt.Errorf("batched mode requires %s mode", inter.DirectRequestMessageType)
		}

		if o.BatchSize == 0 {
			o.BatchSize = len(o.Targets)
		}

		msg.SetFilter(o.Filter)

		if len(o.Targets) > 0 {
			limited, err := o.limitTargets(o.Targets)
			if err != nil {
				return fmt.Errorf("could not limit targets: %s", err)
			}

			o.Targets = limited
			msg.SetDiscoveredHosts(limited)
		} else {
			limited, err := o.limitTargets(msg.DiscoveredHosts())
			if err != nil {
				return fmt.Errorf("could not limit targets: %s", err)
			}

			o.Targets = limited
		}

		o.totalStats.SetDiscoveredNodes(o.Targets)

	case inter.ServiceRequestMessageType:
		if len(o.Targets) > 0 {
			return fmt.Errorf("service requests does not support custom targets")
		}

		if !o.Filter.Empty() {
			return fmt.Errorf("service requests does not support filters")
		}

		msg.SetFilter(protocol.NewFilter())
		msg.SetDiscoveredHosts([]string{})
	}

	err = msg.SetType(o.RequestType)
	if err != nil {
		return err
	}

	msg.SetProtocolVersion(o.ProtocolVersion)

	stdtarget := msg.ReplyTarget()
	if o.ReplyTo == "" {
		o.ReplyTo = stdtarget
	}

	// the reply target is such that we'd probably not receive replies
	// so disable processing replies
	if stdtarget != o.ReplyTo {
		o.ProcessReplies = false
	}

	err = msg.SetReplyTo(o.ReplyTo)
	if err != nil {
		return err
	}

	err = msg.SetCollective(o.Collective)
	if err != nil {
		return err
	}

	// calculate a TTL for messages when we have batches and when using cached transports,
	// we need to avoid 2FA interactions for the full duration of the message:
	//
	// (TTL + DiscoveryTimeout + Timeout) * batches
	//
	// We have to allow TTL per batch since the last batch will get it much
	if msg.IsCachedTransport() && o.BatchSize != len(o.Targets) {
		batches := int(math.Ceil(float64(len(o.Targets)) / float64(o.BatchSize)))

		msg.SetTTL(batches * (msg.TTL() + int(o.DiscoveryTimeout.Seconds()) + int(o.Timeout.Seconds())))
		if msg.TTL() > int((5 * time.Hour).Seconds()) {
			return fmt.Errorf("cached transport TTL is unreasonably long")
		}
	}

	return nil
}

// Stats retrieves the stats for the completed request
func (o *RequestOptions) Stats() *Stats {
	return o.totalStats
}

// ReplyExprFilter filters reply by filter f, replies that match f will
// not be recorded and will not be passed to any handlers - they will
// count to received replies though as usual.
//
// When this filter matches a reply and a handler is set the handler will
// be called using a nil 'rpcreply' allowing the handler to process progress
// bars and more
func ReplyExprFilter(f string) RequestOption {
	return func(o *RequestOptions) {
		o.ReplyExprFilter = f
	}
}

// DiscoveryStartCB sets the function to be called before discovery starts
func DiscoveryStartCB(h DiscoveryStartFunc) RequestOption {
	return func(o *RequestOptions) {
		o.DiscoveryStartCB = h
	}
}

// DiscoveryEndCB sets the function to be called after discovery and node limiting
func DiscoveryEndCB(h DiscoveryEndFunc) RequestOption {
	return func(o *RequestOptions) {
		o.DiscoveryEndCB = h
	}
}

// ConnectionName sets the prefix used for various connection names
//
// Setting this when making many clients will minimize prometheus
// metrics being created - 2 or 3 per client which with random generated
// names will snowball over time
func ConnectionName(n string) RequestOption {
	return func(o *RequestOptions) {
		o.ConnectionName = n
	}
}

// Targets configures targets for a RPC request
func Targets(t []string) RequestOption {
	return func(o *RequestOptions) {
		o.Targets = t
	}
}

// Protocol sets the protocol version to use
func Protocol(v string) RequestOption {
	return func(o *RequestOptions) {
		o.ProtocolVersion = v
	}
}

// DirectRequest force the request to be a direct request
func DirectRequest() RequestOption {
	return func(o *RequestOptions) {
		o.RequestType = inter.DirectRequestMessageType
	}
}

// BroadcastRequest for the request to be a broadcast mode
//
// **NOTE:** You need to ensure you have filters etc done
func BroadcastRequest() RequestOption {
	return func(o *RequestOptions) {
		o.RequestType = inter.RequestMessageType
	}
}

// ServiceRequest for the request to be directed at a specific service agent
//
// **Note**: does not support filters or targets
func ServiceRequest() RequestOption {
	return func(o *RequestOptions) {
		o.RequestType = inter.ServiceRequestMessageType
	}
}

// Workers configures the amount of workers used to process responses
// this is ignored during batched mode as that is always done with a
// single worker
func Workers(w int) RequestOption {
	return func(o *RequestOptions) {
		o.Workers = w
	}
}

// Collective sets the collective to target a message at
func Collective(c string) RequestOption {
	return func(o *RequestOptions) {
		o.Collective = c
	}
}

// ReplyTo sets a custom reply to, else the connector will determine it
func ReplyTo(r string) RequestOption {
	return func(o *RequestOptions) {
		o.ReplyTo = r
		o.ProcessReplies = false
	}
}

// InBatches performs requests in batches
func InBatches(size int, sleep int) RequestOption {
	return func(o *RequestOptions) {
		o.BatchSize = size
		o.BatchSleep = time.Second * time.Duration(sleep)
		o.Workers = 1
	}
}

// Replies creates a custom channel for replies and will avoid processing them
func Replies(r chan inter.ConnectorMessage) RequestOption {
	return func(o *RequestOptions) {
		o.Replies = r
		o.ProcessReplies = false
	}
}

// Timeout configures the request timeout
func Timeout(t time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.Timeout = t
	}
}

// DiscoveryTimeout configures the request discovery timeout, defaults to configured discovery timeout
func DiscoveryTimeout(t time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.DiscoveryTimeout = t
	}
}

// Filter sets the filter, if its set discovery will be done prior to performing requests
func Filter(f *protocol.Filter) RequestOption {
	return func(o *RequestOptions) {
		o.Filter = f
	}
}

// ReplyHandler configures a callback to be called for each message received
func ReplyHandler(f Handler) RequestOption {
	return func(o *RequestOptions) {
		o.Handler = f
	}
}

// LimitMethod configures the method to use when limiting targets - "random" or "first"
func LimitMethod(m string) RequestOption {
	return func(o *RequestOptions) {
		o.LimitMethod = m
	}
}

// LimitSize sets limits on the targets, either a number of a percentage like "10%"
func LimitSize(s string) RequestOption {
	return func(o *RequestOptions) {
		o.LimitSize = s
	}
}

// LimitSeed sets the random seed used to select targets when limiting and limit method is "random"
func LimitSeed(s int64) RequestOption {
	return func(o *RequestOptions) {
		o.LimitSeed = s
	}
}

func (o *RequestOptions) shuffleLimitedTargets(targets []string) []string {
	if o.LimitMethod != "random" {
		return targets
	}

	var shuffler *rand.Rand

	if o.LimitSeed > -1 {
		shuffler = rand.New(rand.NewSource(o.LimitSeed))
	} else {
		shuffler = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	shuffler.Shuffle(len(targets), func(i, j int) { targets[i], targets[j] = targets[j], targets[i] })

	return targets
}

func (o *RequestOptions) limitTargets(targets []string) (limited []string, err error) {
	if !(o.LimitMethod == "random" || o.LimitMethod == "first") {
		return targets, fmt.Errorf("limit method '%s' is not valid, only 'random' or 'first' supported", o.LimitMethod)
	}

	if o.LimitSize == "" {
		limited = make([]string, len(targets))
		copy(limited, targets)

		return limited, nil
	}

	pctRe := regexp.MustCompile(`^(\d+)%$`)
	digitRe := regexp.MustCompile(`^(\d+)$`)

	count := 0

	if pctRe.MatchString(o.LimitSize) {
		// already know its a number and it has a matching substring
		pct, _ := strconv.Atoi(pctRe.FindStringSubmatch(o.LimitSize)[1])
		count = int(float64(len(targets)) * (float64(pct) / 100))
	} else if digitRe.MatchString(o.LimitSize) {
		// already know its a number
		count, _ = strconv.Atoi(o.LimitSize)
	} else {
		return limited, fmt.Errorf("could not parse limit as either number or percent")
	}

	if count <= 0 {
		return limited, fmt.Errorf("no targets left after applying target limits of '%s'", o.LimitSize)
	}

	limited = make([]string, count)
	copy(limited, o.shuffleLimitedTargets(targets))

	return limited, err
}

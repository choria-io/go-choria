// generated code; DO NOT EDIT

package executorclient

import (
	"time"

	coreclient "github.com/choria-io/go-choria/client/client"
	rpcclient "github.com/choria-io/go-choria/providers/agent/mcorpc/client"
)

// OptionReset resets the client options to use across requests to an empty list
func (p *ExecutorClient) OptionReset() *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = []rpcclient.RequestOption{}
	p.ns = p.clientOpts.ns
	p.targets = []string{}
	p.filters = []FilterFunc{
		FilterFunc(coreclient.AgentFilter("executor")),
	}

	return p
}

// OptionIdentityFilter adds an identity filter
func (p *ExecutorClient) OptionIdentityFilter(f ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, i := range f {
		if i == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.IdentityFilter(i)))
	}

	p.ns.Reset()

	return p
}

// OptionClassFilter adds a class filter
func (p *ExecutorClient) OptionClassFilter(f ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, i := range f {
		if i == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.ClassFilter(i)))
	}

	p.ns.Reset()

	return p
}

// OptionFactFilter adds a fact filter
func (p *ExecutorClient) OptionFactFilter(f ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, i := range f {
		if i == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.FactFilter(i)))
	}

	p.ns.Reset()

	return p
}

// OptionAgentFilter adds an agent filter
func (p *ExecutorClient) OptionAgentFilter(a ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, f := range a {
		if f == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.AgentFilter(f)))
	}

	p.ns.Reset()

	return p
}

// OptionCombinedFilter adds a combined filter
func (p *ExecutorClient) OptionCombinedFilter(f ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, i := range f {
		if i == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.CombinedFilter(i)))
	}

	p.ns.Reset()

	return p
}

// OptionCompoundFilter adds a compound filter
func (p *ExecutorClient) OptionCompoundFilter(f ...string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	for _, i := range f {
		if i == "" {
			continue
		}

		p.filters = append(p.filters, FilterFunc(coreclient.CompoundFilter(i)))
	}

	p.ns.Reset()

	return p
}

// OptionCollective sets the collective to target
func (p *ExecutorClient) OptionCollective(c string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.Collective(c))
	return p
}

// OptionInBatches performs requests in batches
func (p *ExecutorClient) OptionInBatches(size int, sleep int) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.InBatches(size, sleep))
	return p
}

// OptionDiscoveryTimeout configures the request discovery timeout, defaults to configured discovery timeout
func (p *ExecutorClient) OptionDiscoveryTimeout(t time.Duration) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.DiscoveryTimeout(t))
	return p
}

// OptionLimitMethod configures the method to use when limiting targets - "random" or "first"
func (p *ExecutorClient) OptionLimitMethod(m string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.LimitMethod(m))
	return p
}

// OptionLimitSize sets limits on the targets, either a number of a percentage like "10%"
func (p *ExecutorClient) OptionLimitSize(s string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.LimitSize(s))
	return p
}

// OptionLimitSeed sets the random seed used to select targets when limiting and limit method is "random"
func (p *ExecutorClient) OptionLimitSeed(s int64) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.LimitSeed(s))
	return p
}

// OptionTargets sets specific node targets which would avoid discovery for all action calls until reset
func (p *ExecutorClient) OptionTargets(t []string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.targets = t
	return p
}

// OptionWorkers sets how many worker connections should be started to the broker
func (p *ExecutorClient) OptionWorkers(w int) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.workers = w
	return p
}

// OptionExprFilter sets a filter expression that will remove results from the result set
func (p *ExecutorClient) OptionExprFilter(f string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.exprFilter = f
	return p
}

// OptionReplyTo sets a custom reply target
func (p *ExecutorClient) OptionReplyTo(t string) *ExecutorClient {
	p.Lock()
	defer p.Unlock()

	p.clientRPCOpts = append(p.clientRPCOpts, rpcclient.ReplyTo(t))
	p.noReplies = true
	p.clientOpts.progress = false

	return p
}

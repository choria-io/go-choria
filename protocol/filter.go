package protocol

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/choria-io/go-protocol/filter/agents"
	"github.com/choria-io/go-protocol/filter/classes"
	"github.com/choria-io/go-protocol/filter/facts"
	"github.com/choria-io/go-protocol/filter/identity"
)

// CompoundFilter is a mcollective compound filter
type CompoundFilter []map[string]interface{}

// CompoundFilters is a set of mcollective compound filters
type CompoundFilters []CompoundFilter

// FactFilter is how a fact match is represented to the Filter
type FactFilter struct {
	Fact     string `json:"fact"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// Filter is a MCollective filter
type Filter struct {
	Fact     []FactFilter    `json:"fact"`
	Class    []string        `json:"cf_class"`
	Agent    []string        `json:"agent"`
	Identity []string        `json:"identity"`
	Compound CompoundFilters `json:"compound"`

	mu sync.Mutex
}

// NewFilter creates a new empty filter
func NewFilter() *Filter {
	filter := &Filter{
		Fact:     []FactFilter{},
		Class:    []string{},
		Agent:    []string{},
		Identity: []string{},
		Compound: CompoundFilters{},
	}

	return filter
}

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// MatchRequest determines if a request matches the filter
func (f *Filter) MatchRequest(request Request, agents []string, identity string, classesFile string, factsFile string, log Logger) bool {
	filter, _ := request.Filter()
	passed := 0

	if filter.Empty() {
		log.Debugf("Matching request %s with empty filter", request.RequestID())
		return true
	}

	if len(filter.ClassFilters()) > 0 {
		if filter.MatchClassesFile(classesFile, log) {
			log.Debugf("Matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			return false
		}
	}

	if len(filter.AgentFilters()) > 0 {
		if filter.MatchAgents(agents) {
			log.Debugf("Matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			return false
		}
	}

	if len(filter.IdentityFilters()) > 0 {
		if filter.MatchIdentity(identity) {
			log.Debugf("Matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			return false
		}
	}

	if len(filter.FactFilters()) > 0 {
		if filter.MatchFactsFile(factsFile, log) {
			log.Debugf("Matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			return false
		}
	}

	if len(filter.CompoundFilters()) > 0 {
		log.Warnf("Compound filters are not supported, not matching request %s with filter '%#v'", request.RequestID(), filter.CompoundFilters())
		return false
	}

	return passed > 0
}

// MatchFactsFile determines if the filter would match a given set of facts found in a file
func (f *Filter) MatchFactsFile(file string, log Logger) bool {
	return facts.MatchFile(f.FactFilters(), file, log)
}

// MatchFacts determines if the filter would match a given set of facts found in given JSON data
func (f *Filter) MatchFacts(factsj json.RawMessage, log Logger) bool {
	return facts.MatchFacts(f.FactFilters(), factsj, log)
}

// MatchAgents determines if the filter would match a list of agents
func (f *Filter) MatchAgents(knownAgents []string) bool {
	return agents.Match(f.AgentFilters(), knownAgents)
}

// MatchIdentity determines if the filter would match a given identity
func (f *Filter) MatchIdentity(ident string) bool {
	return identity.Match(f.IdentityFilters(), ident)
}

// MatchClassesFile determines if the filter would match a list of classes
func (f *Filter) MatchClassesFile(file string, log Logger) bool {
	return classes.MatchFile(f.ClassFilters(), file, log)
}

// Empty determines if a filter is empty - that is all its contained filter arrays are empty
func (f *Filter) Empty() bool {
	if f.Fact == nil && f.Class == nil && f.Agent == nil && f.Identity == nil && f.Compound == nil {
		return true
	}

	if len(f.Fact) == 0 && len(f.Class) == 0 && len(f.Agent) == 0 && len(f.Identity) == 0 && len(f.Compound) == 0 {
		return true
	}

	return false
}

// CompoundFilters retrieve the list of compound filters
func (f *Filter) CompoundFilters() CompoundFilters {
	return f.Compound
}

// IdentityFilters retrieve the list of identity filters
func (f *Filter) IdentityFilters() []string {
	return f.Identity
}

// AgentFilters retrieve the list of agent filters
func (f *Filter) AgentFilters() []string {
	return f.Agent
}

// ClassFilters retrieve the list of class filters
func (f *Filter) ClassFilters() []string {
	return f.Class
}

// FactFilters retrieve the list of fact filters
func (f *Filter) FactFilters() [][3]string {
	var filter [][3]string
	filter = [][3]string{}

	for _, f := range f.Fact {
		filter = append(filter, [3]string{f.Fact, f.Operator, f.Value})
	}

	return filter
}

// AddCompoundFilter appends a filter to the compound filters,
// the filter should be a JSON string representing a valid mcollective
// compound filter as parsed by MCollective::Matcher.create_compound_callstack
func (f *Filter) AddCompoundFilter(query string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cf CompoundFilter
	err := json.Unmarshal([]byte(query), &cf)
	if err != nil {
		return fmt.Errorf("could not parse query as JSON: %s", err)
	}

	f.Compound = append(f.Compound, cf)

	return nil
}

// AddIdentityFilter appends a filter to the identity filters
func (f *Filter) AddIdentityFilter(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.contains(id, f.Identity) {
		f.Identity = append(f.Identity, id)
	}
}

// AddAgentFilter appends a filter to the agent filters
func (f *Filter) AddAgentFilter(agent string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.contains(agent, f.Agent) {
		f.Agent = append(f.Agent, agent)
	}
}

// AddClassFilter appends a filter to the class filters
func (f *Filter) AddClassFilter(class string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.contains(class, f.Class) {
		f.Class = append(f.Class, class)
	}
}

// AddFactFilter appends a filter to the fact filters
func (f *Filter) AddFactFilter(fact string, operator string, value string) (err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.contains(operator, []string{">=", "<=", "<", ">", "!=", "==", "=~"}) {
		err = fmt.Errorf("%s is not a valid fact operator", operator)
		return
	}

	filter := FactFilter{
		Fact:     fact,
		Operator: operator,
		Value:    value,
	}

	f.Fact = append(f.Fact, filter)

	return
}

func (f *Filter) contains(needle string, haystack []string) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}

	return false
}

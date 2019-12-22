package protocol

import (
	"encoding/json"
	"fmt"
	"sync"
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

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
func (self *Filter) Empty() bool {
	if self.Fact == nil && self.Class == nil && self.Agent == nil && self.Identity == nil && self.Compound == nil {
		return true
	}

	if len(self.Fact) == 0 && len(self.Class) == 0 && len(self.Agent) == 0 && len(self.Identity) == 0 && len(self.Compound) == 0 {
		return true
	}

	return false
}

// CompoundFilters retrieve the list of compound filters
func (self *Filter) CompoundFilters() CompoundFilters {
	return self.Compound
}

// IdentityFilters retrieve the list of identity filters
func (self *Filter) IdentityFilters() []string {
	return self.Identity
}

// AgentFilters retrieve the list of agent filters
func (self *Filter) AgentFilters() []string {
	return self.Agent
}

// ClassFilters retrieve the list of class filters
func (self *Filter) ClassFilters() []string {
	return self.Class
}

// FactFilters retrieve the list of fact filters
func (self *Filter) FactFilters() [][3]string {
	var filter [][3]string
	filter = [][3]string{}

	for _, f := range self.Fact {
		filter = append(filter, [3]string{f.Fact, f.Operator, f.Value})
	}

	return filter
}

// AddCompoundFilter appends a filter to the compound filters,
// the filter should be a JSON string representing a valid mcollective
// compound filter as parsed by MCollective::Matcher.create_compound_callstack
func (self *Filter) AddCompoundFilter(query string) error {
	self.mu.Lock()
	defer self.mu.Unlock()

	var f CompoundFilter
	err := json.Unmarshal([]byte(query), &f)
	if err != nil {
		return fmt.Errorf("could not parse query as JSON: %s", err)
	}

	self.Compound = append(self.Compound, f)

	return nil
}

// AddIdentityFilter appends a filter to the identity filters
func (self *Filter) AddIdentityFilter(id string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.contains(id, self.Identity) {
		self.Identity = append(self.Identity, id)
	}
}

// AddAgentFilter appends a filter to the agent filters
func (self *Filter) AddAgentFilter(agent string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.contains(agent, self.Agent) {
		self.Agent = append(self.Agent, agent)
	}
}

// AddClassFilter appends a filter to the class filters
func (self *Filter) AddClassFilter(class string) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.contains(class, self.Class) {
		self.Class = append(self.Class, class)
	}
}

// AddFactFilter appends a filter to the fact filters
func (self *Filter) AddFactFilter(fact string, operator string, value string) (err error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if !self.contains(operator, []string{">=", "<=", "<", ">", "!=", "==", "=~"}) {
		err = fmt.Errorf("%s is not a valid fact operator", operator)
		return
	}

	filter := FactFilter{
		Fact:     fact,
		Operator: operator,
		Value:    value,
	}

	self.Fact = append(self.Fact, filter)

	return
}

func (self *Filter) contains(needle string, haystack []string) bool {
	for _, i := range haystack {
		if i == needle {
			return true
		}
	}

	return false
}

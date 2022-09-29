// Copyright (c) 2017-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package protocol

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/choria-io/go-choria/filter/agents"
	"github.com/choria-io/go-choria/filter/classes"
	"github.com/choria-io/go-choria/filter/compound"
	"github.com/choria-io/go-choria/filter/facts"
	"github.com/choria-io/go-choria/filter/identity"
	"github.com/choria-io/go-choria/providers/data/ddl"
)

// FactFilter is how a fact match is represented to the Filter
type FactFilter struct {
	Fact     string `json:"fact"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// note compound filter structure is not ideal at all, but this is
// the structure old compound filters had and will let us not fail
// on older machines as json validation errors, and it's actually not
// bad to store expr based compounds in its own key in case we later
// support other types of filter

// Filter is a Choria filter
type Filter struct {
	Fact     []FactFilter          `json:"fact"`
	Class    []string              `json:"cf_class"`
	Agent    []string              `json:"agent"`
	Identity []string              `json:"identity"`
	Compound [][]map[string]string `json:"compound"`

	mu sync.Mutex
}

// NewFilter creates a new empty filter
func NewFilter() *Filter {
	filter := &Filter{
		Fact:     []FactFilter{},
		Class:    []string{},
		Agent:    []string{},
		Identity: []string{},
		Compound: [][]map[string]string{},
	}

	return filter
}

// Logger provides logging facilities
type Logger interface {
	Warnf(format string, args ...any)
	Debugf(format string, args ...any)
	Errorf(format string, args ...any)
}

type ServerInfoSource interface {
	Classes() []string
	Facts() json.RawMessage
	Identity() string
	KnownAgents() []string
	DataFuncMap() (ddl.FuncMap, error)
}

func (f *Filter) MatchServerRequest(request Request, si ServerInfoSource, log Logger) bool {
	filter, _ := request.Filter()
	passed := 0

	if filter.Empty() {
		log.Debugf("Matching request %s with empty filter", request.RequestID())
		return true
	}

	if len(filter.IdentityFilters()) > 0 {
		if filter.MatchIdentity(si.Identity()) {
			log.Debugf("Matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with identity filters '%#v'", request.RequestID(), filter.IdentityFilters())
			return false
		}
	}

	if len(filter.AgentFilters()) > 0 {
		if filter.MatchAgents(si.KnownAgents()) {
			log.Debugf("Matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with agent filters '%#v'", request.RequestID(), filter.AgentFilters())
			return false
		}
	}

	if len(filter.ClassFilters()) > 0 {
		if filter.MatchClasses(si.Classes(), log) {
			log.Debugf("Matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s with class filters '%#v'", request.RequestID(), filter.ClassFilters())
			return false
		}
	}

	if len(filter.FactFilters()) > 0 {
		if filter.MatchFacts(si.Facts(), log) {
			log.Debugf("Matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s based on fact filters '%#v'", request.RequestID(), filter.FactFilters())
			return false
		}
	}

	if len(filter.CompoundFilters()) > 0 {
		df, err := si.DataFuncMap()
		if err != nil {
			log.Errorf("Cannot resolve data functions map: %s", err)
		}
		if filter.MatchCompound(si.Facts(), si.Classes(), si.KnownAgents(), df, log) {
			log.Debugf("Matching request %s based on compound filter %#v", request.RequestID(), filter.CompoundFilters())
			passed++
		} else {
			log.Debugf("Not matching request %s based on compound filter %#v", request.RequestID(), filter.CompoundFilters())
			return false
		}
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

// MatchClasses determines if the filter would match against the list of classes
func (f *Filter) MatchClasses(knownClasses []string, _ Logger) bool {
	return classes.Match(f.ClassFilters(), knownClasses)
}

// MatchCompoundFiles determines if the filter would match against classes, facts and agents using an expr expression
func (f *Filter) MatchCompoundFiles(factsFile string, classesFile string, knownAgents []string, log Logger) bool {
	return compound.MatchExprStringFiles(f.CompoundFilters(), factsFile, classesFile, knownAgents, log)
}

// MatchCompound determines if the filter would match against classes, facts and agents using an expr expression
func (f *Filter) MatchCompound(facts json.RawMessage, knownClasses []string, knownAgents []string, fm ddl.FuncMap, log Logger) bool {
	return compound.MatchExprString(f.CompoundFilters(), facts, knownClasses, knownAgents, fm, log)
}

// Empty determines if a filter is empty - that is all its contained filter arrays are empty
func (f *Filter) Empty() bool {
	// NOTE: Do not make empty return true when there is only 1 agent filter set, this breaks a bunch of things.

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.Fact == nil && f.Class == nil && f.Agent == nil && f.Identity == nil && f.Compound == nil {
		return true
	}

	if len(f.Fact) == 0 && len(f.Class) == 0 && len(f.Agent) == 0 && len(f.Identity) == 0 && len(f.Compound) == 0 {
		return true
	}

	return false
}

// CompoundFilters retrieve the list of compound filters
func (f *Filter) CompoundFilters() [][]map[string]string {
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

// AddCompoundFilter appends a filter to the compound filters, the filter should be an expr string representing a valid choria filter
func (f *Filter) AddCompoundFilter(query string) error {
	if query == "" {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var cf = []map[string]string{{"expr": query}}

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
	if agent == "" {
		return
	}

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

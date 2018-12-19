package config

import (
	"sync"
)

// Mutator is a function that can mutate the configuration
type Mutator interface {
	Mutate(*Config)
}

var mutators = []Mutator{}
var mutatorNames = []string{}
var mu = &sync.Mutex{}

// RegisterMutator registers a new configuration mutator
func RegisterMutator(name string, m Mutator) {
	mu.Lock()
	defer mu.Unlock()

	mutators = append(mutators, m)
	mutatorNames = append(mutatorNames, name)
}

// MutatorNames are the names of known configuration mutators
func MutatorNames() []string {
	return mutatorNames
}

func mutate(c *Config) {
	mu.Lock()
	defer mu.Unlock()

	for _, mutator := range mutators {
		mutator.Mutate(c)
	}
}

package config

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Mutator is a function that can mutate the configuration
type Mutator interface {
	Mutate(*Config, *logrus.Entry)
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

// Mutate calls all registered mutators on the given configuration
func Mutate(c *Config, log *logrus.Entry) {
	mu.Lock()
	defer mu.Unlock()

	for _, mutator := range mutators {
		mutator.Mutate(c, log)
	}
}

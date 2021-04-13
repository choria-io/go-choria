// +build !windows

package cmd

import (
	"fmt"
	"sync"
)

func (r *serverRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	if len(c.BuildInfo().AgentProviders()) == 0 {
		return fmt.Errorf("invalid Choria Server build, no agent providers present")
	}

	instance, err := r.prepareInstance()
	if err != nil {
		return err
	}

	wg.Add(1)
	return instance.Run(ctx, wg)
}

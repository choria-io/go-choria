// +build !windows

package cmd

import (
	"sync"
)

func (r *serverRunCommand) Run(wg *sync.WaitGroup) (err error) {
	defer wg.Done()

	instance, err := r.prepareInstance()
	if err != nil {
		return err
	}

	wg.Add(1)
	return instance.Run(ctx, wg)
}

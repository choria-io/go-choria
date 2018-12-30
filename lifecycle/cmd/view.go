package cmd

import (
	lifecycle "github.com/choria-io/go-lifecycle"
)

func view() error {
	return lifecycle.View(ctx, &lifecycle.ViewOptions{
		Choria:          fw,
		ComponentFilter: componentFilter,
		TypeFilter:      typeFilter,
		Debug:           debug,
	})
}

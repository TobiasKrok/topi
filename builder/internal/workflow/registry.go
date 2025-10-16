package workflow

import (
	"fmt"

	sharedworkflow "github.com/tobiaskrok/topi/shared/workflow"
)

//TODO: some steps can only be ran once

type StepFactory func(cfg sharedworkflow.StepConfig) (Step, error)

var (
	stepRegistry = make(map[string]StepFactory)
)

func RegisterStep(stepType string, factory StepFactory) {
	stepRegistry[stepType] = factory
}

func GetStepFactory(stepType string) (StepFactory, bool) {
	factory, ok := stepRegistry[stepType]
	return factory, ok
}

// TODO: allow empty "uses" and just use "run" to run a shell script
func NewStep(cfg sharedworkflow.StepConfig) (Step, error) {
	if cfg.Uses == "" && cfg.Run == "" {
		return nil, fmt.Errorf("Step %s has no action, provide one in the 'uses' field", cfg.Name)
	}

	// so you can just use 'run: echo Hello' instead of specyfing shell
	step := cfg.Uses
	if cfg.Uses == "" && cfg.Run != "" {
		step = "shell"
	}
	factory, ok := GetStepFactory(step)
	if !ok {
		return nil, fmt.Errorf("Step %s has unknown action: '%s'", cfg.Name, cfg.Uses)
	}

	return factory(cfg)
}

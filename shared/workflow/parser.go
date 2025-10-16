package workflow

import "gopkg.in/yaml.v3"

func ParseWorkflow(in []byte) (WorkflowConfig, error) {

	var workflow WorkflowConfig
	err := yaml.Unmarshal(in, &workflow)
	if err != nil {
		return WorkflowConfig{}, err
	}
	return workflow, nil
}

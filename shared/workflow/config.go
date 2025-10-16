package workflow

type WorkflowConfig struct {
	Version string               `yaml:"version"`
	Name    string               `yaml:"name"`
	Jobs    map[string]JobConfig `yaml:"jobs"`
	Env     map[string]string    `yaml:"env,omitempty"`
}

type JobConfig struct {
	Needs       string            `yaml:"needs"`
	Steps       []StepConfig      `yaml:"steps"`
	ErrorPolicy string            `yaml:"errorPolicy,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

type StepConfig struct {
	Name        string            `yaml:"name,omitempty"`
	Description string            `yaml:"description,omitempty"`
	Uses        string            `yaml:"uses,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Params      map[string]string `yaml:"params,omitempty"`
	Run         string            `yaml:"run,omitempty"`
}

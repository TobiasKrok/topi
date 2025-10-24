package workflow

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
)

// Regex patterns for environment variable parsing
var (
	// Matches valid environment variable names (must start with letter or underscore, then alphanumeric or underscore)
	envNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// Matches environment variable assignment in format: KEY=VALUE or KEY="VALUE" or KEY='VALUE'
	// Captures: 1=key, 2=unquoted value, 3=double-quoted value, 4=single-quoted value
	envLineRegex = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(?:([^"'\s].*?)|"([^"]*)"|'([^']*)')?\s*$`)
)

type EnvironmentManager struct {
	workflowFile string
	env          map[string]string
}

func NewEnvironmentManager(systemDir string, buildID string) (*EnvironmentManager, error) {
	envDir := path.Join(systemDir, "env", buildID)
	if err := os.MkdirAll(envDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create env directory: %w", err)
	}

	workflowFile := path.Join(envDir, "workflow.env")

	if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
		file, err := os.Create(workflowFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create workflow env file: %w", err)
		}
		file.Close()
	}

	if err := os.Setenv("TOPI_ENV", workflowFile); err != nil {
		return nil, err
	}

	return &EnvironmentManager{
		workflowFile: workflowFile,
		env:          make(map[string]string),
	}, nil
}

func (e *EnvironmentManager) Set(key, value string) error {
	e.env[key] = value
	err := os.Setenv(key, value)
	return err
}

func (e *EnvironmentManager) Get(key string) (string, error) {

	if val, ok := e.env[key]; ok && val != "" {
		return val, nil
	}
	// if val, ok := ctx.Secrets[key]; ok {
	//     return val
	// }

	if val, exists := os.LookupEnv(key); exists {
		return val, nil
	}

	return "", fmt.Errorf("environment variable %s not found", key)
}

func (e *EnvironmentManager) Load() error {
	content, err := os.ReadFile(e.workflowFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read workflow env file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, err := e.parseEnvLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum+1, err)
		}

		err = e.Set(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *EnvironmentManager) parseEnvLine(line string) (string, string, error) {
	matches := envLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return "", "", fmt.Errorf("invalid environment variable format: %s", line)
	}

	key := matches[1]

	if !e.validateEnvName(key) {
		return "", "", fmt.Errorf("invalid environment variable name: %s", key)
	}

	var value string
	if matches[2] != "" {
		// Unquoted value
		value = strings.TrimSpace(matches[2])
	} else if matches[3] != "" {
		// Double-quoted value
		value = matches[3]
	} else if matches[4] != "" {
		// Single-quoted value
		value = matches[4]
	}

	return key, value, nil
}

func (e *EnvironmentManager) validateEnvName(name string) bool {
	return envNameRegex.MatchString(name)
}

func (e *EnvironmentManager) ExpandString(input string) string {
	return os.Expand(input, func(key string) string {
		// Check sources in priority order
		if val, ok := e.env[key]; ok {
			return val
		}
		// if val, ok := ctx.Secrets[key]; ok {
		//     return val
		// }
		if val := os.Getenv(key); val != "" {
			return val
		}
		return "${" + key + "}"
	})
}

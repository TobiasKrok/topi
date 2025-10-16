package utils

import "gopkg.in/yaml.v3"

func ToStruct[T any](m map[string]string) (T, error) {

	var result T
	// Convert to YAML then back to struct
	data, err := yaml.Marshal(m)
	if err != nil {
		return result, err
	}

	err = yaml.Unmarshal(data, &result)
	return result, err
}

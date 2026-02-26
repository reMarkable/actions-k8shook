// Package types defines the data structures used for configuring and managing GitHub Actions runners in a containerized environment.
package types

import (
	"encoding/json"
)

type ContainerHookInput struct {
	Args         InputArgs `json:"args"`
	Command      string    `json:"command"`
	ResponseFile string
	State        map[string]string
}

type InputArgs struct {
	ContainerDefinition
	Container   ContainerDefinition `json:"container"`
	ServicesRaw json.RawMessage     `json:"services"`
	Services    []ServiceDefinition `json:"-"`
}

// UnmarshalJSON handles GitHub Actions service format where contextName is derived from image
func (ia *InputArgs) UnmarshalJSON(data []byte) error {
	type Alias InputArgs
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ia),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if err := json.Unmarshal(data, &ia.ContainerDefinition); err != nil {
		return err
	}

	if len(ia.ServicesRaw) > 0 {
		var servicesArray []ServiceDefinition
		if err := json.Unmarshal(ia.ServicesRaw, &servicesArray); err != nil {
			return err
		}

		for i := range servicesArray {
			if servicesArray[i].ContextName == "" && servicesArray[i].Image != "" {
				servicesArray[i].ContextName = generateContainerName(servicesArray[i].Image)
			}
		}
		ia.Services = servicesArray
	}

	return nil
}

// generateContainerName extracts a container name from an image string.
// Examples:
//   - "redis:7" -> "redis"
//   - "docker.io/library/postgres:14" -> "postgres"
//   - "ghcr.io/user/myimage:latest" -> "myimage"
func generateContainerName(image string) string {
	parts := []rune(image)
	lastSlash := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == '/' {
			lastSlash = i
			break
		}
	}

	nameWithTag := image
	if lastSlash >= 0 {
		nameWithTag = image[lastSlash+1:]
	}

	for i, ch := range nameWithTag {
		if ch == ':' {
			return nameWithTag[:i]
		}
	}

	return nameWithTag
}

type ContainerDefinition struct {
	CreateOptions        string            `json:"createOptions"`
	Dockerfile           string            `json:"dockerfile"`
	Entrypoint           string            `json:"entryPoint"`
	EntrypointArgs       []string          `json:"entryPointArgs"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	PrependPath          []string          `json:"prependPath"`
	WorkingDirectory     string            `json:"workingDirectory"`
	Image                string            `json:"image"`
	PortMappings         []map[int]int     `json:"portMappings"`
	Registry             map[string]string `json:"registry"`
	SystemMountVolumes   []MountVolume     `json:"systemMountVolumes"`
	UserMountVolumes     []MountVolume     `json:"userMountVolumes"`
}

type ServiceDefinition struct {
	ContextName          string            `json:"contextName"`
	Image                string            `json:"image"`
	Entrypoint           string            `json:"entryPoint"`
	EntrypointArgs       []string          `json:"entryPointArgs"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	WorkingDirectory     string            `json:"workingDirectory"`
	PortMappings         []string          `json:"portMappings"`
	Registry             map[string]string `json:"registry"`
	SystemMountVolumes   []MountVolume     `json:"systemMountVolumes"`
	UserMountVolumes     []MountVolume     `json:"userMountVolumes"`
	CreateOptions        string            `json:"createOptions"`
}

type MountVolume struct {
	ReadOnly          bool
	SourceVolumePath  string
	TargetVolumePath  string
	UserProvidedValue any
}

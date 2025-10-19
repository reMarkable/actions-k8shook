// Package types defines the data structures used for configuring and managing GitHub Actions runners in a containerized environment.
package types

type ContainerHookInput struct {
	Args         InputArgs `json:"args"`
	Command      string    `json:"command"`
	ResponseFile string
	State        map[string]string
}

type InputArgs struct {
	ContainerDefinition
	Container ContainerDefinition `json:"container"`
	Services  []map[string]any    `json:"services"`
}

type ContainerDefinition struct {
	CreateOptions        map[string]any    `json:"createOptions"`
	Dockerfile           string            `json:"dockerfile"`
	Entrypoint           string            `json:"entryPoint"`
	EntrypointArgs       []string          `json:"entryPointArgs"`
	EnvironmentVariables map[string]string `json:"environmentVariables"`
	PrependPath          []string          `json:"prependPath"`
	WorkingDirectory     string            `json:"workingDirectory"`
	Image                string            `json:"image"`
	PortMappings         []map[int]int     `json:"portMappings"`
	Registry             string            `json:"registry"`
	SystemMountVolumes   []MountVolume     `json:"systemMountVolumes"`
	UserMountVolumes     []MountVolume     `json:"userMountVolumes"`
}

type MountVolume struct {
	ReadOnly          bool
	SourceVolumePath  string
	TargetVolumePath  string
	UserProvidedValue any
}

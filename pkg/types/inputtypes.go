// Package types defines the data structures used for configuring and managing GitHub Actions runners in a containerized environment.
package types

type ContainerHookInput struct {
	Args         InputArgs
	Command      string
	ResponseFile string
	State        map[string]string
}

type InputArgs struct {
	Container ContainerDefinition
	Services  []map[string]any
}

type ContainerDefinition struct {
	CreateOptions       map[string]any
	Dockerfile          string
	Entrypoint          string
	EntrypointArgs      []string
	EntrypointVariables map[string]string
	Image               string
	PortMappings        []map[int]int
	Registry            string
	SystemMountVolumes  []MountVolume
	UserMountVolumes    []MountVolume
	WorkingDirectory    string
}

type MountVolume struct {
	ReadOnly          bool
	SourceVolumePath  string
	TargetVolumePath  string
	UserProvidedValue any
}

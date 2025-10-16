package types

type ResponseType struct {
	State     ResponseState
	Container ContainerInfo
	IsAlpine  bool
}

type ResponseState struct {
	JobPod string
}

type ContainerInfo struct {
	Image string
	Ports map[int]int
}

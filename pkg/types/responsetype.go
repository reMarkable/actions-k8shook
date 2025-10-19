package types

type ResponseType struct {
	Context  map[string]ContainerInfo `json:"context"`
	State    ResponseState            `json:"state"`
	IsAlpine bool                     `json:"isAlpine"`
}

type ResponseState struct {
	JobPod string `json:"jobPod"`
}

type ContainerInfo struct {
	Image string      `json:"image"`
	Ports map[int]int `json:"ports"`
}

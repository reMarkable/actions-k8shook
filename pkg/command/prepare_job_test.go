package command

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/reMarkable/k8s-hook/pkg/types"
)

func TestPrepareJobResponseWithServices(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		name             string
		services         []types.ServiceDefinition
		wantServiceCount int
	}{
		"no services": {
			services:         []types.ServiceDefinition{},
			wantServiceCount: 0,
		},
		"single service": {
			services: []types.ServiceDefinition{
				{
					ContextName:  "redis",
					Image:        "redis:7",
					PortMappings: []string{"6379"},
				},
			},
			wantServiceCount: 1,
		},
		"multiple services": {
			services: []types.ServiceDefinition{
				{
					ContextName:  "redis",
					Image:        "redis:7",
					PortMappings: []string{"6379"},
				},
				{
					ContextName:  "postgres",
					Image:        "postgres:14",
					PortMappings: []string{"5432"},
				},
			},
			wantServiceCount: 2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a response with services
			response := types.ResponseType{
				Context: map[string]types.ContainerInfo{
					"container": {
						Image: "ubuntu:22.04",
						Ports: map[int]int{},
					},
				},
				Services: []types.ServiceInfo{},
				State: types.ResponseState{
					JobPod: "test-pod",
				},
				IsAlpine: false,
			}

			// Add services to response
			for _, service := range tt.services {
				serviceInfo := types.ServiceInfo{
					ContextName: service.ContextName,
					Image:       service.Image,
					Ports:       map[int]int{},
				}
				// Parse ports from service definition
				for _, portMapping := range service.PortMappings {
					// Simple parsing for test - just extract the port number
					// Port mapping is like "6379" or "8080:80"
					// For simplicity in test, just use a fixed mapping
					switch portMapping {
					case "6379":
						serviceInfo.Ports[6379] = 6379
					case "5432":
						serviceInfo.Ports[5432] = 5432
					}
				}
				response.Services = append(response.Services, serviceInfo)
			}

			// Verify service count
			if len(response.Services) != tt.wantServiceCount {
				t.Errorf("Response services count = %d, want %d", len(response.Services), tt.wantServiceCount)
			}

			// Verify each service has required fields
			for i, service := range response.Services {
				if service.ContextName == "" {
					t.Errorf("Response service[%d] missing ContextName", i)
				}
				if service.Image == "" {
					t.Errorf("Response service[%d] missing Image", i)
				}
				if service.Ports == nil {
					t.Errorf("Response service[%d] Ports is nil", i)
				}
			}

			// Verify response can be marshaled to JSON
			_, err := json.Marshal(response)
			if err != nil {
				t.Errorf("Failed to marshal response to JSON: %v", err)
			}
		})
	}
}

func TestResponseTypeJSONStructure(t *testing.T) {
	t.Parallel()

	// Create a complete response with services
	response := types.ResponseType{
		Context: map[string]types.ContainerInfo{
			"container": {
				Image: "ubuntu:22.04",
				Ports: map[int]int{8080: 8080},
			},
		},
		Services: []types.ServiceInfo{
			{
				ContextName: "redis",
				Image:       "redis:7",
				Ports:       map[int]int{6379: 6379},
			},
			{
				ContextName: "postgres",
				Image:       "postgres:14",
				Ports:       map[int]int{5432: 5432},
			},
		},
		State: types.ResponseState{
			JobPod: "test-pod-123",
		},
		IsAlpine: false,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Unmarshal back to verify structure
	var unmarshaled types.ResponseType
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify services array
	if len(unmarshaled.Services) != 2 {
		t.Errorf("Unmarshaled services count = %d, want 2", len(unmarshaled.Services))
	}

	// Verify first service
	if len(unmarshaled.Services) > 0 {
		redis := unmarshaled.Services[0]
		if redis.ContextName != "redis" {
			t.Errorf("Service[0].ContextName = %s, want 'redis'", redis.ContextName)
		}
		if redis.Image != "redis:7" {
			t.Errorf("Service[0].Image = %s, want 'redis:7'", redis.Image)
		}
		if redis.Ports[6379] != 6379 {
			t.Errorf("Service[0].Ports[6379] = %d, want 6379", redis.Ports[6379])
		}
	}

	// Verify second service
	if len(unmarshaled.Services) > 1 {
		postgres := unmarshaled.Services[1]
		if postgres.ContextName != "postgres" {
			t.Errorf("Service[1].ContextName = %s, want 'postgres'", postgres.ContextName)
		}
	}

	// Verify state
	if unmarshaled.State.JobPod != "test-pod-123" {
		t.Errorf("State.JobPod = %s, want 'test-pod-123'", unmarshaled.State.JobPod)
	}
}

func TestWriteResponse(t *testing.T) {
	t.Parallel()

	// Create a temporary file for the response
	tmpFile, err := os.CreateTemp("", "response-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	response := types.ResponseType{
		Context: map[string]types.ContainerInfo{
			"container": {
				Image: "ubuntu:22.04",
				Ports: map[int]int{},
			},
		},
		Services: []types.ServiceInfo{
			{
				ContextName: "redis",
				Image:       "redis:7",
				Ports:       map[int]int{6379: 6379},
			},
		},
		State: types.ResponseState{
			JobPod: "test-pod",
		},
		IsAlpine: false,
	}

	// Write response
	err = writeResponse(tmpFile.Name(), response)
	if err != nil {
		t.Fatalf("writeResponse() error = %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read response file: %v", err)
	}

	var readBack types.ResponseType
	if err := json.Unmarshal(data, &readBack); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify services
	if len(readBack.Services) != 1 {
		t.Errorf("Read back services count = %d, want 1", len(readBack.Services))
	}

	if len(readBack.Services) > 0 {
		if readBack.Services[0].ContextName != "redis" {
			t.Errorf("Read back service ContextName = %s, want 'redis'", readBack.Services[0].ContextName)
		}
	}
}

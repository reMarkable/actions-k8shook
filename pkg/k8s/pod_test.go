package k8s

import (
	"os"
	"strings"
	"testing"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sClient_waitForPodReady(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		want    v1.PodPhase
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotErr := c.waitForPodReady(name)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("waitForPodReady() failed: %v", gotErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("waitForPodReady() succeeded unexpectedly")
			}
		})
	}
}

func TestK8sClient_CreatePodSpec(t *testing.T) {
	t.Parallel()
	os.Setenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE", "default")
	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}
	input := types.ContainerDefinition{
		Image: "example-image",
		EnvironmentVariables: map[string]string{
			"GITHUB_WORKSPACE": "/tmp/workspace",
		},
	}
	pod := c.preparePodSpec(input, nil, PodTypeJob)
	if pod.Spec.Containers[0].Image != "example-image" {
		t.Errorf("expected image 'example-image', got '%s'", pod.Spec.Containers[0].Image)
	}
	expectedPaths := []string{"/__e", "/__w", "/github/home", "/github/workflow"}
	volumes := pod.Spec.Containers[0].VolumeMounts
	for i, vol := range volumes {
		if vol.MountPath != expectedPaths[i] {
			t.Errorf("job expected mount path '%s', got '%s'", expectedPaths[i], vol.MountPath)
		}
	}
	jobPod := c.preparePodSpec(input, nil, PodTypeContainerStep)
	expectedJobPaths := []string{"/github/workspace", "/github/file_commands", "/__w", "/github/home", "/github/workflow"}
	jobVolumes := jobPod.Spec.Containers[0].VolumeMounts
	for i, vol := range jobVolumes {
		if vol.MountPath != expectedJobPaths[i] {
			t.Errorf("job expected mount path '%s', got '%s'", expectedJobPaths[i], vol.MountPath)
		}
	}
}

func TestParsePortMappings(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		name        string
		mappings    []string
		wantPorts   []v1.ContainerPort
		wantErr     bool
		errContains string
	}{
		"single port": {
			mappings: []string{"80"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"host:container format": {
			mappings: []string{"8080:80"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"port with tcp protocol": {
			mappings: []string{"80/tcp"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"port with TCP protocol uppercase": {
			mappings: []string{"80/TCP"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"port with udp protocol": {
			mappings: []string{"53/udp"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 53, Protocol: v1.ProtocolUDP},
			},
			wantErr: false,
		},
		"port with sctp protocol": {
			mappings: []string{"9999/sctp"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 9999, Protocol: v1.ProtocolSCTP},
			},
			wantErr: false,
		},
		"host:container with protocol": {
			mappings: []string{"8080:80/tcp"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"multiple ports": {
			mappings: []string{"80", "443", "8080:3000"},
			wantPorts: []v1.ContainerPort{
				{ContainerPort: 80, Protocol: v1.ProtocolTCP},
				{ContainerPort: 443, Protocol: v1.ProtocolTCP},
				{ContainerPort: 3000, Protocol: v1.ProtocolTCP},
			},
			wantErr: false,
		},
		"invalid protocol": {
			mappings:    []string{"80/xyz"},
			wantErr:     true,
			errContains: "unsupported protocol",
		},
		"invalid port number": {
			mappings:    []string{"invalid"},
			wantErr:     true,
			errContains: "invalid port number",
		},
		"port too high": {
			mappings:    []string{"99999"},
			wantErr:     true,
			errContains: "port out of range",
		},
		"port zero": {
			mappings:    []string{"0"},
			wantErr:     true,
			errContains: "port out of range",
		},
		"negative port": {
			mappings:    []string{"-1"},
			wantErr:     true,
			errContains: "port out of range",
		},
		"too many colons": {
			mappings:    []string{"8080:80:443"},
			wantErr:     true,
			errContains: "invalid port mapping format",
		},
		"empty string": {
			mappings:    []string{""},
			wantErr:     true,
			errContains: "invalid port number",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			gotPorts, err := parsePortMappings(tt.mappings)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parsePortMappings() expected error containing '%s', got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parsePortMappings() error = %v, want error containing '%s'", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("parsePortMappings() unexpected error = %v", err)
				return
			}
			if len(gotPorts) != len(tt.wantPorts) {
				t.Errorf("parsePortMappings() got %d ports, want %d", len(gotPorts), len(tt.wantPorts))
				return
			}
			for i, got := range gotPorts {
				want := tt.wantPorts[i]
				if got.ContainerPort != want.ContainerPort {
					t.Errorf("parsePortMappings()[%d].ContainerPort = %d, want %d", i, got.ContainerPort, want.ContainerPort)
				}
				if got.Protocol != want.Protocol {
					t.Errorf("parsePortMappings()[%d].Protocol = %s, want %s", i, got.Protocol, want.Protocol)
				}
			}
		})
	}
}

func TestCreateServiceContainer(t *testing.T) {
	t.Parallel()
	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}

	tests := map[string]struct {
		service        types.ServiceDefinition
		wantName       string
		wantImage      string
		wantEnvCount   int // Minimum expected env vars (CI, GITHUB_ACTIONS always added)
		wantPortsCount int
		wantWorkDir    string
		wantCommand    []string
		wantArgs       []string
		wantErr        bool
	}{
		"minimal service": {
			service: types.ServiceDefinition{
				ContextName: "redis",
				Image:       "redis:7",
			},
			wantName:     "redis",
			wantImage:    "redis:7",
			wantEnvCount: 2, // CI and GITHUB_ACTIONS
			wantErr:      false,
		},
		"service with environment variables": {
			service: types.ServiceDefinition{
				ContextName: "postgres",
				Image:       "postgres:14",
				EnvironmentVariables: map[string]string{
					"POSTGRES_PASSWORD": "password",
					"POSTGRES_DB":       "testdb",
				},
			},
			wantName:     "postgres",
			wantImage:    "postgres:14",
			wantEnvCount: 4, // CI, GITHUB_ACTIONS + 2 custom
			wantErr:      false,
		},
		"service with port mappings": {
			service: types.ServiceDefinition{
				ContextName:  "redis",
				Image:        "redis:7",
				PortMappings: []string{"6379", "6380"},
			},
			wantName:       "redis",
			wantImage:      "redis:7",
			wantEnvCount:   2,
			wantPortsCount: 2,
			wantErr:        false,
		},
		"service with working directory": {
			service: types.ServiceDefinition{
				ContextName:      "myapp",
				Image:            "myapp:latest",
				WorkingDirectory: "/app",
			},
			wantName:     "myapp",
			wantImage:    "myapp:latest",
			wantEnvCount: 2,
			wantWorkDir:  "/app",
			wantErr:      false,
		},
		"service with custom entrypoint": {
			service: types.ServiceDefinition{
				ContextName:    "custom",
				Image:          "custom:latest",
				Entrypoint:     "/usr/bin/custom",
				EntrypointArgs: []string{"--flag", "value"},
			},
			wantName:     "custom",
			wantImage:    "custom:latest",
			wantEnvCount: 2,
			wantCommand:  []string{"/usr/bin/custom"},
			wantArgs:     []string{"--flag", "value"},
			wantErr:      false,
		},
		"service with CreateOptions logs warning but succeeds": {
			service: types.ServiceDefinition{
				ContextName:   "test",
				Image:         "test:latest",
				CreateOptions: "--cpus 2",
			},
			wantName:     "test",
			wantImage:    "test:latest",
			wantEnvCount: 2,
			wantErr:      false,
		},
		"service with invalid port mapping": {
			service: types.ServiceDefinition{
				ContextName:  "bad",
				Image:        "bad:latest",
				PortMappings: []string{"invalid"},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := c.createServiceContainer(tt.service)
			if tt.wantErr {
				if err == nil {
					t.Errorf("createServiceContainer() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("createServiceContainer() unexpected error = %v", err)
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("createServiceContainer().Name = %s, want %s", got.Name, tt.wantName)
			}
			if got.Image != tt.wantImage {
				t.Errorf("createServiceContainer().Image = %s, want %s", got.Image, tt.wantImage)
			}
			if len(got.Env) < tt.wantEnvCount {
				t.Errorf("createServiceContainer().Env count = %d, want at least %d", len(got.Env), tt.wantEnvCount)
			}
			// Verify CI and GITHUB_ACTIONS are always present
			hasCI := false
			hasGitHubActions := false
			for _, env := range got.Env {
				if env.Name == "CI" && env.Value == envTrue {
					hasCI = true
				}
				if env.Name == "GITHUB_ACTIONS" && env.Value == envTrue {
					hasGitHubActions = true
				}
			}
			if !hasCI {
				t.Error("createServiceContainer() missing CI=true env var")
			}
			if !hasGitHubActions {
				t.Error("createServiceContainer() missing GITHUB_ACTIONS=true env var")
			}
			if tt.wantPortsCount > 0 && len(got.Ports) != tt.wantPortsCount {
				t.Errorf("createServiceContainer().Ports count = %d, want %d", len(got.Ports), tt.wantPortsCount)
			}
			if tt.wantWorkDir != "" && got.WorkingDir != tt.wantWorkDir {
				t.Errorf("createServiceContainer().WorkingDir = %s, want %s", got.WorkingDir, tt.wantWorkDir)
			}
			if len(tt.wantCommand) > 0 {
				if len(got.Command) != len(tt.wantCommand) {
					t.Errorf("createServiceContainer().Command length = %d, want %d", len(got.Command), len(tt.wantCommand))
				} else {
					for i, cmd := range got.Command {
						if cmd != tt.wantCommand[i] {
							t.Errorf("createServiceContainer().Command[%d] = %s, want %s", i, cmd, tt.wantCommand[i])
						}
					}
				}
			}
			if len(tt.wantArgs) > 0 {
				if len(got.Args) != len(tt.wantArgs) {
					t.Errorf("createServiceContainer().Args length = %d, want %d", len(got.Args), len(tt.wantArgs))
				} else {
					for i, arg := range got.Args {
						if arg != tt.wantArgs[i] {
							t.Errorf("createServiceContainer().Args[%d] = %s, want %s", i, arg, tt.wantArgs[i])
						}
					}
				}
			}
		})
	}
}

func TestExtractServiceInfo(t *testing.T) {
	t.Parallel()
	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}

	tests := map[string]struct {
		podContainers []v1.Container
		wantServices  []types.ServiceInfo
	}{
		"pod with no services": {
			podContainers: []v1.Container{
				{Name: "job", Image: "ubuntu:22.04"},
			},
			wantServices: []types.ServiceInfo{},
		},
		"pod with single service": {
			podContainers: []v1.Container{
				{Name: "job", Image: "ubuntu:22.04"},
				{
					Name:  "redis",
					Image: "redis:7",
					Ports: []v1.ContainerPort{
						{ContainerPort: 6379, Protocol: v1.ProtocolTCP},
					},
				},
			},
			wantServices: []types.ServiceInfo{
				{
					ContextName: "redis",
					Image:       "redis:7",
					Ports: map[int]int{
						6379: 6379,
					},
				},
			},
		},
		"pod with multiple services": {
			podContainers: []v1.Container{
				{Name: "job", Image: "ubuntu:22.04"},
				{
					Name:  "redis",
					Image: "redis:7",
					Ports: []v1.ContainerPort{
						{ContainerPort: 6379},
					},
				},
				{
					Name:  "postgres",
					Image: "postgres:14",
					Ports: []v1.ContainerPort{
						{ContainerPort: 5432},
					},
				},
			},
			wantServices: []types.ServiceInfo{
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
		},
		"service with multiple ports": {
			podContainers: []v1.Container{
				{Name: "job", Image: "ubuntu:22.04"},
				{
					Name:  "myapp",
					Image: "myapp:latest",
					Ports: []v1.ContainerPort{
						{ContainerPort: 8080},
						{ContainerPort: 8081},
						{ContainerPort: 9090},
					},
				},
			},
			wantServices: []types.ServiceInfo{
				{
					ContextName: "myapp",
					Image:       "myapp:latest",
					Ports: map[int]int{
						8080: 8080,
						8081: 8081,
						9090: 9090,
					},
				},
			},
		},
		"service with no ports": {
			podContainers: []v1.Container{
				{Name: "job", Image: "ubuntu:22.04"},
				{
					Name:  "sidecar",
					Image: "sidecar:latest",
					Ports: []v1.ContainerPort{},
				},
			},
			wantServices: []types.ServiceInfo{
				{
					ContextName: "sidecar",
					Image:       "sidecar:latest",
					Ports:       map[int]int{},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Create a fake pod with the test containers
			pod := &v1.Pod{
				Spec: v1.PodSpec{
					Containers: tt.podContainers,
				},
			}
			// Create the pod in the fake clientset
			pod.Name = "test-pod-" + name
			pod.Namespace = "default"
			_, err := c.client.CoreV1().Pods("default").Create(c.ctx, pod, v1Meta.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to create test pod: %v", err)
			}

			got, err := c.ExtractServiceInfo(pod.Name)
			if err != nil {
				t.Errorf("ExtractServiceInfo() unexpected error = %v", err)
				return
			}

			if len(got) != len(tt.wantServices) {
				t.Errorf("ExtractServiceInfo() got %d services, want %d", len(got), len(tt.wantServices))
				return
			}

			for i, gotService := range got {
				wantService := tt.wantServices[i]
				if gotService.ContextName != wantService.ContextName {
					t.Errorf("ExtractServiceInfo()[%d].ContextName = %s, want %s", i, gotService.ContextName, wantService.ContextName)
				}
				if gotService.Image != wantService.Image {
					t.Errorf("ExtractServiceInfo()[%d].Image = %s, want %s", i, gotService.Image, wantService.Image)
				}
				if len(gotService.Ports) != len(wantService.Ports) {
					t.Errorf("ExtractServiceInfo()[%d].Ports count = %d, want %d", i, len(gotService.Ports), len(wantService.Ports))
					continue
				}
				for port, mappedPort := range wantService.Ports {
					if gotService.Ports[port] != mappedPort {
						t.Errorf("ExtractServiceInfo()[%d].Ports[%d] = %d, want %d", i, port, gotService.Ports[port], mappedPort)
					}
				}
			}
		})
	}
}

// verifyServiceContainers is a helper function to verify service containers in a pod
func verifyServiceContainers(t *testing.T, pod *v1.Pod, services []types.ServiceDefinition, wantServiceNames []string) {
	t.Helper()
	// Start at index 1 (skip job container)
	for i, serviceName := range wantServiceNames {
		containerIndex := i + 1
		if containerIndex >= len(pod.Spec.Containers) {
			t.Errorf("preparePodSpec() missing service container at index %d", containerIndex)
			continue
		}
		serviceContainer := pod.Spec.Containers[containerIndex]
		if serviceContainer.Name != serviceName {
			t.Errorf("preparePodSpec() container[%d].Name = %s, want %s", containerIndex, serviceContainer.Name, serviceName)
		}

		// Find matching service definition
		var serviceDef *types.ServiceDefinition
		for _, svc := range services {
			if svc.ContextName == serviceName {
				serviceDef = &svc
				break
			}
		}

		// Verify image
		if serviceDef == nil || serviceContainer.Image != serviceDef.Image {
			t.Errorf("preparePodSpec() service %s image = %s, servicedef: %v", serviceName, serviceContainer.Image, serviceDef)
		}

		// Verify environment variables contain expected ones
		if len(serviceDef.EnvironmentVariables) > 0 {
			envMap := make(map[string]string)
			for _, env := range serviceContainer.Env {
				envMap[env.Name] = env.Value
			}
			for key, value := range serviceDef.EnvironmentVariables {
				if envMap[key] != value {
					t.Errorf("preparePodSpec() service %s env[%s] = %s, want %s", serviceName, key, envMap[key], value)
				}
			}
		}

		// Verify ports
		if len(serviceDef.PortMappings) > 0 {
			if len(serviceContainer.Ports) != len(serviceDef.PortMappings) {
				t.Errorf("preparePodSpec() service %s port count = %d, want %d", serviceName, len(serviceContainer.Ports), len(serviceDef.PortMappings))
			}
		}

		// Verify CI and GITHUB_ACTIONS env vars
		hasCI := false
		hasGitHubActions := false
		for _, env := range serviceContainer.Env {
			if env.Name == "CI" && env.Value == envTrue {
				hasCI = true
			}
			if env.Name == "GITHUB_ACTIONS" && env.Value == envTrue {
				hasGitHubActions = true
			}
		}
		if !hasCI {
			t.Errorf("preparePodSpec() service %s missing CI env var", serviceName)
		}
		if !hasGitHubActions {
			t.Errorf("preparePodSpec() service %s missing GITHUB_ACTIONS env var", serviceName)
		}
	}
}

func TestPreparePodSpecWithServices(t *testing.T) {
	t.Parallel()
	os.Setenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE", "default")
	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}

	tests := map[string]struct {
		container             types.ContainerDefinition
		services              []types.ServiceDefinition
		podType               PodType
		wantContainerCount    int
		wantServiceNames      []string
		wantJobContainerFirst bool
	}{
		"job pod with single service": {
			container: types.ContainerDefinition{
				Image: "ubuntu:22.04",
			},
			services: []types.ServiceDefinition{
				{
					ContextName:  "redis",
					Image:        "redis:7",
					PortMappings: []string{"6379"},
				},
			},
			podType:               PodTypeJob,
			wantContainerCount:    2,
			wantServiceNames:      []string{"redis"},
			wantJobContainerFirst: true,
		},
		"job pod with multiple services": {
			container: types.ContainerDefinition{
				Image: "ubuntu:22.04",
			},
			services: []types.ServiceDefinition{
				{
					ContextName:  "redis",
					Image:        "redis:7",
					PortMappings: []string{"6379"},
				},
				{
					ContextName: "postgres",
					Image:       "postgres:14",
					EnvironmentVariables: map[string]string{
						"POSTGRES_PASSWORD": "password",
					},
					PortMappings: []string{"5432"},
				},
			},
			podType:               PodTypeJob,
			wantContainerCount:    3,
			wantServiceNames:      []string{"redis", "postgres"},
			wantJobContainerFirst: true,
		},
		"container step pod ignores services": {
			container: types.ContainerDefinition{
				Image: "ubuntu:22.04",
			},
			services: []types.ServiceDefinition{
				{
					ContextName: "redis",
					Image:       "redis:7",
				},
			},
			podType:               PodTypeContainerStep,
			wantContainerCount:    1, // Services not added for container steps
			wantServiceNames:      []string{},
			wantJobContainerFirst: true,
		},
		"job pod with no services": {
			container: types.ContainerDefinition{
				Image: "ubuntu:22.04",
			},
			services:              []types.ServiceDefinition{},
			podType:               PodTypeJob,
			wantContainerCount:    1,
			wantServiceNames:      []string{},
			wantJobContainerFirst: true,
		},
		"services with ports and env vars": {
			container: types.ContainerDefinition{
				Image: "ubuntu:22.04",
			},
			services: []types.ServiceDefinition{
				{
					ContextName: "mysql",
					Image:       "mysql:8",
					EnvironmentVariables: map[string]string{
						"MYSQL_ROOT_PASSWORD": "rootpass",
						"MYSQL_DATABASE":      "testdb",
					},
					PortMappings: []string{"3306", "33060"},
				},
			},
			podType:               PodTypeJob,
			wantContainerCount:    2,
			wantServiceNames:      []string{"mysql"},
			wantJobContainerFirst: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			pod := c.preparePodSpec(tt.container, tt.services, tt.podType)

			// Check container count
			if len(pod.Spec.Containers) != tt.wantContainerCount {
				t.Errorf("preparePodSpec() container count = %d, want %d", len(pod.Spec.Containers), tt.wantContainerCount)
			}

			// Check job container is first
			if tt.wantJobContainerFirst && len(pod.Spec.Containers) > 0 {
				if pod.Spec.Containers[0].Name != "job" {
					t.Errorf("preparePodSpec() first container name = %s, want 'job'", pod.Spec.Containers[0].Name)
				}
			}

			// Check service containers
			if len(tt.wantServiceNames) > 0 {
				verifyServiceContainers(t, pod, tt.services, tt.wantServiceNames)
			}

			// Verify pod metadata
			if pod.Labels == nil {
				t.Error("preparePodSpec() pod.Labels is nil")
			}
			if _, ok := pod.Labels["runner-pod"]; !ok {
				t.Error("preparePodSpec() pod.Labels missing 'runner-pod' label")
			}

			// Verify pod has volume
			if len(pod.Spec.Volumes) == 0 {
				t.Error("preparePodSpec() pod has no volumes")
			}
		})
	}
}

func TestCreatePodWithServices(t *testing.T) {
	t.Parallel()
	// Note: This test uses a fake clientset, so pod creation succeeds immediately
	// without actually creating Kubernetes resources
	os.Setenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE", "default")
	os.Setenv("ACTIONS_RUNNER_POD_NAME", "test-runner")
	os.Setenv("ACTIONS_RUNNER_CLAIM_NAME", "test-claim")

	// Skip wait for pod ready in tests
	os.Setenv("ACTIONS_RUNNER_PREPARE_JOB_TIMEOUT_SECONDS", "1")

	c := K8sClient{
		client: fake.NewClientset(),
		ctx:    t.Context(),
	}

	tests := map[string]struct {
		args               types.InputArgs
		wantErr            bool
		wantContainerCount int
	}{
		"create job pod with single service": {
			args: types.InputArgs{
				Container: types.ContainerDefinition{
					Image: "ubuntu:22.04",
				},
				Services: []types.ServiceDefinition{
					{
						ContextName:  "redis",
						Image:        "redis:7",
						PortMappings: []string{"6379"},
					},
				},
			},
			wantErr:            false,
			wantContainerCount: 2,
		},
		"create job pod with multiple services": {
			args: types.InputArgs{
				Container: types.ContainerDefinition{
					Image: "ubuntu:22.04",
				},
				Services: []types.ServiceDefinition{
					{
						ContextName: "redis",
						Image:       "redis:7",
					},
					{
						ContextName: "postgres",
						Image:       "postgres:14",
					},
					{
						ContextName: "mysql",
						Image:       "mysql:8",
					},
				},
			},
			wantErr:            false,
			wantContainerCount: 4, // job + 3 services
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pod := c.preparePodSpec(tt.args.Container, tt.args.Services, PodTypeJob)

			if len(pod.Spec.Containers) != tt.wantContainerCount {
				t.Errorf("CreatePod() container count = %d, want %d", len(pod.Spec.Containers), tt.wantContainerCount)
			}

			// Verify first container is job container
			if pod.Spec.Containers[0].Name != jobContainerName {
				t.Errorf("CreatePod() first container = %s, want %q", pod.Spec.Containers[0].Name, jobContainerName)
			}

			// Verify service containers follow
			for i := 1; i < len(pod.Spec.Containers); i++ {
				serviceIndex := i - 1
				if serviceIndex >= len(tt.args.Services) {
					t.Errorf("CreatePod() unexpected extra container at index %d", i)
					continue
				}
				expectedName := tt.args.Services[serviceIndex].ContextName
				if pod.Spec.Containers[i].Name != expectedName {
					t.Errorf("CreatePod() container[%d].Name = %s, want %s", i, pod.Spec.Containers[i].Name, expectedName)
				}
			}
		})
	}
}

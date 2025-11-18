package k8s

import (
	"testing"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sClient_waitForPodReady(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		want    v1.PodPhase
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	c := K8sClient{
		client: fake.NewSimpleClientset(),
		ctx:    t.Context(),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotErr := c.waitForPodReady(tt.name)
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
	c := K8sClient{
		client: fake.NewSimpleClientset(),
		ctx:    t.Context(),
	}
	input := types.ContainerDefinition{
		Image: "example-image",
		EnvironmentVariables: map[string]string{
			"GITHUB_WORKSPACE": "/tmp/workspace",
		},
	}
	podSpec := c.preparePodSpec(input, PodTypeJob)
	if podSpec.Spec.Containers[0].Image != "example-image" {
		t.Errorf("expected image 'example-image', got '%s'", podSpec.Spec.Containers[0].Image)
	}
	expectedPaths := []string{"/__e", "/__w", "/github/home", "/github/workflow"}
	volumes := podSpec.Spec.Containers[0].VolumeMounts
	for i, vol := range volumes {
		if vol.MountPath != expectedPaths[i] {
			t.Errorf("job expected mount path '%s', got '%s'", expectedPaths[i], vol.MountPath)
		}
	}
	jobSpec := c.preparePodSpec(input, PodTypeContainerStep)
	expectedJobPaths := []string{"/github/workspace", "/github/file_commands", "/__w", "/github/home", "/github/workflow"}
	jobVolumes := jobSpec.Spec.Containers[0].VolumeMounts
	for i, vol := range jobVolumes {
		if vol.MountPath != expectedJobPaths[i] {
			t.Errorf("job expected mount path '%s', got '%s'", expectedJobPaths[i], vol.MountPath)
		}
	}
}

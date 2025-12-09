package k8s

import (
	"os"
	"path/filepath"
	"testing"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func Test_applyTemplateToPodSpec_WithExampleFile(t *testing.T) {
	t.Parallel()

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "job",
					Image: "test:latest",
					Env: []v1.EnvVar{
						{Name: "EXISTING_VAR", Value: "existing-value"},
					},
					VolumeMounts: []v1.VolumeMount{
						{Name: "work", MountPath: "/work"},
					},
				},
			},
			Volumes: []v1.Volume{
				{Name: "work", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
			},
		},
	}

	templatePath := filepath.Join("..", "..", "examples", "extension.yaml")
	err := applyTemplateToPod(pod, templatePath)
	if err != nil {
		t.Fatalf("applyTemplateToPodSpec() failed: %v", err)
	}

	// Verify ObjectMeta labels
	if val, ok := pod.Labels["labeled-by"]; !ok || val != "extension" {
		t.Fatalf("expected label 'labeled-by=extension', got '%s=%s'", "example-label", val)
	}

	// Verify service account was set
	if pod.Spec.ServiceAccountName != "custom-service-account" {
		t.Errorf("expected service account 'custom-service-account', got '%s'", pod.Spec.ServiceAccountName)
	}

	// Verify volumes were appended
	if len(pod.Spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(pod.Spec.Volumes))
	}
	if pod.Spec.Volumes[1].Name != "custom-volume" {
		t.Errorf("expected second volume 'custom-volume', got '%s'", pod.Spec.Volumes[1].Name)
	}

	// Verify env vars were appended to job container
	envVars := pod.Spec.Containers[0].Env
	if len(envVars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(envVars))
	}
	if envVars[0].Name != "EXISTING_VAR" {
		t.Errorf("expected first env var 'EXISTING_VAR', got '%s'", envVars[0].Name)
	}
	if envVars[1].Name != "ENV1" || envVars[1].Value != "value1" {
		t.Errorf("expected second env var 'ENV1=value1', got '%s=%s'", envVars[1].Name, envVars[1].Value)
	}

	// Verify volume mounts were appended to job container
	mounts := pod.Spec.Containers[0].VolumeMounts
	if len(mounts) != 2 {
		t.Fatalf("expected 2 volume mounts, got %d", len(mounts))
	}
	if mounts[0].Name != "work" {
		t.Errorf("expected first mount 'work', got '%s'", mounts[0].Name)
	}
	if mounts[1].Name != "custom-volume" || mounts[1].MountPath != "/custom/path" {
		t.Errorf("expected second mount 'custom-volume' at '/custom/path', got '%s' at '%s'",
			mounts[1].Name, mounts[1].MountPath)
	}
}

func Test_applyMinmalTemplate(t *testing.T) {
	t.Parallel()

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "job",
					Image: "test:latest",
					Env: []v1.EnvVar{
						{Name: "EXISTING_VAR", Value: "existing-value"},
					},
					VolumeMounts: []v1.VolumeMount{
						{Name: "work", MountPath: "/work"},
					},
				},
			},
			Volumes: []v1.Volume{
				{Name: "work", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
			},
		},
	}

	templatePath := filepath.Join("..", "..", "examples", "minimal-extension.yaml")
	err := applyTemplateToPod(pod, templatePath)
	if err != nil {
		t.Fatalf("applyTemplateToPodSpec() failed: %v", err)
	}
	if len(pod.Spec.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(pod.Spec.Volumes))
	}
	if pod.Spec.ServiceAccountName != "custom-service-account" {
		t.Fatalf("expected service account 'custom-service-account', got '%s'", pod.Spec.ServiceAccountName)
	}
}

func Test_applyTemplateToPodSpec_FileNotFound(t *testing.T) {
	t.Parallel()

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{Name: "job", Image: "test:latest"},
			},
		},
	}

	nonExistentFile := filepath.Join(t.TempDir(), "non-existent.yaml")
	err := applyTemplateToPod(pod, nonExistentFile)

	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func Test_applyTemplateToPodSpec_MultipleJobContainers(t *testing.T) {
	t.Parallel()

	pod := v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "container1",
					Env: []v1.EnvVar{
						{Name: "VAR1", Value: "value1"},
					}, VolumeMounts: []v1.VolumeMount{
						{
							Name:      "mount1",
							MountPath: "/mount1",
						},
					},
				},
			},
		},
	}
	templateYAML, err := yaml.Marshal(pod)
	if err != nil {
		t.Fatalf("Failed to marshal pod to YAML: %v", err)
	}

	tmpFile, err := os.CreateTemp("", "template-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Fatalf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tmpFile.WriteString(string(templateYAML)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	pod = v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "job",
					Image: "test:latest",
					Env: []v1.EnvVar{
						{Name: "EXISTING", Value: "value"},
					},
				},
				{
					Name:  "sidecar",
					Image: "sidecar:latest",
				},
			},
		},
	}

	err = applyTemplateToPod(&pod, tmpFile.Name())
	if err != nil {
		t.Fatalf("applyTemplateToPodSpec() failed: %v", err)
	}

	if len(pod.Spec.Containers[0].Env) != 1 {
		t.Errorf("expected 1 env vars on job container, got %d", len(pod.Spec.Containers[0].Env))
	}
	if len(pod.Spec.Containers[0].VolumeMounts) != 0 {
		t.Errorf("expected 0 volume mount on job container, got %d", len(pod.Spec.Containers[0].VolumeMounts))
	}

	if len(pod.Spec.Containers[1].Env) != 0 {
		t.Errorf("expected sidecar container to remain unchanged, got %d env vars", len(pod.Spec.Containers[1].Env))
	}
	pod = v1.Pod{
		Spec: v1.PodSpec{
			ServiceAccountName: "runner-sa",
		},
	}

	err = applyTemplateToPod(&pod, tmpFile.Name())
	if err != nil {
		t.Fatalf("applyTemplateToPodSpec() failed: %v", err)
	}
}

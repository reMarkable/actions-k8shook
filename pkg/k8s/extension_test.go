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

func Test_applyTemplateToPod_WithServices(t *testing.T) {
	t.Parallel()

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  jobContainerName,
					Image: "ubuntu:22.04",
					Env: []v1.EnvVar{
						{Name: "JOB_VAR", Value: "job-value"},
					},
					VolumeMounts: []v1.VolumeMount{
						{Name: "work", MountPath: "/work"},
					},
				},
				{
					Name:  "redis",
					Image: "redis:7",
				},
				{
					Name:  "postgres",
					Image: "postgres:14",
				},
			},
			Volumes: []v1.Volume{
				{Name: "work", VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
			},
		},
	}

	templatePath := filepath.Join("..", "..", "examples", "extension-with-services.yaml")
	err := applyTemplateToPod(pod, templatePath)
	if err != nil {
		t.Fatalf("applyTemplateToPod() failed: %v", err)
	}

	// Verify volumes were appended
	if len(pod.Spec.Volumes) != 3 {
		t.Fatalf("expected 3 volumes, got %d", len(pod.Spec.Volumes))
	}
	if pod.Spec.Volumes[1].Name != "redis-data" {
		t.Errorf("expected second volume 'redis-data', got '%s'", pod.Spec.Volumes[1].Name)
	}
	if pod.Spec.Volumes[2].Name != "postgres-data" {
		t.Errorf("expected third volume 'postgres-data', got '%s'", pod.Spec.Volumes[2].Name)
	}

	// Verify job container was modified
	jobContainer := pod.Spec.Containers[0]
	if len(jobContainer.Env) != 2 {
		t.Fatalf("expected 2 env vars on job container, got %d", len(jobContainer.Env))
	}
	if jobContainer.Env[1].Name != "DATABASE_URL" || jobContainer.Env[1].Value != "postgres://localhost:5432" {
		t.Errorf("expected DATABASE_URL env var, got '%s=%s'", jobContainer.Env[1].Name, jobContainer.Env[1].Value)
	}

	// Verify redis service container was modified
	redisContainer := pod.Spec.Containers[1]
	if len(redisContainer.Env) != 1 {
		t.Fatalf("expected 1 env var on redis container, got %d", len(redisContainer.Env))
	}
	if redisContainer.Env[0].Name != "REDIS_PASSWORD" || redisContainer.Env[0].Value != "secret" {
		t.Errorf("expected REDIS_PASSWORD env var, got '%s=%s'", redisContainer.Env[0].Name, redisContainer.Env[0].Value)
	}
	if len(redisContainer.VolumeMounts) != 1 {
		t.Fatalf("expected 1 volume mount on redis container, got %d", len(redisContainer.VolumeMounts))
	}
	if redisContainer.VolumeMounts[0].Name != "redis-data" || redisContainer.VolumeMounts[0].MountPath != "/data" {
		t.Errorf("expected redis-data mount at /data, got '%s' at '%s'",
			redisContainer.VolumeMounts[0].Name, redisContainer.VolumeMounts[0].MountPath)
	}

	// Verify postgres service container was modified
	postgresContainer := pod.Spec.Containers[2]
	if len(postgresContainer.Env) != 2 {
		t.Fatalf("expected 2 env vars on postgres container, got %d", len(postgresContainer.Env))
	}
	if postgresContainer.Env[0].Name != "POSTGRES_PASSWORD" || postgresContainer.Env[0].Value != "password" {
		t.Errorf("expected POSTGRES_PASSWORD env var, got '%s=%s'", postgresContainer.Env[0].Name, postgresContainer.Env[0].Value)
	}
	if postgresContainer.Env[1].Name != "POSTGRES_DB" || postgresContainer.Env[1].Value != "testdb" {
		t.Errorf("expected POSTGRES_DB env var, got '%s=%s'", postgresContainer.Env[1].Name, postgresContainer.Env[1].Value)
	}
	if len(postgresContainer.VolumeMounts) != 1 {
		t.Fatalf("expected 1 volume mount on postgres container, got %d", len(postgresContainer.VolumeMounts))
	}
	if postgresContainer.VolumeMounts[0].Name != "postgres-data" || postgresContainer.VolumeMounts[0].MountPath != "/var/lib/postgresql/data" {
		t.Errorf("expected postgres-data mount at /var/lib/postgresql/data, got '%s' at '%s'",
			postgresContainer.VolumeMounts[0].Name, postgresContainer.VolumeMounts[0].MountPath)
	}
}

func Test_applyTemplateToPod_ServiceNotFound(t *testing.T) {
	t.Parallel()

	pod := &v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  jobContainerName,
					Image: "ubuntu:22.04",
				},
				{
					Name:  "redis",
					Image: "redis:7",
				},
			},
		},
	}

	// Create a template that references a non-existent service
	templatePod := v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name: "$nonexistent",
					Env: []v1.EnvVar{
						{Name: "TEST_VAR", Value: "test-value"},
					},
				},
			},
		},
	}
	templateYAML, err := yaml.Marshal(templatePod)
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

	err = applyTemplateToPod(pod, tmpFile.Name())
	if err != nil {
		t.Fatalf("applyTemplateToPod() failed: %v", err)
	}

	// Verify that the nonexistent service was silently ignored (no error, no changes)
	if len(pod.Spec.Containers[0].Env) != 0 {
		t.Errorf("expected job container to remain unchanged, got %d env vars", len(pod.Spec.Containers[0].Env))
	}
	if len(pod.Spec.Containers[1].Env) != 0 {
		t.Errorf("expected redis container to remain unchanged, got %d env vars", len(pod.Spec.Containers[1].Env))
	}
}

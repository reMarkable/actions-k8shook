//go:build !containers_image_storage_stub
// +build !containers_image_storage_stub

package container

import (
	"context"
	"strings"
	"testing"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func TestGetEntrypoint_Nginx(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Nginx has /docker-entrypoint.sh as entrypoint
	entrypoint, err := inspector.GetEntrypoint("docker.io/library/nginx:latest", nil)
	if err != nil {
		t.Fatalf("Failed to get entrypoint: %v", err)
	}

	if entrypoint == "" {
		t.Error("Expected non-empty entrypoint for nginx image")
	}

	if !strings.Contains(entrypoint, "docker-entrypoint.sh") {
		t.Errorf("Expected entrypoint to contain docker-entrypoint.sh, got: %s", entrypoint)
	}
}

func TestGetEntrypoint_Redis(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Redis has docker-entrypoint.sh as entrypoint
	entrypoint, err := inspector.GetEntrypoint("docker.io/library/redis:latest", nil)
	if err != nil {
		t.Fatalf("Failed to get entrypoint: %v", err)
	}

	if entrypoint == "" {
		t.Error("Expected non-empty entrypoint for redis image")
	}

	if !strings.Contains(entrypoint, "docker-entrypoint.sh") {
		t.Errorf("Expected entrypoint to contain docker-entrypoint.sh, got: %s", entrypoint)
	}
}

func TestGetEntrypoint_NoEntrypoint(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Alpine has no entrypoint, only CMD
	entrypoint, err := inspector.GetEntrypoint("docker.io/library/alpine:latest", nil)
	if err != nil {
		t.Fatalf("Failed to inspect image: %v", err)
	}

	// Should return empty string when no entrypoint is defined
	if entrypoint != "" {
		t.Errorf("Expected empty entrypoint for alpine image (only has CMD), got: %s", entrypoint)
	}
}

func TestGetEntrypoint_WithDockerPrefix(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Test with explicit docker:// prefix
	entrypoint, err := inspector.GetEntrypoint("docker://docker.io/library/nginx:latest", nil)
	if err != nil {
		t.Fatalf("Failed to get entrypoint: %v", err)
	}

	if entrypoint == "" {
		t.Error("Expected non-empty entrypoint for nginx image")
	}

	if !strings.Contains(entrypoint, "docker-entrypoint.sh") {
		t.Errorf("Expected entrypoint to contain docker-entrypoint.sh, got: %s", entrypoint)
	}
}

func TestGetEntrypoint_InvalidImage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Test with non-existent image
	_, err := inspector.GetEntrypoint("docker.io/library/this-image-does-not-exist-12345:latest", nil)
	if err == nil {
		t.Error("Expected error for non-existent image, got nil")
	}
}

func TestGetEntrypoint_InvalidReference(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	inspector := NewInspector(ctx)

	// Test with invalid image reference format
	_, err := inspector.GetEntrypoint("not-a-valid-image-reference::", nil)
	if err == nil {
		t.Error("Expected error for invalid image reference, got nil")
	}
}

func TestGetEntrypoint_WithSHAAndTag(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := NewInspector(ctx)

	// Test with both tag and SHA - the tag should be stripped
	// Using nginx image with a specific SHA (this is a real nginx SHA)
	entrypoint, err := inspector.GetEntrypoint("docker.io/library/nginx:latest@sha256:447a8665cc1dab95b1ca778e162215839ccbb9189104c79d7ec3a81e14577add", nil)
	if err != nil {
		t.Fatalf("Failed to get entrypoint: %v", err)
	}

	if entrypoint == "" {
		t.Error("Expected non-empty entrypoint for nginx image")
	}

	if !strings.Contains(entrypoint, "docker-entrypoint.sh") {
		t.Errorf("Expected entrypoint to contain docker-entrypoint.sh, got: %s", entrypoint)
	}
}

func TestExtractEntrypoint(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		config   *v1.Image
		expected string
	}{
		"nil config": {
			config:   nil,
			expected: "",
		},
		"empty entrypoint": {
			config: &v1.Image{
				Config: v1.ImageConfig{
					Entrypoint: []string{},
				},
			},
			expected: "",
		},
		"single entrypoint": {
			config: &v1.Image{
				Config: v1.ImageConfig{
					Entrypoint: []string{"/bin/sh"},
				},
			},
			expected: "/bin/sh",
		},
		"multiple entrypoint parts": {
			config: &v1.Image{
				Config: v1.ImageConfig{
					Entrypoint: []string{"/bin/sh", "-c"},
				},
			},
			expected: "/bin/sh -c",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := extractEntrypoint(tt.config)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

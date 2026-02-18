// Package container provides utilities for inspecting container images.
//
//go:build !containers_image_storage_stub

package container

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"go.podman.io/image/v5/image"
	"go.podman.io/image/v5/transports/alltransports"
	"go.podman.io/image/v5/types"
)

// Inspector provides methods to inspect container images.
type Inspector struct {
	ctx context.Context
}

// ImageConfig represents the relevant configuration extracted from a container image.
type ImageConfig struct {
	Entrypoint []string
	Digest     digest.Digest
}

// NewInspector creates a new container image inspector.
func NewInspector(ctx context.Context) *Inspector {
	return &Inspector{
		ctx: ctx,
	}
}

// GetEntrypoint retrieves the entrypoint from a container image's configuration.
// It returns the entrypoint as a single string (space-joined array), or an empty string if not found.
//
// Parameters:
//   - imageRef: Image reference (e.g., "ghcr.io/remarkable/helmfile-nix:latest" or "docker://ghcr.io/remarkable/helmfile-nix:latest")
//   - registry: Optional registry authentication map with "username", "password", and "serverurl" keys
//
// Returns:
//   - The entrypoint as a space-joined string, or empty string if the image has no entrypoint
//   - An error if the image cannot be inspected
func (i *Inspector) GetEntrypoint(imageRef string, registry map[string]string) (string, error) {
	// Ensure the image reference has a transport prefix
	if !strings.Contains(imageRef, "://") {
		imageRef = "docker://" + imageRef
	}

	slog.Debug("Inspecting image for entrypoint", "image", imageRef)

	ref, err := alltransports.ParseImageName(imageRef)
	if err != nil {
		return "", fmt.Errorf("failed to parse image reference: %w", err)
	}

	sys := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.OptionalBoolFalse,
		OCIInsecureSkipTLSVerify:    false,
	}

	if registry != nil {
		if username, ok := registry["username"]; ok && username != "" {
			password := ""
			if p, ok := registry["password"]; ok {
				password = p
			}
			sys.DockerAuthConfig = &types.DockerAuthConfig{
				Username: username,
				Password: password,
			}
			slog.Debug("Using registry authentication", "username", username)
		}
	}

	src, err := ref.NewImageSource(i.ctx, sys)
	if err != nil {
		return "", fmt.Errorf("failed to create image source: %w", err)
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			slog.Warn("Failed to close image source", "err", closeErr)
		}
	}()

	unparsedInstance := image.UnparsedInstance(src, nil)

	img, err := image.FromUnparsedImage(i.ctx, sys, unparsedInstance)
	if err != nil {
		return "", fmt.Errorf("failed to parse image: %w", err)
	}

	config, err := img.OCIConfig(i.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get image config: %w", err)
	}

	entrypoint := extractEntrypoint(config)

	if entrypoint == "" {
		slog.Debug("Image has no entrypoint defined", "image", imageRef)
	} else {
		slog.Debug("Found entrypoint in image config", "image", imageRef, "entrypoint", entrypoint)
	}

	return entrypoint, nil
}

// extractEntrypoint extracts and formats the entrypoint from an OCI image config.
func extractEntrypoint(config *v1.Image) string {
	if config == nil || config.Config.Entrypoint == nil || len(config.Config.Entrypoint) == 0 {
		return ""
	}

	return strings.Join(config.Config.Entrypoint, " ")
}

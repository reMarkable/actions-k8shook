package command

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/reMarkable/k8s-hook/pkg/container"
	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func RunContainerStep(input types.ContainerHookInput) int {
	if input.Args.Entrypoint == "" {
		if !trySetEntrypointFromImage(&input) {
			return 1
		}
	}

	if input.Args.Dockerfile != "" {
		slog.Error("Self hosted container steps do not support Docker builder at this time")
		return 1
	}

	k, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
		return 1
	}

	args := input.Args
	args.Container = args.ContainerDefinition
	podName, err := k.CreatePod(args, k8s.PodTypeContainerStep)
	if err != nil {
		slog.Error("Failed to create pod", "err", err)
		return 1
	}

	defer func() {
		err = k.DeletePod(podName)
		if err != nil {
			slog.Error("Failed to clean up pod", "err", err)
		}
	}()
	err = k.ExecStepInPod(podName, input.Args)
	if err != nil {
		slog.Error("Failed to run container", "err", err)
		return 1
	}

	return 0
}

// trySetEntrypointFromImage attempts to set the entrypoint from image inspection
// or environment variables. Returns false if entrypoint cannot be determined.
func trySetEntrypointFromImage(input *types.ContainerHookInput) bool {
	// EXPERIMENTAL: Try to inspect the image to get entrypoint if ENV_HOOK_INSPECT_IMAGE is set
	if os.Getenv("ENV_HOOK_INSPECT_IMAGE") == "1" {
		if inspectAndSetEntrypoint(input) {
			return true
		}
	}

	entrypointEnv := os.Getenv("ENV_HOOK_CONTAINER_STEP_ENTRYPOINT")
	if entrypointEnv != "" {
		slog.Info("Entrypoint not set, using ENV_HOOK_CONTAINER_STEP_ENTRYPOINT from environment", "entrypoint", entrypointEnv)
		input.Args.Entrypoint = entrypointEnv
		return true
	}

	slog.Error("Self hosted container steps requires entrypoint to be set")
	return false
}

// inspectAndSetEntrypoint inspects the container image and sets the entrypoint if found.
// Returns true if entrypoint was successfully set.
func inspectAndSetEntrypoint(input *types.ContainerHookInput) bool {
	slog.Info("ENV_HOOK_INSPECT_IMAGE is enabled, attempting to inspect image for entrypoint", "image", input.Args.Image)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inspector := container.NewInspector(ctx)
	entrypoint, err := inspector.GetEntrypoint(input.Args.Image, input.Args.Registry)

	switch {
	case err != nil:
		slog.Warn("Failed to inspect image for entrypoint, will fall back to environment variable", "err", err, "image", input.Args.Image)
		return false
	case entrypoint != "":
		slog.Info("Using entrypoint from image config", "entrypoint", entrypoint, "image", input.Args.Image)
		input.Args.Entrypoint = entrypoint
		return true
	default:
		slog.Debug("Image has no entrypoint defined, will fall back to environment variable", "image", input.Args.Image)
		return false
	}
}

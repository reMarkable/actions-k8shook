package command

import (
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func RunContainerStep(input types.ContainerHookInput) int {
	if input.Args.Entrypoint == "" {
		entrypointEnv := os.Getenv("ENV_HOOK_CONTAINER_STEP_ENTRYPOINT")
		if entrypointEnv != "" {
			slog.Info("Entrypoint not set, using ENV_HOOK_CONTAINER_STEP_ENTRYPOINT from environment", "entrypoint", entrypointEnv)
			input.Args.Entrypoint = os.Getenv("ENV_HOOK_DEFAULT_ENTRYPOINT")
		} else {
			slog.Error("Self hosted container steps requires entrypoint to be set")
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

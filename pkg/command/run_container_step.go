package command

import (
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func RunContainerStep(input types.ContainerHookInput) {
	exitCode := 0
	if input.Args.Entrypoint == "" {
		slog.Error("Self hosted container steps requires entrypoint to be set")
		os.Exit(1)
	}
	if input.Args.Dockerfile != "" {
		slog.Error("Self hosted container steps do not support Docker builder at this time")
		os.Exit(1)
	}
	k8s, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
	}
	args := input.Args
	args.Container = args.ContainerDefinition
	podName, err := k8s.CreatePod(args, true)
	if err != nil {
		slog.Error("Failed to create pod", "err", err)
		os.Exit(1)
	}
	defer func() {
		err := k8s.DeletePod(podName)
		if err != nil {
			slog.Error("Failed to clean up pod", "err", err)
			exitCode = 1
		}
		os.Exit(exitCode)
	}()
	err = k8s.ExecStepInPod(podName, input.Args)
	if err != nil {
		slog.Error("Failed to run container", "err", err)
		exitCode = 1
	}
}

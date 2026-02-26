// Package command contains the various commands that can be executed by the hook
package command

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
	"github.com/reMarkable/k8s-hook/pkg/validation"
)

func PrepareJob(input types.ContainerHookInput) int {
	if err := validation.ValidateServices(input.Args.Services); err != nil {
		slog.Error("Invalid service configuration", "err", err)
		return 1
	}

	k, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
		return 1
	}

	podName, err := k.CreatePod(input.Args, k8s.PodTypeJob)
	if err != nil {
		// FIXME: We need more robust error handling here
		slog.Error("Failed to create pod", "err", err)
		return 1
	}
	alpineArgs := []string{"-c", "test -f /etc/alpine-release"}
	isAlpine := k.ExecInPod(podName, alpineArgs)

	slog.Info("Created pod", "pod", podName)

	services, err := k.ExtractServiceInfo(podName)
	if err != nil {
		slog.Warn("Failed to extract service info", "err", err)
	}

	response := types.ResponseType{
		State: types.ResponseState{
			JobPod: podName,
		},
		Context: map[string]types.ContainerInfo{
			"container": {
				Image: input.Args.Container.Image,
				Ports: map[int]int{},
			},
		},
		Services: services,
		IsAlpine: isAlpine == nil,
	}
	if err := writeResponse(input.ResponseFile, response); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write response: %v\n", err)
		return 1
	}

	return 0
}

func writeResponse(file string, response types.ResponseType) error {
	body, err := json.MarshalIndent(response, "", "  ")
	slog.Debug("Writing response", "body", string(body))
	if err != nil {
		return err
	}

	fh, err := os.Create(file)
	if err != nil {
		return err
	}

	_, err = io.Writer.Write(fh, body)
	if err != nil {
		return err
	}

	return nil
}

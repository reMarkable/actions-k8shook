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
)

func PrepareJob(input types.ContainerHookInput) int {
	k8s, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
		return 1
	}
	podName, err := k8s.CreatePod(input.Args, false)
	if err != nil {
		// FIXME: We need more robust error handling here
		slog.Error("Failed to create pod", "err", err)
		return 1
	}
	slog.Info("Created pod", "pod", podName)
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
		IsAlpine: false,
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

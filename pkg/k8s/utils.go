package k8s

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ErrPodStartup = errors.New("pod failed to start")
	ErrValidation = errors.New("validation error")
)

func (c *K8sClient) GetNS() string {
	namespace := os.Getenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE")
	if namespace != "" {
		return namespace
	}
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		slog.Warn("Failed to read namespace from ACTIONS_RUNNER_KUBERNETES_NAMESPACE or service account, defaulting to 'default'", "error", err)
	} else {
		return strings.TrimSpace(string(namespaceBytes))
	}

	return "default"
}

func (c *K8sClient) GetPodNodeName(mname string) (string, error) {
	pod, err := c.client.CoreV1().Pods(c.GetNS()).Get(c.ctx, mname, v1Meta.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

func (c *K8sClient) GetRunnerPodName() string {
	name := os.Getenv("ACTIONS_RUNNER_POD_NAME")
	if name == "" {
		name = "local-pod"
	}
	return name
}

func (c *K8sClient) GetVolumeClaimName() string {
	name := os.Getenv("ACTIONS_RUNNER_CLAIM_NAME")
	if name == "" {
		return c.GetRunnerPodName() + "-work"
	}
	return name
}

func (c *K8sClient) checkPermissions() {
	// FIXME: Implement permission checks
	slog.Warn("TODO: Implement permission check for creating pods")
}

// writeRunScript generates a shell script that sets up the environment and runs the specified entrypoint with its arguments.
// returns the path to the script in the container and the local path of the script file, or error if any.
func (c *K8sClient) writeRunScript(args types.InputArgs) (string, string, error) {
	prependPath := strings.Join(args.PrependPath, ":")
	cl := strings.Join(append([]string{args.Entrypoint}, args.EntrypointArgs...), " ")
	scriptEnv, err := scriptEnvironment(args.EnvironmentVariables)
	if err != nil {
		return "", "", err
	}
	script := fmt.Sprintf(`
	#!/bin/sh -l
	set -e
	export PATH=%s:$PATH
	cd %s && exec %s %s
  `, prependPath, args.WorkingDirectory, scriptEnv, cl)
	f, err := os.CreateTemp(os.Getenv("RUNNER_TEMP"), "run-script-*.sh")
	if err != nil {
		return "", "", err
	}
	_, err = io.WriteString(f, script)
	if err != nil {
		return "", "", err
	}
	return "/__w/_temp/" + filepath.Base(f.Name()), f.Name(), nil
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		// Copy file
		return copyFile(path, targetPath, info.Mode())
	})
}

// copyExternals copies the externals directory into the workspace directory.
func copyExternals() {
	workspace := os.Getenv("RUNNER_WORKSPACE")
	if workspace != "" {
		slog.Info("Copying externals to workspace", "workspace", workspace)
		err := copyDir(filepath.Join(workspace, "../../externals"), filepath.Join(workspace, "../externals"))
		if err != nil {
			slog.Error("Failed to copy externals", "error", err)
		}
	}
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err = in.Close(); err != nil {
			slog.Warn("Failed to close source file", "error", err)
		}
	}()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			slog.Warn("Failed to close destination file", "error", err)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func getPrepareJobTimeout() int {
	const defaultTimeout = 600
	t, ok := os.LookupEnv("ACTIONS_RUNNER_PREPARE_JOB_TIMEOUT_SECONDS")
	if ok {
		convTimeout, convErr := strconv.Atoi(t)
		if convErr != nil {
			slog.Info("Invalid timeout value, using default of 600 seconds")
			return defaultTimeout
		}
		return convTimeout
	}
	slog.Info("Using default timeout of 600 seconds for preparing job pod")
	return defaultTimeout
}

func podPostfix() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	post := make([]rune, 5)
	for i := range post {
		post[i] = letters[rand.Intn(len(letters))] // #nosec G404
	}
	return string(post)
}

func podEventHandler(cancel context.CancelFunc, errPtr *error) func(oldObj, newObj any) {
	return func(oldObj, newObj any) {
		pod, ok := newObj.(*v1.Pod)
		if !ok {
			slog.Error("Received non-pod object in UpdateFunc")
			return
		}
		slog.Debug("Pod status changed", "pod", pod.Name, "status", pod.Status.Phase)
		for _, c := range pod.Status.ContainerStatuses {
			slog.Debug("Container state", "name", c.Name, "state", c.State)
			if c.State.Waiting != nil && c.State.Waiting.Reason == "ImagePullBackOff" {
				slog.Error("Runner failed to pull image", "pod", pod.Name, "reason", c.State.Waiting.Reason, "message", c.State.Waiting.Message)
				*errPtr = fmt.Errorf("%w: failed to pull image: %s", ErrPodStartup, c.State.Waiting.Message)
				cancel()
			}
			if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
				slog.Error("Runner image crashing on startup", "pod", pod.Name, "reason", c.State.Waiting.Reason, "message", c.State.Waiting.Message)
				*errPtr = fmt.Errorf("%w: image crashing on startup: %s", ErrPodStartup, c.State.Waiting.Message)
				cancel()
			}
		}
		if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodFailed {
			if pod.Status.Phase == v1.PodFailed {
				*errPtr = fmt.Errorf("%w: pod failed", ErrPodStartup)
			}
			cancel()
		}
	}
}

func scriptEnvironment(env map[string]string) (string, error) {
	var envstr strings.Builder
	envstr.WriteString("env")
	for k, v := range env {
		if strings.ContainsAny(k, `"'=$`) {
			return "", fmt.Errorf("%w: invalid character [\"'=$] in environment variable key: %s", ErrValidation, k)
		}
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		v = strings.ReplaceAll(v, `$`, `\$`)
		v = strings.ReplaceAll(v, "`", "\\`")
		envstr.WriteString(fmt.Sprintf(` "%s=%s"`, k, v))
	}
	return envstr.String(), nil
}

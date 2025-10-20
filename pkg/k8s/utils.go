package k8s

import (
	"log/slog"
	"os"

	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *K8sClient) GetNS() string {
	namespace := os.Getenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE")
	if namespace != "" {
		return namespace
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
	slog.Warn("TODO: Implement permission check for creating pods")
}

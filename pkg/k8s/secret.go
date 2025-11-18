package k8s

import (
	"fmt"
	"log/slog"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func (c *K8sClient) PruneSecrets() error {
	secretList, err := c.client.CoreV1().Secrets(c.GetNS()).List(c.ctx, v1Meta.ListOptions{
		LabelSelector: fmt.Sprintf("runner-pod=%s", c.GetRunnerPodName()),
	})
	if err != nil {
		return err
	}
	for _, secret := range secretList.Items {
		slog.Info("Pruning secret", "secret", secret.Name)
		err = c.client.CoreV1().Secrets(c.GetNS()).Delete(c.ctx, secret.Name, v1Meta.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *K8sClient) createImagePullSecret(cont types.ContainerDefinition) (string, error) {
	registryURL := cont.Registry["serverUrl"]
	if registryURL == "" {
		registryURL = "ghcr.io"
	}
	authContent := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s"}}}`,
		registryURL, cont.Registry["username"], cont.Registry["password"])
	secret := v1.Secret{
		Immutable: ptr.To(true),
		ObjectMeta: v1Meta.ObjectMeta{
			Name: c.GetRunnerPodName() + "-pull-secret-" + podPostfix(),
			Labels: map[string]string{
				"runner-pod": c.GetRunnerPodName(),
			},
		},
		StringData: map[string]string{".dockerconfigjson": authContent},
		Type:       v1.SecretTypeDockerConfigJson,
	}
	s, err := c.client.CoreV1().Secrets(c.GetNS()).Create(c.ctx, &secret, v1Meta.CreateOptions{})
	if err != nil {
		return "", err
	}
	return s.Name, nil
}

package k8s

import (
	"fmt"
	"log/slog"

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

// createImagePullSecretFromRegistry creates a Kubernetes image pull secret from registry credentials
func (c *K8sClient) createImagePullSecret(registry map[string]string) (string, error) {
	if registry == nil {
		return "", nil
	}
	registryURL := registry["serverUrl"]
	if registryURL == "" {
		registryURL = "ghcr.io"
	}
	authContent := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s"}}}`,
		registryURL, registry["username"], registry["password"])
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

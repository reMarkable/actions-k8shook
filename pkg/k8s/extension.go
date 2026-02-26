package k8s

import (
	"maps"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// findTargetContainer finds the pod container that matches the extension container name pattern.
// Returns nil if no match is found.
// - "$job" matches the first container (job container)
// - "$<serviceName>" matches service containers by name (without $ prefix)
func findTargetContainer(podContainers []v1.Container, extensionContainerName string) *v1.Container {
	if extensionContainerName == "$job" {
		return &podContainers[0]
	}

	if len(extensionContainerName) > 1 && extensionContainerName[0] == '$' {
		serviceName := extensionContainerName[1:]
		for i := 1; i < len(podContainers); i++ {
			if podContainers[i].Name == serviceName {
				return &podContainers[i]
			}
		}
	}

	return nil
}

func applyTemplateToPod(pod *v1.Pod, template string) error {
	templateContent, err := os.ReadFile(template)
	if err != nil {
		return err
	}
	podExtension := v1.Pod{}
	if err := yaml.Unmarshal(templateContent, &podExtension); err != nil {
		return err
	}
	extensionSpec := podExtension.Spec
	podSpec := &pod.Spec
	podSpec.Volumes = append(podSpec.Volumes, extensionSpec.Volumes...)
	if extensionSpec.ServiceAccountName != "" {
		podSpec.ServiceAccountName = extensionSpec.ServiceAccountName
	}
	if podExtension.Labels != nil {
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		maps.Copy(pod.Labels, podExtension.Labels)
	}
	if podExtension.Annotations != nil {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		maps.Copy(pod.Annotations, podExtension.Annotations)
	}
	if len(podSpec.Containers) != 0 {
		for _, tContainer := range extensionSpec.Containers {
			targetContainer := findTargetContainer(podSpec.Containers, tContainer.Name)
			if targetContainer != nil {
				targetContainer.Env = append(targetContainer.Env, tContainer.Env...)
				targetContainer.VolumeMounts = append(targetContainer.VolumeMounts, tContainer.VolumeMounts...)
			}
		}
	}
	return nil
}

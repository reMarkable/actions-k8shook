package k8s

import (
	"maps"
	"os"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

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
			if tContainer.Name == "$job" {
				jobContainer := &podSpec.Containers[0]
				jobContainer.Env = append(jobContainer.Env, tContainer.Env...)
				jobContainer.VolumeMounts = append(jobContainer.VolumeMounts, tContainer.VolumeMounts...)
			}
		}
	}
	return nil
}

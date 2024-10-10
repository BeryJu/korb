package migrator

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Migrator) getPVCPods(pvcToCheck *v1.PersistentVolumeClaim) ([]v1.Pod, error) {
	nsPods, err := m.kClient.CoreV1().Pods(m.SourceNamespace).List(m.ctx, metav1.ListOptions{})
	if err != nil {
		return []v1.Pod{}, err
	}

	var pods []v1.Pod

	for _, pod := range nsPods.Items {
		pvcs := getPVCs(pod.Spec.Volumes)

		for _, pvc := range pvcs {
			if pvc.PersistentVolumeClaim.ClaimName == pvcToCheck.Name {
				m.log.WithField("pod", pod.Name).Debug("Found pod which mounts source PVC")
				pods = append(pods, pod)
			}
		}
	}

	return pods, nil
}

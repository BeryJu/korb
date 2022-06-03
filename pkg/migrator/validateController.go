package migrator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (m *Migrator) getPVCControllers(pvcToCheck *corev1.PersistentVolumeClaim) ([]interface{}, error) {
	pods, err := m.getPVCPods(pvcToCheck)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		for _, owner := range m.resolveOwner(pod.ObjectMeta, &appsv1.StatefulSet{}) {
			switch owner.(type) {
			case *appsv1.Deployment:
				m.log.Debug("Found deployment")
			case *appsv1.StatefulSet:
				m.log.Debug("Found statefulset")
			}
		}
	}

	return nil, nil
}

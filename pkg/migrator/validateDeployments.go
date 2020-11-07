package migrator

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (m *Migrator) getPVCDeployments(pvc *corev1.PersistentVolumeClaim) ([]*appsv1.Deployment, error) {
	pods, err := m.getPVCPods(pvc)
	if err != nil {
		return nil, err
	}

	affectedOwners := make([]*appsv1.Deployment, 0)
	for _, pod := range pods {
		for _, owner := range m.resolveOwner(pod.ObjectMeta, &appsv1.Deployment{}) {
			affectedOwners = append(affectedOwners, owner.(*appsv1.Deployment))
		}
	}

	return affectedOwners, nil
}

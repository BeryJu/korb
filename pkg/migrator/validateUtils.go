package migrator

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Migrator) resolveOwner(meta metav1.ObjectMeta, expectedType interface{}) []interface{} {
	m.log.WithField("meta", meta.Name).Debug("Walking owners")
	owners := make([]interface{}, 0)
	for _, owner := range meta.OwnerReferences {
		l := m.log.WithField("meta", meta.Name).WithField("owner", owner.Name).WithField("kind", owner.Kind)
		var ownerInstance interface{}
		var err error
		var meta metav1.ObjectMeta
		if owner.Kind == "ReplicaSet" {
			var rs *appsv1.ReplicaSet
			rs, err = m.kClient.AppsV1().ReplicaSets(m.SourceNamespace).Get(m.ctx, owner.Name, metav1.GetOptions{})
			ownerInstance = rs
			meta = rs.ObjectMeta
		} else if owner.Kind == "Deployment" {
			var deployment *appsv1.Deployment
			deployment, err = m.kClient.AppsV1().Deployments(m.SourceNamespace).Get(m.ctx, owner.Name, metav1.GetOptions{})
			ownerInstance = deployment
			meta = deployment.ObjectMeta
		}
		if err != nil {
			l.Warningf("Failed to get owning %s", owner.Kind)
			continue
		}
		owners = append(owners, m.resolveOwner(meta, expectedType)...)
		// if reflect.TypeOf(ownerInstance) == reflect.TypeOf(expectedType) {
		// 	l.Debug("Found matching owner")
		// }
		owners = append(owners, ownerInstance)
	}
	return owners
}

func getPVCs(volumes []v1.Volume) []v1.Volume {
	var pvcs []v1.Volume

	for _, volume := range volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			pvcs = append(pvcs, volume)
		}
	}

	return pvcs
}

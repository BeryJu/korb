package migrator

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Migrator) GetDestPVCSize(fallback resource.Quantity) resource.Quantity {
	var destSize resource.Quantity
	if m.DestPVCSize != "" {
		destSize = resource.MustParse(m.DestPVCSize)
	} else {
		destSize = fallback
	}
	return destSize
}

func (m *Migrator) GetDestinationPVCTemplate(sourcePVC *v1.PersistentVolumeClaim) *v1.PersistentVolumeClaim {
	var sc *string
	if m.DestPVCStorageClass != "" {
		sc = &m.DestPVCStorageClass
	}
	destPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.SourcePVCName,
			Namespace: m.kNS,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: sourcePVC.Spec.AccessModes,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): m.GetDestPVCSize(*sourcePVC.Spec.Resources.Requests.Storage()),
				},
			},
			StorageClassName: sc,
		},
	}
	return destPVC
}

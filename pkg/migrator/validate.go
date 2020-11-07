package migrator

import (
	"context"

	"github.com/BeryJu/korb/pkg/strategies"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Migrator) Validate() (*v1.PersistentVolumeClaim, []strategies.Strategy) {
	pvc := m.validateSourcePVC()
	controllers, err := m.getPVCControllers(pvc)
	if err != nil {
		m.log.WithError(err).Panic("Failed to get controllers")
	}
	baseStrategy := strategies.NewBaseStrategy(m.kConfig, m.kClient, m.kNS)
	allStrategies := strategies.StrategyInstances(baseStrategy)
	compatibleStrategies := make([]strategies.Strategy, 0)
	for _, strategy := range allStrategies {
		if strategy.CompatibleWithControllers(controllers...) {
			compatibleStrategies = append(compatibleStrategies, strategy)
		}
	}
	return pvc, compatibleStrategies
}

func (m *Migrator) validateSourcePVC() *v1.PersistentVolumeClaim {
	pvc, err := m.kClient.CoreV1().PersistentVolumeClaims(m.kNS).Get(context.TODO(), m.SourcePVCName, metav1.GetOptions{})
	if err != nil {
		m.log.WithError(err).Panic("Failed to get Source PVC")
	}
	m.log.WithField("uid", pvc.UID).WithField("name", pvc.Name).Debug("Got Source PVC")
	destPVCTemplate := m.GetDestinationPVCTemplate(pvc)
	sourceSize := pvc.Spec.Resources.Requests.Storage()
	destSize := destPVCTemplate.Spec.Resources.Requests.Storage()
	if sourceSize.Cmp(*destSize) == 1 {
		l := m.log.WithField("src-size", sourceSize.String()).WithField("destSize", destSize.String())
		if m.Force {
			l.Warning("Destination PVC is smaller than source, ignoring because force.")
		} else {
			l.Panic("Destination PVC is smaller than source.")
		}
	}
	if m.DestPVCName == "" {
		m.log.Debug("No new Name given, using old name")
		m.DestPVCName = pvc.Name
	}
	return pvc
}

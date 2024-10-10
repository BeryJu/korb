package migrator

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"beryju.org/korb/v2/pkg/strategies"
)

func (m *Migrator) Validate() (*v1.PersistentVolumeClaim, []strategies.Strategy) {
	pvc := m.validateSourcePVC()
	controllers, err := m.getPVCControllers(pvc)
	if err != nil {
		m.log.WithError(err).Panic("Failed to get controllers")
	}
	baseStrategy := strategies.NewBaseStrategy(&strategies.BaseStrategyOpts{
		Config:           m.kConfig,
		Client:           m.kClient,
		TolerateAllNodes: m.TolerateAllNodes,
		Timeout:          m.Timeout,
		CopyTimeout:      m.CopyTimeout,
		Ctx:              m.ctx,
	})
	allStrategies := strategies.StrategyInstances(baseStrategy)
	compatibleStrategies := make([]strategies.Strategy, 0)
	ctx := strategies.MigrationContext{
		PVCControllers: controllers,
		SourcePVC:      *pvc,
	}
	for _, strategy := range allStrategies {
		err := strategy.CompatibleWithContext(ctx)
		if err == nil {
			compatibleStrategies = append(compatibleStrategies, strategy)
		} else {
			m.log.WithError(err).Info("Strategy not compatible")
		}
	}
	return pvc, compatibleStrategies
}

func (m *Migrator) validateSourcePVC() *v1.PersistentVolumeClaim {
	pvc, err := m.kClient.CoreV1().PersistentVolumeClaims(m.SourceNamespace).Get(context.TODO(), m.SourcePVCName, metav1.GetOptions{})
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

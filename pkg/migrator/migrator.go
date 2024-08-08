package migrator

import (
	"time"

	"beryju.org/korb/v2/pkg/strategies"
	log "github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Migrator struct {
	SourceNamespace string
	SourcePVCName   string

	DestNamespace       string
	DestPVCStorageClass string
	DestPVCSize         string
	DestPVCName         string
	DestPVCAccessModes  []string

	Force                  bool
	WaitForTempDestPVCBind bool
	TolerateAllNodes       bool
	Timeout                *time.Duration

	kConfig *rest.Config
	kClient *kubernetes.Clientset

	log      *log.Entry
	strategy string
}

func New(kubeconfigPath string, strategy string, tolerateAllNode bool) *Migrator {
	m := &Migrator{
		log:              log.WithField("component", "migrator"),
		TolerateAllNodes: tolerateAllNode,
		strategy:         strategy,
	}
	if kubeconfigPath != "" {
		m.log.WithField("kubeconfig", kubeconfigPath).Debug("Created client from kubeconfig")
		cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{})

		// use the current context in kubeconfig
		config, err := cc.ClientConfig()

		if err != nil {
			m.log.WithError(err).Panic("Failed to get client config")
		}
		m.kConfig = config
		ns, _, err := cc.Namespace()
		if err != nil {
			m.log.WithError(err).Panic("Failed to get current namespace")
		} else {
			m.log.WithField("namespace", ns).Debug("Got current namespace")
			m.SourceNamespace = ns
			m.DestNamespace = ns
		}
	} else {
		m.log.Panic("Kubeconfig cannot be empty")
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(m.kConfig)
	if err != nil {
		panic(err.Error())
	}
	m.kClient = clientset
	return m
}

func (m *Migrator) Run() {
	sourcePVC, compatibleStrategies := m.Validate()
	m.log.Debug("Compatible Strategies:")
	for _, compatibleStrategy := range compatibleStrategies {
		m.log.WithField("identifier", compatibleStrategy.Identifier()).Debug(compatibleStrategy.Description())
	}
	destTemplate := m.GetDestinationPVCTemplate(sourcePVC)
	destTemplate.Name = m.DestPVCName

	var selected strategies.Strategy

	if len(compatibleStrategies) == 1 {
		m.log.Debug("Only one compatible strategy, running")
		selected = compatibleStrategies[0]
	} else {
		for _, strat := range compatibleStrategies {
			if strat.Identifier() == m.strategy {
				m.log.WithField("identifier", strat.Identifier()).Debug("User selected strategy")
				selected = strat
				break
			}
		}
	}
	if selected == nil {
		m.log.Error("No (compatible) strategy selected.")
		return
	}
	err := selected.Do(sourcePVC, destTemplate, m.WaitForTempDestPVCBind)
	if err != nil {
		m.log.WithError(err).Warning("Failed to migrate")
	}
}

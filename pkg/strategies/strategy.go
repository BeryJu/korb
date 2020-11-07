package strategies

import (
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type BaseStrategy struct {
	kConfig *rest.Config
	kClient *kubernetes.Clientset
	kNS     string

	log *log.Entry
}

func NewBaseStrategy(config *rest.Config, client *kubernetes.Clientset, ns string) BaseStrategy {
	return BaseStrategy{
		kConfig: config,
		kClient: client,
		kNS:     ns,
		log:     log.WithField("component", "strategy"),
	}
}

type Strategy interface {
	CompatibleWithControllers(...interface{}) bool
	Description() string
	Do(sourcePVC *v1.PersistentVolumeClaim, destTemplate *v1.PersistentVolumeClaim) error
}

func StrategyInstances(b BaseStrategy) []Strategy {
	s := []Strategy{
		NewCopyTwiceNameStrategy(b),
	}
	return s
}

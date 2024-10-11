package strategies

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type BaseStrategy struct {
	kConfig *rest.Config
	kClient *kubernetes.Clientset

	log              *log.Entry
	tolerateAllNodes bool
	timeout          time.Duration
	copyTimeout      *time.Duration
	ctx              context.Context
}

type BaseStrategyOpts struct {
	Config           *rest.Config
	Client           *kubernetes.Clientset
	TolerateAllNodes bool
	Timeout          *time.Duration
	CopyTimeout      *time.Duration
	Ctx              context.Context
}

func NewBaseStrategy(opts *BaseStrategyOpts) BaseStrategy {
	var t time.Duration
	if opts.Timeout == nil {
		t = 60 * time.Second
	} else {
		t = *opts.Timeout
	}
	return BaseStrategy{
		kConfig:          opts.Config,
		kClient:          opts.Client,
		tolerateAllNodes: opts.TolerateAllNodes,
		timeout:          t,
		copyTimeout:      opts.CopyTimeout,
		ctx:              opts.Ctx,
		log:              log.WithField("component", "strategy"),
	}
}

type Strategy interface {
	CompatibleWithContext(MigrationContext) error
	Description() string
	Identifier() string
	Do(sourcePVC *v1.PersistentVolumeClaim, destTemplate *v1.PersistentVolumeClaim, WaitForTempDestPVCBind bool) error
}

type MigrationContext struct {
	PVCControllers []interface{}
	SourcePVC      v1.PersistentVolumeClaim
}

func StrategyInstances(b BaseStrategy) []Strategy {
	s := []Strategy{
		NewCopyTwiceNameStrategy(b),
		NewExportStrategy(b),
		NewImportStrategy(b),
	}
	return s
}

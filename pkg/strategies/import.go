package strategies

import (
	"errors"
	"fmt"
	"io"
	"os"

	"beryju.org/korb/pkg/mover"
	"github.com/schollz/progressbar/v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

type ImportStrategy struct {
	BaseStrategy

	TempDestPVC *v1.PersistentVolumeClaim

	tempMover *mover.MoverJob
}

func NewImportStrategy(b BaseStrategy) *ImportStrategy {
	s := &ImportStrategy{
		BaseStrategy: b,
	}
	s.log = s.log.WithField("strategy", s.Identifier())
	return s
}

func (c *ImportStrategy) Identifier() string {
	return "import"
}

func (c *ImportStrategy) CompatibleWithContext(ctx MigrationContext) error {
	path := fmt.Sprintf("%s.tar", ctx.SourcePVC.Name)
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Expected import file '%s' does not exist", path)
	}
	return nil
}

func (c *ImportStrategy) Description() string {
	return "Import data into a PVC from a tar archive."
}

func (c *ImportStrategy) Do(sourcePVC *v1.PersistentVolumeClaim, destTemplate *v1.PersistentVolumeClaim, WaitForTempDestPVCBind bool) error {
	c.log.Warning("This strategy assumes you've stopped all pods accessing this data.")

	c.log.Debug("starting mover job")
	c.tempMover = mover.NewMoverJob(c.kClient, mover.MoverTypeSleep)
	c.tempMover.Namespace = destTemplate.ObjectMeta.Namespace
	c.tempMover.SourceVolume = sourcePVC
	c.tempMover.Name = fmt.Sprintf("korb-job-%s", sourcePVC.UID)

	pod := c.tempMover.Start().WaitForRunning()
	if pod == nil {
		c.log.Warning("Failed to move data")
		return c.Cleanup()
	}
	c.log.Debug("mover pod running, starting copy")

	err := c.CopyInto(*pod, c.kConfig, fmt.Sprintf("%s.tar", sourcePVC.Name))
	if err != nil {
		c.log.WithError(err).Warning("failed to copy file")
		return c.Cleanup()
	}
	c.log.Info("Finished copying into pvc")
	return c.Cleanup()
}

func (c *ImportStrategy) CopyInto(pod v1.Pod, config *rest.Config, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer file.Close()
	bar := progressbar.DefaultBytes(
		-1,
		"uploading",
	)
	err = c.tempMover.Exec(pod, config, []string{
		"tar", "xvf", "-",
	}, io.MultiReader(file, bar), os.Stdout)
	if err != nil {
		return err
	}
	return nil
}

func (c *ImportStrategy) Cleanup() error {
	c.log.Info("Cleaning up...")
	if c.tempMover != nil {
		c.tempMover.Cleanup()
	}
	return nil
}

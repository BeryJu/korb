// flag: export
// Behavior: Exports a tar archive of the pvc to your $pwd

package strategies

import (
	"fmt"
	"io"
	"os"

	"github.com/schollz/progressbar/v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"beryju.org/korb/v2/pkg/mover"
)

type ExportStrategy struct {
	BaseStrategy

	TempDestPVC *v1.PersistentVolumeClaim

	tempMover *mover.MoverJob
}

func NewExportStrategy(b BaseStrategy) *ExportStrategy {
	s := &ExportStrategy{
		BaseStrategy: b,
	}
	s.log = s.log.WithField("strategy", s.Identifier())
	return s
}

func (c *ExportStrategy) Identifier() string {
	return "export"
}

func (c *ExportStrategy) CompatibleWithContext(ctx MigrationContext) error {
	return nil
}

func (c *ExportStrategy) Description() string {
	return "Export PVC content into a tar archive."
}

func (c *ExportStrategy) Do(sourcePVC *v1.PersistentVolumeClaim, destTemplate *v1.PersistentVolumeClaim, WaitForTempDestPVCBind bool) error {
	c.log.Warning("This strategy assumes you've stopped all pods accessing this data.")

	c.log.Debug("starting mover job")
	c.tempMover = mover.NewMoverJob(c.ctx, c.kClient, mover.MoverTypeSleep, c.tolerateAllNodes)
	c.tempMover.Namespace = destTemplate.ObjectMeta.Namespace
	c.tempMover.SourceVolume = sourcePVC
	c.tempMover.Name = fmt.Sprintf("korb-job-%s", sourcePVC.UID)

	pod := c.tempMover.Start().WaitForRunning(c.timeout)
	if pod == nil {
		c.log.Warning("Failed to move data")
		return c.Cleanup()
	}
	c.log.Debug("mover pod running, starting copy")

	output, err := c.CopyOut(*pod, c.kConfig, sourcePVC.Name)
	if err != nil {
		c.log.WithError(err).Warning("failed to copy file")
		return c.Cleanup()
	}
	c.log.Info("Finished copying")
	c.log.Infof("Export at '%s'", output)
	return c.Cleanup()
}

func (c *ExportStrategy) CopyOut(pod v1.Pod, config *rest.Config, name string) (string, error) {
	file, err := os.CreateTemp(".", "korb-mover-")
	if err != nil {
		return "", err
	}
	defer file.Close()
	bar := progressbar.DefaultBytes(
		-1,
		"downloading",
	)
	cmd := []string{
		"bash",
		"-c",
		fmt.Sprintf("cd \"%s\" && tar cvzf - . ; sleep 5", mover.SourceMount),
	}
	err = c.tempMover.Exec(pod, config, cmd, nil, io.MultiWriter(file, bar))
	if err != nil {
		return "", err
	}
	finalPath := fmt.Sprintf("%s.tar", name)
	if err = os.Rename(file.Name(), finalPath); err != nil {
		return "", err
	}
	return finalPath, nil
}

func (c *ExportStrategy) Cleanup() error {
	c.log.Info("Cleaning up...")
	if c.tempMover != nil {
		return c.tempMover.Cleanup()
	}
	return nil
}

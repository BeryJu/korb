package strategies

import (
	"context"
	"fmt"
	"time"

	"beryju.org/korb/pkg/mover"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type CopyTwiceNameStrategy struct {
	BaseStrategy

	DestPVC     *v1.PersistentVolumeClaim
	TempDestPVC *v1.PersistentVolumeClaim

	tempMover  *mover.MoverJob
	finalMover *mover.MoverJob

	MoveTimeout time.Duration

	WaitForTempDestPVCBind bool

	pvcsToDelete []*v1.PersistentVolumeClaim
}

func NewCopyTwiceNameStrategy(b BaseStrategy) *CopyTwiceNameStrategy {
	s := &CopyTwiceNameStrategy{
		BaseStrategy: b,
		pvcsToDelete: make([]*v1.PersistentVolumeClaim, 0),
	}
	s.log = s.log.WithField("strategy", s.Identifier())
	return s
}

func (c *CopyTwiceNameStrategy) Identifier() string {
	return "copy-twice-name"
}

func (c *CopyTwiceNameStrategy) CompatibleWithControllers(...interface{}) bool {
	return true
}

func (c *CopyTwiceNameStrategy) Description() string {
	return "Copy the PVC to the new Storage class and with new size and a new name, delete the old PVC, and copy it back to the old name."
}

func (c *CopyTwiceNameStrategy) getDeleteOptions() metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}
}

func (c *CopyTwiceNameStrategy) Do(sourcePVC *v1.PersistentVolumeClaim, destTemplate *v1.PersistentVolumeClaim, WaitForTempDestPVCBind bool) error {
	c.setTimeout(destTemplate)
	c.log.Warning("This strategy assumes you've stopped all pods accessing this data.")
	suffix := time.Now().Unix()
	tempDest := destTemplate.DeepCopy()
	tempDest.Name = fmt.Sprintf("%s-copy-%d", tempDest.Name, suffix)

	c.log.WithField("stage", 1).Debug("creating temporary PVC")
	tempDestInst, err := c.kClient.CoreV1().PersistentVolumeClaims(destTemplate.ObjectMeta.Namespace).Create(context.TODO(), tempDest, metav1.CreateOptions{})
	c.TempDestPVC = tempDestInst
	if err != nil {
		return err
	}

	if c.WaitForTempDestPVCBind {
		err = c.waitForBound(tempDest)
		if err != nil {
			c.log.WithError(err).Warning("Waiting for PVC to be bound failed")
			return c.Cleanup()
		}
	} else {
		c.log.WithField("stage", 2).Debug("skipping waiting for PVC to be bound")
	}

	c.log.WithField("stage", 2).Debug("starting mover job")
	c.tempMover = mover.NewMoverJob(c.kClient, mover.MoverTypeSync)
	c.tempMover.Namespace = destTemplate.ObjectMeta.Namespace
	c.tempMover.SourceVolume = sourcePVC
	c.tempMover.DestVolume = c.TempDestPVC
	c.tempMover.Name = fmt.Sprintf("korb-job-%s", sourcePVC.UID)
	err = c.tempMover.Start().Wait(c.MoveTimeout)
	if err != nil {
		c.log.WithError(err).Warning("Failed to move data")
		c.pvcsToDelete = []*v1.PersistentVolumeClaim{c.TempDestPVC}
		return c.Cleanup()
	}

	c.log.WithField("stage", 3).Debug("deleting original PVC")
	err = c.kClient.CoreV1().PersistentVolumeClaims(sourcePVC.ObjectMeta.Namespace).Delete(context.TODO(), sourcePVC.Name, c.getDeleteOptions())
	if err != nil {
		c.log.WithError(err).Warning("Failed to delete source pvc")
		return c.Cleanup()
	}
	c.waitForPVCDeletion(sourcePVC)

	c.log.WithField("stage", 4).Debug("creating final destination PVC")
	destInst, err := c.kClient.CoreV1().PersistentVolumeClaims(destTemplate.ObjectMeta.Namespace).Create(context.TODO(), destTemplate, metav1.CreateOptions{})
	if err != nil {
		c.log.WithError(err).Warning("Failed to create final pvc")
		return c.Cleanup()
	}
	c.DestPVC = destInst

	c.log.WithField("stage", 5).Debug("starting mover job to final PVC")
	c.finalMover = mover.NewMoverJob(c.kClient, mover.MoverTypeSync)
	c.finalMover.Namespace = destTemplate.ObjectMeta.Namespace
	c.finalMover.SourceVolume = c.TempDestPVC
	c.finalMover.DestVolume = c.DestPVC
	c.finalMover.Name = fmt.Sprintf("korb-job-%s", tempDestInst.UID)
	err = c.finalMover.Start().Wait(c.MoveTimeout)
	if err != nil {
		c.log.WithError(err).Warning("Failed to move data")
		c.pvcsToDelete = []*v1.PersistentVolumeClaim{c.DestPVC}
		return c.Cleanup()
	}

	c.log.WithField("stage", 6).Debug("deleting temporary PVC")
	err = c.kClient.CoreV1().PersistentVolumeClaims(destTemplate.ObjectMeta.Namespace).Delete(context.TODO(), c.TempDestPVC.Name, c.getDeleteOptions())
	if err != nil {
		c.log.WithError(err).Warning("Failed to delete temporary destination pvc")
		return c.Cleanup()
	}
	c.waitForPVCDeletion(c.TempDestPVC)

	c.log.Info("And we're done")

	return c.Cleanup()
}

func (c *CopyTwiceNameStrategy) Cleanup() error {
	c.log.Info("Cleaning up...")
	for _, pvc := range c.pvcsToDelete {
		err := c.kClient.CoreV1().PersistentVolumeClaims(pvc.ObjectMeta.Namespace).Delete(context.Background(), pvc.Name, metav1.DeleteOptions{})
		if err != nil {
			c.log.WithError(err).Warning("Error during temporary PVC cleanup, continuing")
		}
	}
	return nil
}

func (c *CopyTwiceNameStrategy) setTimeout(pvc *v1.PersistentVolumeClaim) {
	sizeInByes, _ := pvc.Spec.Resources.Requests.Storage().AsInt64()
	sizeInGB := sizeInByes / 1024 / 1024 / 1024
	c.MoveTimeout = time.Duration(sizeInGB*60) * time.Second
	c.log.WithField("timeout", c.MoveTimeout).Debug("Set timeout from PVC size")
}

func (c *CopyTwiceNameStrategy) waitForPVCDeletion(pvc *v1.PersistentVolumeClaim) error {
	return wait.Poll(2*time.Second, 30*time.Second, func() (bool, error) {
		_, err := c.kClient.CoreV1().PersistentVolumeClaims(pvc.ObjectMeta.Namespace).Get(context.TODO(), pvc.ObjectMeta.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		c.log.WithField("pvc-name", pvc.ObjectMeta.Name).Debug("Waiting for PVC Deletion, retrying")
		return false, nil
	})
}

func (c *CopyTwiceNameStrategy) waitForBound(pvc *v1.PersistentVolumeClaim) error {
	return wait.Poll(2*time.Second, 30*time.Second, func() (bool, error) {
		pvc, err := c.kClient.CoreV1().PersistentVolumeClaims(pvc.ObjectMeta.Namespace).Get(context.TODO(), pvc.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if pvc.Status.Phase != v1.ClaimBound {
			c.log.WithField("pvc-name", pvc.ObjectMeta.Name).Warning("PVC not bound yet, retrying")
			return false, nil
		}
		return true, nil
	})
}

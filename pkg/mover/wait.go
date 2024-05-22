package mover

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (m *MoverJob) getPods(ctx context.Context) []v1.Pod {
	selector := fmt.Sprintf("job-name=%s", m.Name)
	pods, err := m.kClient.CoreV1().Pods(m.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		m.log.WithError(err).Warning("Failed to get pods")
		return make([]v1.Pod, 0)
	}
	return pods.Items
}

func (m *MoverJob) WaitForRunning(timeout time.Duration) *v1.Pod {
	// First we wait for all pods to be running
	var runningPod v1.Pod
	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		pods := m.getPods(ctx)
		if len(pods) != 1 {
			return false, nil
		}
		pod := pods[0]
		if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodSucceeded {
			runningPod = pod
			return true, nil
		}
		m.log.WithField("phase", pod.Status.Phase).Debug("Pod not in correct state yet")
		return false, nil
	})
	if err != nil {
		m.log.WithError(err).Warning("failed to wait for pod to be running")
		return nil
	}
	return &runningPod
}

func (m *MoverJob) Wait(startTimeout time.Duration, moveTimeout time.Duration) error {
	pod := m.WaitForRunning(startTimeout)
	if pod == nil {
		return errors.New("pod not in correct state")
	}
	runningPod := *pod
	go m.followLogs(runningPod)

	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, moveTimeout, true, func(ctx context.Context) (bool, error) {
		job, err := m.kClient.BatchV1().Jobs(m.Namespace).Get(ctx, m.kJob.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if job.Status.Succeeded != int32(len(job.Spec.Template.Spec.Containers)) {
			return false, nil
		}
		return true, nil
	})

	if err == nil {
		// Job was run successfully, so we delete it to cleanup
		m.log.Debug("Cleaning up successful job")
		return m.Cleanup()
	}
	return err
}

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

func (m *MoverJob) getPods() []v1.Pod {
	selector := fmt.Sprintf("job-name=%s", m.Name)
	pods, err := m.kClient.CoreV1().Pods(m.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		m.log.WithError(err).Warning("Failed to get pods")
		return make([]v1.Pod, 0)
	}
	return pods.Items
}

func (m *MoverJob) WaitForRunning() *v1.Pod {
	// First we wait for all pods to be running
	var runningPod v1.Pod
	err := wait.PollImmediate(15*time.Second, 120*time.Second, func() (bool, error) {
		pods := m.getPods()
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

func (m *MoverJob) Wait(timeout time.Duration) error {
	pod := m.WaitForRunning()
	if pod == nil {
		return errors.New("pod not in correct state")
	}
	runningPod := *pod
	go m.followLogs(runningPod)

	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		job, err := m.kClient.BatchV1().Jobs(m.Namespace).Get(context.TODO(), m.kJob.Name, metav1.GetOptions{})
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

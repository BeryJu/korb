package mover

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/BeryJu/korb/pkg/config"
	"github.com/goware/prefixer"
	log "github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type MoverJob struct {
	Name         string
	Namespace    string
	SourceVolume *corev1.PersistentVolumeClaim
	DestVolume   *corev1.PersistentVolumeClaim

	kJob    *batchv1.Job
	kClient *kubernetes.Clientset

	log *log.Entry
}

func NewMoverJob(client *kubernetes.Clientset) *MoverJob {
	return &MoverJob{
		kClient: client,
		log:     log.WithField("component", "mover-job"),
	}
}

func (m *MoverJob) Start() *MoverJob {
	volumes := []corev1.Volume{
		{
			Name: "source",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.SourceVolume.Name,
					ReadOnly:  false,
				},
			},
		},
		{
			Name: "dest",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.DestVolume.Name,
					ReadOnly:  false,
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"sidecar.istio.io/inject": "false",
					},
				},
				Spec: corev1.PodSpec{
					Volumes:       volumes,
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "mover",
							Image: config.DockerImage,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "source",
									MountPath: "/source",
								},
								{
									Name:      "dest",
									MountPath: "/dest",
								},
							},
						},
					},
				},
			},
		},
	}
	j, err := m.kClient.BatchV1().Jobs(m.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		// temp
		panic(err)
	}
	m.kJob = j
	return m
}

func (m *MoverJob) followLogs(pod v1.Pod) {
	req := m.kClient.CoreV1().Pods(m.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Follow: true})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		m.log.WithError(err).Warning("error opening log stream")
		return
	}
	defer podLogs.Close()
	prefixReader := prefixer.New(podLogs, "[mover logs]: ")

	for {
		io.Copy(os.Stdout, prefixReader)
	}
}

func (m *MoverJob) Wait(timeout time.Duration) error {
	// First we wait for all pods to be running
	var runningPod v1.Pod
	err := wait.Poll(2*time.Second, 60*time.Second, func() (bool, error) {
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
	go m.followLogs(runningPod)

	err = wait.Poll(2*time.Second, timeout, func() (bool, error) {
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

func (m *MoverJob) getPods() []v1.Pod {
	selector := fmt.Sprintf("job-name=%s", m.Name)
	pods, err := m.kClient.CoreV1().Pods(m.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		m.log.WithError(err).Warning("Failed to get pods")
		return make([]v1.Pod, 0)
	}
	return pods.Items
}

func (m *MoverJob) Cleanup() error {
	err := m.kClient.BatchV1().Jobs(m.Namespace).Delete(context.TODO(), m.Name, metav1.DeleteOptions{})
	if err != nil {
		m.log.WithError(err).Debug("Failed to delete job")
		return err
	}
	pods := m.getPods()
	for _, pod := range pods {
		m.kClient.CoreV1().Pods(m.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	}
	return nil
}

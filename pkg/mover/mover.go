package mover

import (
	"context"
	"io"
	"os"

	"beryju.org/korb/pkg/config"
	"github.com/goware/prefixer"
	log "github.com/sirupsen/logrus"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ContainerName = "mover"
)

type MoverType string

const (
	MoverTypeSync  MoverType = "sync"
	MoverTypeSleep MoverType = "sleep"
)

const SourceMount = "/source"
const DestMount = "/dest"

type MoverJob struct {
	Name         string
	Namespace    string
	SourceVolume *corev1.PersistentVolumeClaim
	DestVolume   *corev1.PersistentVolumeClaim

	kJob    *batchv1.Job
	kClient *kubernetes.Clientset

	mode MoverType
	log  *log.Entry
}

func NewMoverJob(client *kubernetes.Clientset, mode MoverType) *MoverJob {
	return &MoverJob{
		kClient: client,
		log:     log.WithField("component", "mover-job"),
		mode:    mode,
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
	}
	mounts := []corev1.VolumeMount{
		{
			Name:      "source",
			MountPath: SourceMount,
		},
	}
	if m.mode == MoverTypeSync {
		volumes = append(volumes, corev1.Volume{
			Name: "dest",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: m.DestVolume.Name,
					ReadOnly:  false,
				},
			},
		})
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "dest",
			MountPath: DestMount,
		})
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
						"linkerd.io/inject":       "disabled",
					},
				},
				Spec: corev1.PodSpec{
					Volumes:       volumes,
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:            ContainerName,
							Image:           config.ContainerImage,
							ImagePullPolicy: v1.PullAlways,
							Args:            []string{string(m.mode)},
							VolumeMounts:    mounts,
						},
					},
				},
			},
		},
	}
	j, err := m.kClient.BatchV1().Jobs(m.Namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	m.kJob = j
	return m
}

func (m *MoverJob) followLogs(pod v1.Pod) {
	req := m.kClient.CoreV1().Pods(m.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow:    true,
		Container: ContainerName,
	})
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

func (m *MoverJob) getDeleteOptions() metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}
}

func (m *MoverJob) Cleanup() error {
	err := m.kClient.BatchV1().Jobs(m.Namespace).Delete(context.TODO(), m.Name, m.getDeleteOptions())
	if err != nil {
		m.log.WithError(err).Debug("Failed to delete job")
		return err
	}
	pods := m.getPods()
	for _, pod := range pods {
		m.kClient.CoreV1().Pods(m.Namespace).Delete(context.TODO(), pod.Name, m.getDeleteOptions())
	}
	return nil
}

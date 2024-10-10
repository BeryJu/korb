package mover

import (
	"context"
	"io"
	"os"

	"github.com/goware/prefixer"
	log "github.com/sirupsen/logrus"

	"beryju.org/korb/v2/pkg/config"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

const (
	SourceMount = "/source"
	DestMount   = "/dest"
)

type MoverJob struct {
	Name         string
	Namespace    string
	SourceVolume *corev1.PersistentVolumeClaim
	DestVolume   *corev1.PersistentVolumeClaim

	kJob    *batchv1.Job
	kClient *kubernetes.Clientset

	mode             MoverType
	log              *log.Entry
	tolerateAllNodes bool
	ctx              context.Context
}

func NewMoverJob(ctx context.Context, client *kubernetes.Clientset, mode MoverType, tolerateAllNodes bool) *MoverJob {
	return &MoverJob{
		kClient:          client,
		log:              log.WithField("component", "mover-job"),
		tolerateAllNodes: tolerateAllNodes,
		mode:             mode,
		ctx:              ctx,
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
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{string(m.mode)},
							VolumeMounts:    mounts,
							TTY:             true,
							Stdin:           true,
						},
					},
				},
			},
		},
	}

	if m.tolerateAllNodes {
		job.Spec.Template.Spec.Tolerations = []corev1.Toleration{
			{
				Operator: corev1.TolerationOpExists,
			},
		}
	}

	j, err := m.kClient.BatchV1().Jobs(m.Namespace).Create(m.ctx, job, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	m.kJob = j
	return m
}

func (m *MoverJob) followLogs(pod corev1.Pod) {
	req := m.kClient.CoreV1().Pods(m.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Follow:    true,
		Container: ContainerName,
	})
	podLogs, err := req.Stream(m.ctx)
	if err != nil {
		m.log.WithError(err).Warning("error opening log stream")
		return
	}
	defer podLogs.Close()
	prefixReader := prefixer.New(podLogs, "[mover logs]: ")

	for {
		_, err := io.Copy(os.Stdout, prefixReader)
		if err != nil && err == io.EOF {
			m.log.Debug("log stream complete")
			break
		}

		if err != nil {
			m.log.WithError(err).Warning("failed to copy")
		}
	}
}

func (m *MoverJob) getDeleteOptions() metav1.DeleteOptions {
	policy := metav1.DeletePropagationForeground
	return metav1.DeleteOptions{
		PropagationPolicy: &policy,
	}
}

func (m *MoverJob) Cleanup() error {
	err := m.kClient.BatchV1().Jobs(m.Namespace).Delete(m.ctx, m.Name, m.getDeleteOptions())
	if err != nil {
		m.log.WithError(err).WithField("name", m.Name).Debug("Failed to delete job")
		return err
	}
	pods := m.getPods(m.ctx)
	for _, pod := range pods {
		err := m.kClient.CoreV1().Pods(m.Namespace).Delete(m.ctx, pod.Name, m.getDeleteOptions())
		if err != nil {
			m.log.WithError(err).WithField("name", pod.Name).Warning("failed to delete pod")
		}
	}
	return nil
}

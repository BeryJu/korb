package mover

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/goware/prefixer"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func (m *MoverJob) Exec(pod v1.Pod, config *rest.Config, cmd []string, input io.Reader, output io.Writer) error {
	req := m.kClient.CoreV1().RESTClient().Post().Resource("pods").Name(pod.Name).Namespace(m.Namespace).SubResource("exec")
	req.VersionedParams(
		&v1.PodExecOptions{
			Container: ContainerName,
			Command:   cmd,
			Stdin:     input != nil,
			Stdout:    true,
			Stderr:    true,
		},
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}
	errBuff := bytes.NewBuffer([]byte{})
	prefixReader := prefixer.New(errBuff, "[mover logs]: ")
	done := false
	go func() {
		for {
			_, err := io.Copy(os.Stdout, prefixReader)
			if err != nil && err == io.EOF {
				m.log.Debug("log stream complete")
				break
			}

			if err != nil {
				m.log.WithError(err).Warning("failed to copy")
			}
			if done {
				return
			}
		}
	}()
	err = exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  input,
		Stdout: output,
		Stderr: os.Stdout,
	})
	done = true
	if err != nil {
		return err
	}
	return nil
}

package pods

import (
	"context"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetPods(clientSet *kubernetes.Clientset, namespace string) (*corev1.PodList, error) {

	pods, err := clientSet.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return pods, nil

}

func GetPodLogs(clientSet *kubernetes.Clientset, namespace string, podName string, follow bool, sinceSeconds *int64, sinceTime *metav1.Time, timestamps bool, tailLines *int64, bufferSize int64) (io.ReadCloser, string, error) {

	podLogOptions := corev1.PodLogOptions{
		Follow:       follow,
		SinceSeconds: sinceSeconds,
		SinceTime:    sinceTime,
		Timestamps:   timestamps,
		TailLines:    tailLines,
	}

	podLogRequest := clientSet.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	stream, err := podLogRequest.Stream(context.TODO())
	if err != nil {
		return nil, "", err
	}

	if follow {
		return stream, "", nil
	}
	defer stream.Close()

	var logs string
	buf := make([]byte, bufferSize)
	for {
		numBytes, err := stream.Read(buf)
		if err == io.EOF {
			break
		}
		if numBytes == 0 {
			time.Sleep(time.Second)
			continue
		}
		if err != nil {
			return nil, "", err
		}
		message := string(buf[:numBytes])
		logs += message
	}
	return nil, logs, nil
}

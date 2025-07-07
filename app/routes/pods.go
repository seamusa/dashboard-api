package routes

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	repo "github.com/chechetech/app/azure-go/repositories/pods"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CustomPodStatus is a custom struct to hold the desired fields
type CustomPodStatus struct {
	Image     string    `json:"image"`
	Name      string    `json:"name"`
	Phase     string    `json:"phase"`
	StartTime time.Time `json:"startTime"`
}

func RegisterPodsRoutes(r *gin.Engine) {
	r.GET("/pods", func(c *gin.Context) {
		// Set up Kubernetes client

		getClientset, exists := c.Get("clientset")
		if !exists {
			c.JSON(500, gin.H{"error": "clientset not found"})
			return
		}
		clientset := getClientset.(*kubernetes.Clientset)

		getNamespace, exists := c.Get("namespace")
		if !exists {
			c.JSON(500, gin.H{"error": "namespace not found"})
			return
		}
		namespace := getNamespace.(string)

		pods, err := repo.GetPods(clientset, namespace)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to get pods: %v", err)})
			return
		}
		// Create a slice to hold the custom pod statuses
		var customPodStatuses []CustomPodStatus

		// Populate the custom pod statuses
		for _, pod := range pods.Items {
			customPodStatus := CustomPodStatus{
				Image:     pod.Spec.Containers[0].Image,
				Name:      pod.Name,
				Phase:     string(pod.Status.Phase),
				StartTime: pod.Status.StartTime.Time,
			}
			customPodStatuses = append(customPodStatuses, customPodStatus)
		}

		c.JSON(http.StatusOK, customPodStatuses)

	})

	r.GET("/pods/:name/logs", func(c *gin.Context) {

		getClientset, exists := c.Get("clientset")
		if !exists {
			c.JSON(500, gin.H{"error": "clientset not found"})
			return
		}
		clientset := getClientset.(*kubernetes.Clientset)

		getNamespace, exists := c.Get("namespace")
		if !exists {
			c.JSON(500, gin.H{"error": "namespace not found"})
			return
		}
		namespace := getNamespace.(string)

		fmt.Println("namespace", namespace)

		podName := c.Param("name")

		follow, err := strconv.ParseBool(c.Query("follow"))
		if err != nil {
			follow = false
		}

		var sinceSeconds *int64
		if sinceSecondsStr := c.Query("sinceSeconds"); sinceSecondsStr != "" {
			parsedSinceSeconds, err := strconv.ParseInt(sinceSecondsStr, 10, 64)
			if err == nil {
				sinceSeconds = &parsedSinceSeconds
			}
		}

		var sinceTime *metav1.Time
		if sinceTimeStr := c.Query("sinceTime"); sinceTimeStr != "" {
			parsedSinceTime, err := time.Parse(time.RFC3339, sinceTimeStr)
			fmt.Println("parsedSinceTime", parsedSinceTime)
			if err == nil {
				fmt.Println("error parsing time", err)
				sinceTime = &metav1.Time{Time: parsedSinceTime}
			}
		}

		timestamps, err := strconv.ParseBool(c.Query("timestamps"))
		if err != nil {
			timestamps = false
		}

		var tailLines *int64
		if tailLinesStr := c.Query("tailLines"); tailLinesStr != "" {
			parsedTailLines, err := strconv.ParseInt(tailLinesStr, 10, 64)
			if err == nil {
				tailLines = &parsedTailLines
			}
		} else {
			defaultTailLines := int64(100)
			tailLines = &defaultTailLines
		}

		bufferSize := int64(16384)

		logStream, logs, err := repo.GetPodLogs(clientset, namespace, podName, follow, sinceSeconds, sinceTime, timestamps, tailLines, bufferSize)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to get logs: %v", err)})
			return
		}

		if follow {
			fmt.Println("streaming started.")
			defer logStream.Close()

			ctx := c.Request.Context()

			go func() {
				<-ctx.Done()
				fmt.Println("client disconnected, closing stream.")
				logStream.Close()
			}()

			c.Stream(func(w io.Writer) bool {
				buf := make([]byte, bufferSize)
				numBytes, err := logStream.Read(buf)
				if err != nil {
					return false
				}
				if numBytes > 0 {
					c.Writer.Write(buf[:numBytes])
				}
				fmt.Println("streaming ending.")
				return true

			})
		} else {
			c.String(200, logs)
		}

	})

}

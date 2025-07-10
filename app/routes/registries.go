package routes

import (
	"fmt"
	"net/http"
	"strings"

	repo "github.com/chechetech/app/azure-go/repositories/registries"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
)

type WebhookTarget struct {
	MediaType  string
	Size       int32
	Digest     string
	Length     int32
	Repository string
	Tag        string
}

type WebhookRequest struct {
	ID        string
	Host      string
	Method    string
	UserAgent string
}

type Webhook struct {
	ID        string
	Timestamp string
	Action    string
	Target    WebhookTarget
	Request   WebhookRequest
}

func RegisterRegistriesRoutes(router *gin.Engine) {
	router.POST("/update-deployment", func(c *gin.Context) {

		fmt.Printf("Request IP: %s\n", c.ClientIP())
		fmt.Printf("Request User-Agent: %s\n", c.Request.UserAgent())
		fmt.Printf("Request Referer: %s\n", c.Request.Referer())

		getClientset, exists := c.Get("clientset")
		if !exists {
			c.JSON(500, gin.H{"error": "clientset not found"})
			return
		}
		clientset := getClientset.(*kubernetes.Clientset)

		var webhook Webhook
		if err := c.ShouldBindJSON(&webhook); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fmt.Println("Webhook Target Repository: ", webhook.Target.Repository)

		company := strings.SplitN(webhook.Target.Repository, "/", 2)[0]
		var namespace, deploymentName, imagePath string

		if company == "cheche" {
			parts := strings.SplitN(webhook.Target.Repository, "/", 2)
			namespace = parts[0]
			deploymentName = strings.ReplaceAll(parts[1], "/", "-")
			imagePath = fmt.Sprintf("%s/%s:%s", webhook.Request.Host, webhook.Target.Repository, webhook.Target.Tag)
		} else {
			parts := strings.SplitN(webhook.Target.Repository, "/", 3)
			namespace = parts[1]
			deploymentName = strings.ReplaceAll(parts[2], "/", "-")
			imagePath = fmt.Sprintf("%s/%s:%s", webhook.Request.Host, webhook.Target.Repository, webhook.Target.Tag)
		}

		fmt.Println("Namespace: ", namespace, "Deployment Name: ", deploymentName, "Image Path: ", imagePath)

		err := repo.UpdateDeploymentImage(clientset, namespace, deploymentName, imagePath)
		if err != nil {
			fmt.Printf("Error updating deployment: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deployment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Deployment updated successfully"})
	})
}

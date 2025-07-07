package registries

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func UpdateDeploymentImage(clientset *kubernetes.Clientset, namespace, deploymentName, imagePath string) error {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %v", err)
	}

	originalImage := deployment.Spec.Template.Spec.Containers[0].Image
	deployment.Spec.Template.Spec.Containers[0].Image = imagePath

	_, updateErr := clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, v1.UpdateOptions{})
	if updateErr != nil {
		deployment.Spec.Template.Spec.Containers[0].Image = originalImage
		_, revertErr := clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, v1.UpdateOptions{})
		if revertErr != nil {
			return fmt.Errorf("update failed and revert also failed: %v", revertErr)
		}
		return fmt.Errorf("update failed but reverted to original image: %v", updateErr)
	}

	return nil
}

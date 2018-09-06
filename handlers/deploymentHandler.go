package handlers

import (
	"fmt"

	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type deploymentHandler struct {
	kubeClient *kubernetes.Clientset
	logger     *zap.SugaredLogger
}

func newDeploymentHandler(clientset *kubernetes.Clientset, logger *zap.SugaredLogger) *deploymentHandler {
	return &deploymentHandler{
		kubeClient: clientset,
		logger:     logger,
	}
}

func (h *deploymentHandler) apply(rawData runtime.Object) string {
	deployment := rawData.(*appsv1.Deployment)
	deploymentsClient := h.kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault)
	name := deployment.ObjectMeta.Name
	current, getErr := deploymentsClient.Get(name, metav1.GetOptions{})

	if current != nil && getErr == nil {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			current.ObjectMeta.Labels = deployment.ObjectMeta.Labels
			current.ObjectMeta.Annotations = deployment.ObjectMeta.Annotations
			current.Spec = deployment.Spec
			_, err := deploymentsClient.Update(current)
			return err
		})
		if err != nil {
			msg := fmt.Sprintf("update deployment err: %s\n", err.Error())
			h.logger.Errorf(msg)
			return msg
		}
		msg := fmt.Sprintf("update deployment %q\n", name)
		h.logger.Infof(msg)
		return msg
	} else if errors.IsNotFound(getErr) {
		result, err := deploymentsClient.Create(deployment)
		if err != nil {
			msg := fmt.Sprintf("create deployment err: %s\n", err.Error())
			h.logger.Errorf(msg)
			return msg
		}
		msg := fmt.Sprintf("create deployment %q\n", result.GetObjectMeta().GetName())
		h.logger.Infof(msg)
		return msg
	} else {
		msg := fmt.Sprintf("get deployment err: %s\n", getErr.Error())
		h.logger.Errorf(msg)
		return msg
	}
}

func (h *deploymentHandler) delete(rawData runtime.Object) string {
	deployment := rawData.(*appsv1.Deployment)
	deploymentsClient := h.kubeClient.AppsV1().Deployments(apiv1.NamespaceDefault)
	name := deployment.ObjectMeta.Name
	current, getErr := deploymentsClient.Get(name, metav1.GetOptions{})

	if current != nil && getErr == nil {
		deletePolicy := metav1.DeletePropagationForeground
		if err := deploymentsClient.Delete(name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			msg := fmt.Sprintf("delete deployment err: %s\n", err.Error())
			h.logger.Errorf(msg)
			return msg
		}
		msg := fmt.Sprintf("delete deployment %q\n", name)
		h.logger.Infof(msg)
		return msg
	} else if errors.IsNotFound(getErr) {
		msg := fmt.Sprintf("deployment does not exist: %s\n", name)
		h.logger.Infof(msg)
		return msg
	} else {
		msg := fmt.Sprintf("get deployment err: %s\n", getErr.Error())
		h.logger.Errorf(msg)
		return msg
	}
}
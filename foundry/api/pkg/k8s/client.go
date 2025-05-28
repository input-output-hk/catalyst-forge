package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/client.go . Client

// Client defines the interface for Kubernetes operations
type Client interface {
	CreateDeployment(ctx context.Context, deployment *models.ReleaseDeployment) error
}

// K8sClient implements the Client interface
type K8sClient struct {
	dynamicClient dynamic.Interface
	namespace     string
	logger        *slog.Logger
}

// New creates a new Kubernetes client
func New(namespace string, logger *slog.Logger) (Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if os.Getenv("KUBECONFIG") != "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
		}

		logger.Info("Using kubeconfig file", "path", kubeconfig)
	} else {
		logger.Info("Using in-cluster Kubernetes configuration")
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes dynamic client: %w", err)
	}

	if namespace == "" {
		namespace = "default"
	}

	return &K8sClient{
		dynamicClient: dynamicClient,
		namespace:     namespace,
		logger:        logger,
	}, nil
}

// CreateDeployment creates a new Kubernetes custom resource for the deployment
func (c *K8sClient) CreateDeployment(ctx context.Context, deployment *models.ReleaseDeployment) error {
	c.logger.Info("Creating Kubernetes release deployment resource",
		"deploymentID", deployment.ID,
		"releaseID", deployment.ReleaseID)

	gvr := schema.GroupVersionResource{
		Group:    "foundry.projectcatalyst.io",
		Version:  "v1alpha1",
		Resource: "releasedeployments",
	}

	deploymentObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "foundry.projectcatalyst.io/v1alpha1",
			"kind":       "ReleaseDeployment",
			"metadata": map[string]interface{}{
				"name": deployment.ID,
			},
			"spec": map[string]interface{}{
				"id":         deployment.ID,
				"release_id": deployment.ReleaseID,
			},
		},
	}

	_, err := c.dynamicClient.Resource(gvr).Namespace(c.namespace).Create(ctx, deploymentObj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes resource: %w", err)
	}

	return nil
}

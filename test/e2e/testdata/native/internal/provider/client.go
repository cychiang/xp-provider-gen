package provider

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client talks to the external system: a Kubernetes API reached with the
// kubeconfig the ProviderConfig's credentials hold.
type Client struct {
	Kube     kubernetes.Interface
	Endpoint string
	Region   string
}

// NewClient builds a Client from the resolved ProviderConfig.
func NewClient(_ context.Context, cfg ClientConfig) (*Client, error) {
	rc, err := clientcmd.RESTConfigFromKubeConfig(cfg.Credentials)
	if err != nil {
		return nil, fmt.Errorf("parsing kubeconfig credentials: %w", err)
	}
	kube, err := kubernetes.NewForConfig(rc)
	if err != nil {
		return nil, fmt.Errorf("building kubernetes client: %w", err)
	}
	return &Client{Kube: kube, Endpoint: cfg.Spec.Endpoint, Region: *Region}, nil
}

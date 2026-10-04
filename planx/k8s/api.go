package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	client *kubernetes.Clientset
)

func InitClient(ns string) error {
	if ns == "" {
		// not allowed for default namespace
		return ErrInvalidNamespace
	}
	var err error
	var cfg *rest.Config
	cfg, err = rest.InClusterConfig()
	if err != nil {
		return err
	}
	client, err = kubernetes.NewForConfig(cfg)
	if err == nil {
		// set the default namespace
		defaultId = defaultId.Namespace(ns)
	}
	return err
}

func Client() *kubernetes.Clientset { return client }

func IsInitialized() bool {
	return client != nil
}

package k8s

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

var ()

func ListDeployments(ctx context.Context, namespace string) ([]appv1.Deployment, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	if namespace == "" {
		return nil, ErrInvalidNamespace
	}
	deployments, err := Client().AppsV1().Deployments(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return deployments.Items, nil
}

func GetDeployment(ctx context.Context, id ResourceId) (*appv1.Deployment, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	if err := id.IsValid(); err != nil {
		return nil, err
	}

	namespace := id.namespace
	name := id.name

	deployment, err := Client().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

// GetDeploymentPods gets the pods of the deployment.
func GetDeploymentPods(ctx context.Context, id ResourceId) ([]corev1.Pod, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	return ListResourcePods(ctx, ListPodsParam[*appv1.Deployment]{
		Id:   id,
		Warp: GenWrapDeployment,
		Get:  GetDeployment,
	})
}

// ScaleDeployment scales the deployment to the specified number of replicas.
func ScaleDeployment(ctx context.Context, param ScaleParam) error {
	if !IsInitialized() {
		return ErrNotInitialized
	}
	return ScaleResourcePods[
		*appv1.Deployment, v1.DeploymentInterface,
	](ctx, scaleDeployment, param)
}

// scaleDeployment is scale for deployments
var scaleDeployment = ScaleControl[v1.DeploymentInterface]{
	Control: func(ns string) v1.DeploymentInterface {
		return Client().AppsV1().Deployments(ns)
	},
}

// warp for scale deployment

type WrapDeployment struct{ *appv1.Deployment }

func (w WrapDeployment) GetReplicas() int32                 { return int32Val(w.Spec.Replicas) }
func (w WrapDeployment) SetReplicas(replicas int32)         { w.Spec.Replicas = &replicas }
func (w WrapDeployment) GetResourceVersion() string         { return w.Deployment.GetResourceVersion() }
func (w WrapDeployment) GetSelector() *metav1.LabelSelector { return w.Spec.Selector }

// GenWrapDeployment is a function that generates a WrapDeployment object
func GenWrapDeployment(deployment *appv1.Deployment) IReplicaResource {
	return WrapDeployment{deployment}
}

type DeploymentWatchAction struct {
	WatchParams
	End bool
}

func GenDeploymentWatchAction(param WatchParams) IWatchAction[*appv1.Deployment, *appv1.DeploymentList] {
	return &DeploymentWatchAction{
		WatchParams: param,
	}
}

// LogDeploymentStatus logs the deployment status.
func (p *DeploymentWatchAction) LogDeploymentStatus(typ watch.EventType, deployment *appv1.Deployment) {

	// log deployment status
	p.Logger.Infof(
		"[%v]deployment status revision: %s, replicas: %d, ready replicas: %d, unavailable replicas: %d",
		typ,
		deployment.GetResourceVersion(),
		deployment.Status.Replicas,
		deployment.Status.ReadyReplicas,
		deployment.Status.UnavailableReplicas,
	)
}

// Init initializes the action. maybe need to init some data before watch.
// use ListOptions to filter resources.
func (p *DeploymentWatchAction) Init(
	ctx context.Context,
	lister ListFunc[*appv1.DeploymentList],
	opt metav1.ListOptions) error {
	deployments, err := lister(ctx, opt)
	if err != nil {
		return err
	}

	for _, deployment := range deployments.Items {
		p.LogDeploymentStatus("Init", &deployment)
	}
	return nil
}

// OnEvent is called for each event.
func (p *DeploymentWatchAction) OnEvent(event watch.EventType, deployment *appv1.Deployment) {
	currentReplicas := deployment.Status.Replicas
	unavailableReplicas := deployment.Status.UnavailableReplicas
	p.LogDeploymentStatus(event, deployment)
	if unavailableReplicas == 0 && currentReplicas == int32(p.TargetReplicas) {
		p.Logger.Infof("deployment is ready")
		p.End = true
		return
	}
}

// IsEnd is called when watch is end.
func (p *DeploymentWatchAction) IsEnd() bool {
	return p.End
}

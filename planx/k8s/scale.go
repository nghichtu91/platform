package k8s

import (
	"context"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyconfigurationsautoscalingv1 "k8s.io/client-go/applyconfigurations/autoscaling/v1"
	applyscalemetav1 "k8s.io/client-go/applyconfigurations/meta/v1"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	scaleTypeMetaApplyConfiguration applyscalemetav1.TypeMetaApplyConfiguration
)

func init() {
	scaleTypeMetaApplyConfiguration.
		WithKind("scale").
		WithAPIVersion("autoscaling/v1")
}

func int32Val(i *int32) int32 {
	if i != nil {
		return *i
	}
	return 0
}

// IResources is an interface for k8s resources control.
type IResources[T IResource] interface {
	Get(context.Context, string, metav1.GetOptions) (T, error)
	ApplyScale(context.Context, string, *applyconfigurationsautoscalingv1.ScaleApplyConfiguration, metav1.ApplyOptions) (*autoscalingv1.Scale, error)
	GetScale(context.Context, string, metav1.GetOptions) (*autoscalingv1.Scale, error)
	UpdateScale(context.Context, string, *autoscalingv1.Scale, metav1.UpdateOptions) (*autoscalingv1.Scale, error)
}

type IResourceList[T IResource, List any] interface {
	IResources[T]
	IWatchResource[List]
}

// IReplicaResource is an interface for k8s resources that can be scaled.
type IReplicaResource interface {
	SetReplicas(int32)
	GetReplicas() int32
	GetResourceVersion() string
	GetSelector() *metav1.LabelSelector
}

type ScaleControl[RS any] struct {
	Control func(string) RS // generate resources control by namespace
}

type ScaleParam struct {
	ResourceId
	replicas int32 // target replicas
	wait     bool  // wait for the scaling to complete or not
}

func Scale(id ResourceId, replicas int32) ScaleParam {
	return ScaleParam{
		ResourceId: id,
		replicas:   replicas,
	}
}

// Wait sets the wait flag to true
func (p ScaleParam) Wait(wait bool) ScaleParam {
	p.wait = wait
	return p
}

// ScaleResourcePods scale the resource to the specified number of replicas
// and watch resource pods finish scaling.
func ScaleResourcePods[T IResource, RC IResources[T]](
	ctx context.Context, sc ScaleControl[RC], param ScaleParam,
) error {
	if !IsInitialized() {
		return ErrNotInitialized
	}
	ret, err := ApplyScaleResourcesReplicas(ctx, sc, param)
	if err != nil {
		return err
	}
	if !ret.NeedWatch {
		return nil
	}

	if !param.wait {
		ret.Logger.Infof("skip watch resource pods")
		return nil
	}

	action := GenPodWatchAction(ret.WatchParams(int(param.replicas)))
	err = GenWatcher[*v1.Pod, *v1.PodList](
		action, Client().CoreV1().Pods(param.namespace), ret.Logger).
		Watch(ctx, ret.ListOptions())
	if err != nil {
		ret.Logger.Errorf("failed to watch pods %v", err)
		return err
	}
	return nil
}

type GenResourceWatchActionFunc[T IResource, R any] func(WatchParams) IWatchAction[T, R]

// ScaleResources scale the resource to the specified number of replicas.
func ScaleResources[Item IResource, ItemList any, RT IResourceList[Item, ItemList]](
	ctx context.Context, sc ScaleControl[RT],
	gen GenResourceWatchActionFunc[Item, ItemList],
	param ScaleParam,
) error {
	ret, err := ApplyScaleResourcesReplicas(ctx, sc, param)
	if err != nil {
		return err
	}
	if !ret.NeedWatch {
		return nil
	}
	if !param.wait {
		ret.Logger.Infof("skip watch resources")
		return nil
	}

	err = GenWatcher(
		gen(ret.WatchParams(int(param.replicas))),
		sc.Control(param.namespace), ret.Logger).
		Watch(ctx, ret.ListOptions())
	if err != nil {
		ret.Logger.Errorf("failed to watch pods %v", err)
		return err
	}
	return nil
}

type ApplyScaleReplicasRet struct {
	Logger          tilogs.TiLogger
	Revision        string
	LabelSelector   string
	CurrentReplicas int32
	TargetReplicas  int32
	NeedWatch       bool // need to watch the resource finish scaling or not
}

// ListOptions is a struct that contains the list options
func (u *ApplyScaleReplicasRet) ListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector:   u.LabelSelector,
		ResourceVersion: u.Revision,
	}
}

// WatchParams is a struct that contains the watch parameters
func (u *ApplyScaleReplicasRet) WatchParams(replicas int) WatchParams {
	return WatchParams{
		CurrentReplicas: int(u.CurrentReplicas),
		TargetReplicas:  replicas,
		Logger:          u.Logger,
	}
}

func ApplyScaleResourcesReplicas[T IResource, RT IResources[T]](
	ctx context.Context, sc ScaleControl[RT], param ScaleParam,
) (ret ApplyScaleReplicasRet, err error) {

	replicas := param.replicas
	name := param.name
	namespace := param.namespace

	if replicas < 0 {
		err = ErrInvalidReplicas
		return
	}

	if name == "" {
		err = ErrInvalidResourceName
		return
	}

	if namespace == "" {
		err = ErrInvalidNamespace
		return
	}

	logger := tilogs.L().With(
		"resource", name, "namespace", namespace,
	)

	resources := sc.Control(namespace)

	scale, err := resources.GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		logger.Errorf("failed to get scale %v", err)
		return
	}

	revision := scale.GetResourceVersion()
	if scale.Spec.Replicas != replicas {
		scale.Spec.Replicas = replicas
		scale, err = resources.UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	}

	if err != nil {
		logger.Errorf("failed to apply scale %v", err)
		return
	}

	currentReplicas := scale.Status.Replicas
	targetReplicas := scale.Spec.Replicas
	selector := scale.Status.Selector

	logger.Infof(
		"update replicas %d -> %d, selector %s, revision %s -> %s",
		currentReplicas, targetReplicas, selector, revision, scale.GetResourceVersion(),
	)

	ret = ApplyScaleReplicasRet{
		Logger:          logger,
		Revision:        scale.GetResourceVersion(),
		LabelSelector:   selector,
		TargetReplicas:  targetReplicas,
		CurrentReplicas: currentReplicas,
		NeedWatch:       targetReplicas != currentReplicas,
	}
	return
}

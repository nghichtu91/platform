package k8s

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

func ListStatefulSets(ctx context.Context, namespace string) ([]appv1.StatefulSet, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	if namespace == "" {
		return nil, ErrInvalidNamespace
	}
	statefulSets, err := Client().AppsV1().StatefulSets(namespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return statefulSets.Items, nil
}

func GetStatefulSet(ctx context.Context, id ResourceId) (*appv1.StatefulSet, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	if err := id.IsValid(); err != nil {
		return nil, err
	}

	namespace := id.namespace
	name := id.name
	statefulSet, err := Client().AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return statefulSet, nil
}

// GetStatefulSetPods gets the pods of the statefulSet.
func GetStatefulSetPods(ctx context.Context, id ResourceId) ([]corev1.Pod, error) {
	return ListResourcePods(ctx, ListPodsParam[*appv1.StatefulSet]{
		Id:   id,
		Warp: GenWrapStatefulSet,
		Get:  GetStatefulSet,
	})
}

// ScaleStatefulSet scales the statefulSet to the specified number of replicas.
func ScaleStatefulSet(ctx context.Context, param ScaleParam) error {
	return ScaleResourcePods[
		*appv1.StatefulSet, v1.StatefulSetInterface,
	](ctx, scaleStatefulSet, param)
}

// scaleStatefulSet is scale for stateful sets
var scaleStatefulSet = ScaleControl[v1.StatefulSetInterface]{
	Control: func(ns string) v1.StatefulSetInterface { return Client().AppsV1().StatefulSets(ns) },
}

// wrap for scale stateful set

type WrapStatefulSet struct{ *appv1.StatefulSet }

func (w WrapStatefulSet) GetReplicas() int32                 { return int32Val(w.Spec.Replicas) }
func (w WrapStatefulSet) SetReplicas(replicas int32)         { w.Spec.Replicas = &replicas }
func (w WrapStatefulSet) GetResourceVersion() string         { return w.StatefulSet.GetResourceVersion() }
func (w WrapStatefulSet) GetSelector() *metav1.LabelSelector { return w.Spec.Selector }

// GenWrapStatefulSet is a function that generates a WrapStatefulSet object
func GenWrapStatefulSet(statefulSet *appv1.StatefulSet) IReplicaResource {
	return WrapStatefulSet{statefulSet}
}

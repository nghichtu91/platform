package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func ListPods(ctx context.Context, namespace string, opt metav1.ListOptions) ([]v1.Pod, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}
	if namespace == "" {
		return nil, ErrInvalidNamespace
	}
	pods, err := Client().CoreV1().Pods(namespace).
		List(ctx, opt)
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

type ListPodsParam[T IResource] struct {
	Id   ResourceId
	Warp func(T) IReplicaResource
	Get  func(context.Context, ResourceId) (T, error)
}

func ListResourcePods[T IResource](ctx context.Context, param ListPodsParam[T]) ([]v1.Pod, error) {
	resource, err := param.Get(ctx, param.Id)
	if err != nil {
		return nil, err
	}
	warpResource := param.Warp(resource)
	pods, err := ListPods(ctx, param.Id.namespace, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(warpResource.GetSelector()),
	})
	return pods, err
}

func GetPod(ctx context.Context, id ResourceId) (*v1.Pod, error) {
	if !IsInitialized() {
		return nil, ErrNotInitialized
	}

	if err := id.IsValid(); err != nil {
		return nil, err
	}
	name, namespace := id.name, id.namespace

	pod, err := Client().CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// for pods watch event

type IPodWatchAction interface {
	IWatchAction[*v1.Pod, *v1.PodList]
	GetProgress() int
}

type podWatchActionBase struct {
	logger   tilogs.TiLogger
	initPods map[string]*v1.Pod
	target   int // target number of pods
	progress int // current change of pods
}

// GetProgress returns the progress of the action.
func (p *podWatchActionBase) GetProgress() int {
	return p.progress
}

// Init initializes the action. maybe need to init some data before watch.
// use ListOptions to filter resources.
func (p *podWatchActionBase) Init(ctx context.Context, lister ListFunc[*v1.PodList], opt metav1.ListOptions) error {
	p.logger.Infof("list options: %v", opt.String())
	pods, err := lister(ctx, opt)
	if err != nil {
		return err
	}
	p.initPods = make(map[string]*v1.Pod, len(pods.Items))
	for i := range pods.Items {
		pod := &pods.Items[i]
		p.logger.Infof("init pod name: %s phase: %s, resource version: %s",
			pod.Name, string(pod.Status.Phase), pod.GetResourceVersion(),
		)
		p.initPods[pods.Items[i].Name] = &pods.Items[i]
	}
	srcTarget := p.target
	// reset target to the wanted number of pods
	// maybe need to scale in or scale out
	p.target -= len(p.initPods)
	p.logger.Infof("init pods %d, target %d",
		len(p.initPods), srcTarget,
	)

	return nil
}

// OnEvent is called for each event.
func (p *podWatchActionBase) OnEvent(_ watch.EventType, _ *v1.Pod) {}

// IsEnd returns true if the watch should be stopped.
func (p *podWatchActionBase) IsEnd() bool {
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return abs(p.progress) >= abs(p.target)
}

type PodScaleOutAction struct {
	podWatchActionBase
	expansionPods map[string]struct{}
}

type IProgress interface{ GetProgress() int }

// GenPodScaleOutAction generates a PodScaleOutAction.
func GenPodScaleOutAction(target int, logger tilogs.TiLogger) *PodScaleOutAction {
	return &PodScaleOutAction{
		podWatchActionBase: podWatchActionBase{
			target: target,
			logger: logger,
		},
	}
}

// Init initializes the action. maybe need to init some data before watch.
// use ListOptions to filter resources.
func (p *PodScaleOutAction) Init(
	ctx context.Context,
	lister ListFunc[*v1.PodList],
	opt metav1.ListOptions) error {

	err := p.podWatchActionBase.Init(ctx, lister, opt)
	if err != nil {
		return err
	}
	// 扩容的话, target 一定是正数
	if p.target <= 0 {
		return ErrInvalidReplicas
	}
	p.expansionPods = make(map[string]struct{}, p.target)
	// 扩容不关心已有的pod, 但是需要通过 initPods 来计算 target
	p.initPods = nil
	return nil
}

// OnEvent is called for each event.
func (p *PodScaleOutAction) OnEvent(t watch.EventType, pod *v1.Pod) {
	switch t {
	case watch.Added:
		if pod.Status.PodIP != "" {
			return
		}
		if pod.Status.Phase != v1.PodPending {
			return
		}
		_, ok := p.expansionPods[pod.Name]
		if ok {
			return
		}
		p.logger.Infof("expansion pod scheduled name: %s", pod.Name)
		p.expansionPods[pod.Name] = struct{}{}
	case watch.Modified:
		if _, ok := p.expansionPods[pod.Name]; !ok {
			return
		}
		if pod.Status.PodIP == "" {
			return
		}
		if pod.Status.Phase != v1.PodRunning {
			return
		}
		p.logger.Infof(
			"expansion pod running name: %s pod-ip: %s host-ip: %s revision: %s",
			pod.Name, pod.Status.PodIP, pod.Status.HostIP, pod.GetResourceVersion(),
		)
		delete(p.expansionPods, pod.Name)
		p.progress++
	}
}

// PodScaleInAction watches the pod scale in event.
type PodScaleInAction struct {
	podWatchActionBase
}

// GenPodScaleInAction generates a PodScaleInAction.
func GenPodScaleInAction(target int, logger tilogs.TiLogger) *PodScaleInAction {
	return &PodScaleInAction{
		podWatchActionBase: podWatchActionBase{
			target: target,
			logger: logger,
		},
	}
}

// OnEvent is called for each event.
func (p *PodScaleInAction) OnEvent(t watch.EventType, pod *v1.Pod) {
	switch t {
	case watch.Deleted:
		// only care about the pod that is in initPods
		if _, ok := p.initPods[pod.Name]; !ok {
			return
		}
		p.logger.Infof("delete pod name: %s phase: %s",
			pod.Name, string(pod.Status.Phase),
		)
		delete(p.initPods, pod.Name)
		p.progress--
	}
}

func GenPodWatchAction(param WatchParams) IPodWatchAction {
	var action IPodWatchAction
	replicas := param.TargetReplicas
	if param.TargetReplicas > param.CurrentReplicas {
		action = GenPodScaleOutAction(replicas, param.Logger)
	} else {
		action = GenPodScaleInAction(replicas, param.Logger)
	}
	return action
}

var (
	// for type check

	_ IWatchAction[*v1.Pod, *v1.PodList] = &PodScaleOutAction{}
	_ IWatchAction[*v1.Pod, *v1.PodList] = &PodScaleInAction{}
)

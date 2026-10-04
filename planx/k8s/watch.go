package k8s

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type ListFunc[R any] func(context.Context, metav1.ListOptions) (R, error)

// IWatchResource is an interface for k8s watching resources.
type IWatchResource[ItemList any] interface {
	Watch(context.Context, metav1.ListOptions) (watch.Interface, error)
	List(context.Context, metav1.ListOptions) (ItemList, error)
}

// IWatchAction is an interface for k8s resource has watching actions.
type IWatchAction[Item runtime.Object, ItemList any] interface {
	// Init initializes the action. maybe need to init some data before watch.
	// use ListOptions to filter resources.
	Init(context.Context, ListFunc[ItemList], metav1.ListOptions) error
	// OnEvent is called for each event.
	OnEvent(watch.EventType, Item)
	// IsEnd returns true if the watch should be stopped.
	IsEnd() bool
}

type IResource interface {
	runtime.Object

	GetObjectMeta() metav1.Object
}

// Watcher watches the resource and calls the action for each event.
// Item is the resource type,
// ItemList is the list type of the resource.
type Watcher[Item IResource, ItemList any] struct {
	// logger is the logger for the watcher.
	logger tilogs.TiLogger
	// action is the watch action for specific resource.
	action IWatchAction[Item, ItemList]
	// api is the k8s resource api.
	api IWatchResource[ItemList]
}

// Watch watches the resource and calls the action for each event.
// not handle watch reconnect
// opt.ResourceVersion should be set.
// if not set, return ErrWatchInvalidRev.
// if not set to "0", init the action using the list function by opt with exact match.
func (w Watcher[Item, ItemList]) Watch(ctx context.Context, opt metav1.ListOptions) error {
	// ListOptions.ResourceVersion has different meanings for List and Watch.
	// see https://kubernetes.io/zh-cn/docs/reference/using-api/api-concepts/#resource-versions

	if opt.ResourceVersion != "" && opt.ResourceVersion != "0" {
		initOpt := opt.DeepCopy()
		// init must use exact match for getting the resource.
		// so resource-version must be set.
		initOpt.ResourceVersionMatch = metav1.ResourceVersionMatchExact
		if err := w.action.Init(ctx, w.api.List, *initOpt); err != nil {
			return errors.Join(ErrWatchInitFailed, err)
		}
	}
	w.logger.Infof("watch init success, start receiving events resource-version: %s",
		opt.ResourceVersion,
	)
	// for watch, resource-version is "" or "0" is valid.
	watcher, err := w.api.Watch(ctx, opt)
	if err != nil {
		return errors.Join(ErrWatchFailed, err)
	}
	defer watcher.Stop()
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				// watch channel closed, maybe need to reconnect?
				w.logger.Infof("watch channel closed")
				return ErrWatchInterrupt
			}
			var obj Item
			obj, ok = event.Object.(Item)
			if !ok {
				continue
			}
			objMeta := obj.GetObjectMeta()
			w.logger.Debugf("watch event type: %s, name: %s, resource-version: %s",
				event.Type, objMeta.GetName(), objMeta.GetResourceVersion(),
			)
			w.action.OnEvent(event.Type, obj)
			if w.action.IsEnd() {
				// stop successfully
				w.logger.Infof("watch end")
				return nil
			}
		case <-ctx.Done():
			// stop by context
			w.logger.Infof("watch stopped by context")
			return ctx.Err()
		}
	}
}

func GenWatcher[Item IResource, ItemList any](
	action IWatchAction[Item, ItemList], api IWatchResource[ItemList],
	logger tilogs.TiLogger,
) Watcher[Item, ItemList] {
	return Watcher[Item, ItemList]{
		action: action,
		api:    api,
		logger: logger,
	}
}

type WatchParams struct {
	CurrentReplicas int
	TargetReplicas  int
	Logger          tilogs.TiLogger
}

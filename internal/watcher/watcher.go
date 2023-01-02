package watcher

import (
	"context"
	"time"

	"github.com/ahsayde/analytics-controller/api/v1alpha1"
	elasticSink "github.com/ahsayde/analytics-controller/internal/sinks/elastic"
	fileSink "github.com/ahsayde/analytics-controller/internal/sinks/file"
	sqliteSink "github.com/ahsayde/analytics-controller/internal/sinks/sqlite"
	webhookSink "github.com/ahsayde/analytics-controller/internal/sinks/webhook"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Watcher struct {
	mgr   ctrl.Manager
	sinks map[string]Sink
}

func New(mgr ctrl.Manager) *Watcher {
	return &Watcher{
		mgr:   mgr,
		sinks: make(map[string]Sink),
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	informer, err := w.mgr.GetCache().GetInformer(ctx, &v1.Event{})
	if err != nil {
		return err
	}

	log.Log.Info("watting for sinks to be registered ...")

	for len(w.sinks) == 0 {
		time.Sleep(1 * time.Second)
	}

	log.Log.Info("starting events listener ...")

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			w.handler(ctx, obj)
		},
	})

	<-ctx.Done()

	for name, sink := range w.sinks {
		if err := sink.Stop(); err != nil {
			log.Log.Error(err, "failed to stop sink", "sink", name)
		}
	}

	return nil
}

func (w *Watcher) handler(ctx context.Context, obj interface{}) {
	event, ok := obj.(*v1.Event)
	if !ok {
		return
	}

	var eventSets v1alpha1.EventSetList
	if err := w.mgr.GetCache().List(ctx, &eventSets); err != nil {
		log.Log.Error(err, "failed to list event sets")
		return
	}

	for _, eventSet := range eventSets.Items {
		if eventSet.Spec.Match.Match(event) {
			for _, ref := range eventSet.Spec.SinkRefs {
				sink, ok := w.sinks[ref.Name]
				if !ok {
					log.Log.Error(nil, "sink not found", "sink", ref.Name)
					continue
				}
				if err := sink.Write(ctx, *event); err != nil {
					log.Log.Error(err, "failed to write event to sink", "sink", ref.Name)
				}
			}
		}
	}
}

func (w *Watcher) RegisterSink(ctx context.Context, cr v1alpha1.Sink, secretConf map[string]string) error {
	var err error
	var sink Sink

	if cr.Spec.File != nil {
		sink, err = fileSink.New(cr.Spec.File.Path)
		if err != nil {
			return err
		}
	} else if cr.Spec.SQLite != nil {
		sink, err = sqliteSink.New(cr.Spec.SQLite.Path)
		if err != nil {
			return err
		}
	} else if cr.Spec.Webhook != nil {
		sink, err = webhookSink.New(cr.Spec.Webhook.Endpoint, cr.Spec.Webhook.Headers)
		if err != nil {
			return err
		}
	} else if cr.Spec.Elastic != nil {
		cr.Spec.Elastic.SetSecretConf(secretConf)
		sink, err = elasticSink.New(
			cr.Spec.Elastic.Address,
			cr.Spec.Elastic.IndexName,
			cr.Spec.Elastic.Username,
			cr.Spec.Elastic.Password,
			cr.Spec.Elastic.BatchSize,
			cr.Spec.Elastic.BatchExpiry,
		)
		if err != nil {
			return err
		}
	}

	if err := sink.Start(ctx); err != nil {
		return err
	}

	w.sinks[cr.Name] = sink

	return nil
}

func (w *Watcher) RemoveSink(name string) {
	if sink, ok := w.sinks[name]; ok {
		delete(w.sinks, name)
		if err := sink.Stop(); err != nil {
			log.Log.Error(err, "failed to stop sink")
		}
	}
}

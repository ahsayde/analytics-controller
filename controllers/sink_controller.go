/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ahsayde/analytics-controller/internal/watcher"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/ahsayde/analytics-controller/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	AvailableReason     = "Available"
	FailedToStartReason = "FailedToStart"
	InvalidSecretReason = "InvalidSecret"

	secretRefIndexKey = "spec.secretRef"
)

// SinkReconciler reconciles a Sink object
type SinkReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Watcher *watcher.Watcher
}

//+kubebuilder:rbac:groups=analytics.weave.works,resources=sinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=analytics.weave.works,resources=sinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=analytics.weave.works,resources=sinks/finalizers,verbs=update

func (r *SinkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var sink v1alpha1.Sink
	if err := r.Get(ctx, req.NamespacedName, &sink); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to get sink")
		return ctrl.Result{}, err
	}

	if !sink.ObjectMeta.DeletionTimestamp.IsZero() {
		r.Watcher.RemoveSink(req.Name)
		return ctrl.Result{}, nil
	}

	patch := client.MergeFrom(sink.DeepCopy())

	sink.Status.ObservedGeneration = sink.Generation

	var secretConf map[string]string
	var err error

	if sink.Spec.SecretRef != nil {
		secretConf, err = r.getConfigFromSecret(ctx, sink.Spec.SecretRef)
		if err != nil {
			sink.MarkAsNotReady(err.Error(), InvalidSecretReason)
			if err := r.updateStatus(ctx, sink, patch); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	if err := r.Watcher.RegisterSink(ctx, sink, secretConf); err != nil {
		sink.MarkAsNotReady(err.Error(), FailedToStartReason)
	} else {
		sink.MarkAsReady("Sink is ready.", AvailableReason)
	}

	if err := r.updateStatus(ctx, sink, patch); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SinkReconciler) getConfigFromSecret(ctx context.Context, secretRef *v1.SecretReference) (map[string]string, error) {
	var secret v1.Secret
	key := client.ObjectKey{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}

	if err := r.Get(ctx, key, &secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, err
	}

	data := make(map[string]string)
	for key, value := range secret.Data {
		data[strings.ToLower(key)] = string(value)
	}

	return data, nil
}

func (r *SinkReconciler) updateStatus(ctx context.Context, sink v1alpha1.Sink, patch client.Patch) error {
	if err := r.Status().Patch(ctx, &sink, patch); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func secretRefIndexHandler(obj client.Object) []string {
	sink, ok := obj.(*v1alpha1.Sink)
	if !ok {
		return nil
	}
	if sink.Spec.SecretRef == nil {
		return nil
	}

	return []string{
		fmt.Sprintf(
			"%s/%s",
			sink.Spec.SecretRef.Namespace,
			sink.Spec.SecretRef.Name,
		),
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&v1alpha1.Sink{},
		secretRefIndexKey,
		secretRefIndexHandler,
	)

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Sink{}).
		Watches(
			&source.Kind{Type: &v1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.secretWatcher),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *SinkReconciler) secretWatcher(obj client.Object) []reconcile.Request {
	key := fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())
	opts := client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(secretRefIndexKey, key),
	}

	var list v1alpha1.SinkList
	ctx := context.Background()
	if err := r.List(ctx, &list, &opts); err != nil {
		log.Log.Error(err, "")
		return nil
	}

	var requests []reconcile.Request
	for _, item := range list.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: item.Name,
			},
		})
	}

	return requests
}

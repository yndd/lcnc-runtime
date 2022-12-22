/*
Copyright 2021 NDD.

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

package eventhandler

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/yndd/lcnc-runtime/pkg/dag"
	"github.com/yndd/lcnc-runtime/pkg/executor"
	"github.com/yndd/lcnc-runtime/pkg/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type EventHandlerInfo struct {
	Client client.Client
	GVK    *schema.GroupVersionKind
	DAG    dag.DAG
}

func New(e *EventHandlerInfo) handler.EventHandler {
	opts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	return &eventhandler{
		//ctx:    ctx,
		client: e.Client,
		gvk:    e.GVK,
		d:      e.DAG,
		l:      ctrl.Log.WithName("lcnc eventhandler"),
	}
}

type eventhandler struct {
	client client.Client
	//ctx    context.Context
	gvk *schema.GroupVersionKind
	d   dag.DAG

	l logr.Logger
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (r *eventhandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	r.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (r *eventhandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	r.add(evt.ObjectOld, q)
	r.add(evt.ObjectNew, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (r *eventhandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	r.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (r *eventhandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	r.add(evt.Object, q)
}

func (r *eventhandler) add(obj runtime.Object, queue adder) {
	r.l.Info("watch event started...")

	o, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	x, err := meta.MarshalData(o)
	if err != nil {
		r.l.Error(err, "cannot marshal data")
		return
	}

	namespace := o.GetNamespace()
	if o.GetNamespace() == "" {
		namespace = "default"
	}

	e := executor.New(&executor.Config{
		Name:       o.GetName(),
		Namespace:  namespace,
		RootVertex: r.d.GetRootVertex(),
		Data:       x,
		Client:     r.client,
		GVK:        r.gvk,
		DAG:        r.d,
	})

	e.Run(context.TODO())
	e.GetOutput()
	e.GetResult()

	// for all the output add the queues

	/*
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: toponode.GetNamespace(),
			Name:      toponode.GetName()}},
		)
	*/

	r.l.Info("watch event finsihed...")
}

type adder interface {
	Add(item interface{})
}

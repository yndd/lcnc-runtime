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

	"github.com/yndd/ndd-runtime/pkg/logging"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

type EventHandlerInfo struct {
	Client client.Client
	Log    logging.Logger
	Gvk    schema.GroupVersionKind
	Fn     string // to be updated
}

func New(ctx context.Context, e *EventHandlerInfo) handler.EventHandler {
	return &eventhandler{
		ctx:    ctx,
		client: e.Client,
		gvk:    e.Gvk,
		fn:     e.Fn,
		log:    e.Log,
	}
}

type eventhandler struct {
	client client.Client
	log    logging.Logger
	ctx    context.Context
	gvk    schema.GroupVersionKind
	fn     string // to be updated
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *eventhandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *eventhandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.ObjectOld, q)
	e.add(evt.ObjectNew, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *eventhandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

// Create enqueues a request for all infrastructures which pertains to the topology.
func (e *eventhandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.add(evt.Object, q)
}

func (e *eventhandler) add(obj runtime.Object, queue adder) {
	o, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	log := e.log.WithValues("function", "watch topologies", "name", o.GetName())
	log.Debug("topologynode handleEvent")

	/*
		queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: toponode.GetNamespace(),
			Name:      toponode.GetName()}},
		)
	*/
}

type adder interface {
	Add(item interface{})
}

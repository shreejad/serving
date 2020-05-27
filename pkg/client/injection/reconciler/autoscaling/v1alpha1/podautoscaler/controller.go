/*
Copyright 2020 The Knative Authors

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

// Code generated by injection-gen. DO NOT EDIT.

package podautoscaler

import (
	context "context"
	fmt "fmt"
	reflect "reflect"
	strings "strings"

	corev1 "k8s.io/api/core/v1"
	watch "k8s.io/apimachinery/pkg/watch"
	scheme "k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	record "k8s.io/client-go/tools/record"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	controller "knative.dev/pkg/controller"
	logging "knative.dev/pkg/logging"
	versionedscheme "knative.dev/serving/pkg/client/clientset/versioned/scheme"
	client "knative.dev/serving/pkg/client/injection/client"
	podautoscaler "knative.dev/serving/pkg/client/injection/informers/autoscaling/v1alpha1/podautoscaler"
)

const (
	defaultControllerAgentName = "podautoscaler-controller"
	defaultFinalizerName       = "podautoscalers.autoscaling.internal.knative.dev"

	// ClassAnnotationKey points to the annotation for the class of this resource.
	ClassAnnotationKey = "autoscaling.knative.dev/class"
)

// NewImpl returns a controller.Impl that handles queuing and feeding work from
// the queue through an implementation of controller.Reconciler, delegating to
// the provided Interface and optional Finalizer methods. OptionsFn is used to return
// controller.Options to be used but the internal reconciler.
func NewImpl(ctx context.Context, r Interface, classValue string, optionsFns ...controller.OptionsFn) *controller.Impl {
	logger := logging.FromContext(ctx)

	// Check the options function input. It should be 0 or 1.
	if len(optionsFns) > 1 {
		logger.Fatalf("up to one options function is supported, found %d", len(optionsFns))
	}

	podautoscalerInformer := podautoscaler.Get(ctx)

	rec := &reconcilerImpl{
		Client:        client.Get(ctx),
		Lister:        podautoscalerInformer.Lister(),
		reconciler:    r,
		finalizerName: defaultFinalizerName,
		classValue:    classValue,
	}

	t := reflect.TypeOf(r).Elem()
	queueName := fmt.Sprintf("%s.%s", strings.ReplaceAll(t.PkgPath(), "/", "-"), t.Name())

	impl := controller.NewImpl(rec, logger, queueName)
	agentName := defaultControllerAgentName

	// Pass impl to the options. Save any optional results.
	for _, fn := range optionsFns {
		opts := fn(impl)
		if opts.ConfigStore != nil {
			rec.configStore = opts.ConfigStore
		}
		if opts.FinalizerName != "" {
			rec.finalizerName = opts.FinalizerName
		}
		if opts.AgentName != "" {
			agentName = opts.AgentName
		}
	}

	rec.Recorder = createRecorder(ctx, agentName)

	return impl
}

func createRecorder(ctx context.Context, agentName string) record.EventRecorder {
	logger := logging.FromContext(ctx)

	recorder := controller.GetEventRecorder(ctx)
	if recorder == nil {
		// Create event broadcaster
		logger.Debug("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		watches := []watch.Interface{
			eventBroadcaster.StartLogging(logger.Named("event-broadcaster").Infof),
			eventBroadcaster.StartRecordingToSink(
				&v1.EventSinkImpl{Interface: kubeclient.Get(ctx).CoreV1().Events("")}),
		}
		recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: agentName})
		go func() {
			<-ctx.Done()
			for _, w := range watches {
				w.Stop()
			}
		}()
	}

	return recorder
}

func init() {
	versionedscheme.AddToScheme(scheme.Scheme)
}

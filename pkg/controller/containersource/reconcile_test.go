/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Veroute.on 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package containersource

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	//"fmt"
	"testing"

	sourcesv1alpha1 "github.com/knative/eventing-sources/pkg/apis/sources/v1alpha1"
	controllertesting "github.com/knative/eventing-sources/pkg/controller/testing"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	//"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	trueVal   = true
	targetURI = "http://target.example.com"
)

const (
	image             = "github.com/knative/test/image"
	fromChannelName   = "fromchannel"
	resultChannelName = "resultchannel"
	sourceName        = "source"
	routeName         = "callroute"
	channelKind       = "Channel"
	routeKind         = "Route"
	sourceKind        = "Source"
	targetDNS         = "myfunction.mynamespace.svc.cluster.local"

	eventType           = "myeventtype"
	containerSourceName = "testcontainersource"
	testNS              = "testnamespace"
	k8sServiceName      = "testk8sservice"

	sinkableDNS = "sinkable.sink.svc.cluster.local"

	sinkableName       = "testsink"
	sinkableKind       = "Sink"
	sinkableAPIVersion = "duck.knative.dev/v1alpha1"

	unsinkableName       = "testunsinkable"
	unsinkableKind       = "KResource"
	unsinkableAPIVersion = "duck.knative.dev/v1alpha1"

	sinkServiceName       = "testsinkservice"
	sinkServiceKind       = "Service"
	sinkServiceAPIVersion = "v1"
)

// Adds the list of known types to Scheme.
func duckAddKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		duckv1alpha1.SchemeGroupVersion,
		&duckv1alpha1.SinkList{},
	)
	metav1.AddToGroupVersion(scheme, duckv1alpha1.SchemeGroupVersion)
	return nil
}

func init() {
	// Add types to scheme
	sourcesv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
	duckv1alpha1.AddToScheme(scheme.Scheme)
	duckAddKnownTypes(scheme.Scheme)
}

var testCases = []controllertesting.TestCase{
	{
		Name:         "non existent key",
		Reconciles:   &sourcesv1alpha1.ContainerSource{},
		ReconcileKey: "non-existent-test-ns/non-existent-test-key",
		WantErr:      false,
	}, {
		Name:       "valid containersource, but sink does not exist",
		Reconciles: &sourcesv1alpha1.ContainerSource{},
		InitialState: []runtime.Object{
			getContainerSource(),
		},
		ReconcileKey: fmt.Sprintf("%s/%s", testNS, containerSourceName),
		WantErrMsg:   `sinks.duck.knative.dev "testsink" not found`,
	}, {
		Name:       "valid containersource, but sink is not sinkable",
		Reconciles: &sourcesv1alpha1.ContainerSource{},
		InitialState: []runtime.Object{
			getContainerSource_unsinkable(),
		},
		ReconcileKey: fmt.Sprintf("%s/%s", testNS, containerSourceName),
		Scheme:       scheme.Scheme,
		Objects: []runtime.Object{
			// An unsinkable resource
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": unsinkableAPIVersion,
					"kind":       unsinkableKind,
					"metadata": map[string]interface{}{
						"namespace": testNS,
						"name":      unsinkableName,
					},
				},
			},
		},
		WantErrMsg: "sink does not contain sinkable",
	}, {
		Name:       "valid containersource, sink is sinkable",
		Reconciles: &sourcesv1alpha1.ContainerSource{},
		InitialState: []runtime.Object{
			getContainerSource(),
		},
		ReconcileKey: fmt.Sprintf("%s/%s", testNS, containerSourceName),
		Scheme:       scheme.Scheme,
		Objects: []runtime.Object{
			// k8s Service
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": sinkableAPIVersion,
					"kind":       sinkableKind,
					"metadata": map[string]interface{}{
						"namespace": testNS,
						"name":      sinkableName,
					},
					"status": map[string]interface{}{
						"sinkable": map[string]interface{}{
							"domainInternal": sinkableDNS,
						},
					},
				},
			},
		},
	}, /* TODO: support k8s service {
		Name:       "valid containersource, sink is a k8s service",
		Reconciles: &sourcesv1alpha1.ContainerSource{},
		InitialState: []runtime.Object{
			getContainerSource_sinkService(),
		},
		ReconcileKey: fmt.Sprintf("%s/%s", testNS, containerSourceName),
		Scheme:       scheme.Scheme,
		Objects: []runtime.Object{
			// sinkable
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": sinkServiceAPIVersion,
					"kind":       sinkServiceKind,
					"metadata": map[string]interface{}{
						"namespace": testNS,
						"name":      sinkServiceName,
					},
				}},
		},
	},*/
}

func TestAllCases(t *testing.T) {
	recorder := record.NewBroadcaster().NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	for _, tc := range testCases {
		c := tc.GetClient()
		dc := tc.GetDynamicClient()

		r := &reconciler{
			client:        c,
			dynamicClient: dc,
			scheme:        tc.Scheme,
			restConfig:    &rest.Config{},
			recorder:      recorder,
		}
		t.Run(tc.Name, tc.RunnerSDK(t, r, c))
	}
}

//
//func getNewFromChannel() *eventingv1alpha1.Channel {
//	return getNewChannel(fromChannelName)
//}
//
//func getNewResultChannel() *eventingv1alpha1.Channel {
//	return getNewChannel(resultChannelName)
//}
//
//func getNewChannel(name string) *eventingv1alpha1.Channel {
//	channel := &eventingv1alpha1.Channel{
//		TypeMeta:   channelType(),
//		ObjectMeta: om("test", name),
//		Spec:       eventingv1alpha1.ChannelSpec{},
//	}
//	channel.ObjectMeta.OwnerReferences = append(channel.ObjectMeta.OwnerReferences, getOwnerReference(false))
//
//	// selflink is not filled in when we create the object, so clear it
//	channel.ObjectMeta.SelfLink = ""
//	return channel
//}
//
func getContainerSource() *sourcesv1alpha1.ContainerSource {
	obj := &sourcesv1alpha1.ContainerSource{
		TypeMeta:   containerSourceType(),
		ObjectMeta: om(testNS, containerSourceName),
		Spec: sourcesv1alpha1.ContainerSourceSpec{
			Image: image,
			Args:  []string{},
			Sink: &corev1.ObjectReference{
				Name:       sinkableName,
				Kind:       sinkableKind,
				APIVersion: sinkableAPIVersion,
			},
		},
	}
	// selflink is not filled in when we create the object, so clear it
	obj.ObjectMeta.SelfLink = ""
	return obj
}

func getContainerSource_sinkService() *sourcesv1alpha1.ContainerSource {
	obj := &sourcesv1alpha1.ContainerSource{
		TypeMeta:   containerSourceType(),
		ObjectMeta: om(testNS, containerSourceName),
		Spec: sourcesv1alpha1.ContainerSourceSpec{
			Image: image,
			Args:  []string{},
			Sink: &corev1.ObjectReference{
				Name:       sinkServiceName,
				Kind:       sinkServiceKind,
				APIVersion: sinkServiceAPIVersion,
			},
		},
	}
	// selflink is not filled in when we create the object, so clear it
	obj.ObjectMeta.SelfLink = ""
	return obj
}

func getContainerSource_unsinkable() *sourcesv1alpha1.ContainerSource {
	obj := &sourcesv1alpha1.ContainerSource{
		TypeMeta:   containerSourceType(),
		ObjectMeta: om(testNS, containerSourceName),
		Spec: sourcesv1alpha1.ContainerSourceSpec{
			Image: image,
			Args:  []string{},
			Sink: &corev1.ObjectReference{
				Name:       unsinkableName,
				Kind:       unsinkableKind,
				APIVersion: unsinkableAPIVersion,
			},
		},
	}
	// selflink is not filled in when we create the object, so clear it
	obj.ObjectMeta.SelfLink = ""
	return obj
}

//
//func getNewSubscriptionToK8sService() *eventingv1alpha1.Subscription {
//	sub := getNewSubscription()
//	sub.Spec.Call = &eventingv1alpha1.Callable{
//		Target: &corev1.ObjectReference{
//			Name:       k8sServiceName,
//			Kind:       "Service",
//			APIVersion: "v1",
//		},
//	}
//	return sub
//}
//
//func getNewSubscriptionWithSource() *eventingv1alpha1.Subscription {
//	subscription := &eventingv1alpha1.Subscription{
//		TypeMeta:   subscriptionType(),
//		ObjectMeta: om(testNS, subscriptionName),
//		Spec: eventingv1alpha1.SubscriptionSpec{
//			From: corev1.ObjectReference{
//				Name:       sourceName,
//				Kind:       sourceKind,
//				APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
//			},
//			Call: &eventingv1alpha1.Callable{
//				Target: &corev1.ObjectReference{
//					Name:       routeName,
//					Kind:       routeKind,
//					APIVersion: "serving.knative.dev/v1alpha1",
//				},
//			},
//			Result: &eventingv1alpha1.ResultStrategy{
//				Target: &corev1.ObjectReference{
//					Name:       resultChannelName,
//					Kind:       channelKind,
//					APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
//				},
//			},
//		},
//	}
//	subscription.ObjectMeta.OwnerReferences = append(subscription.ObjectMeta.OwnerReferences, getOwnerReference(false))
//
//	// selflink is not filled in when we create the object, so clear it
//	subscription.ObjectMeta.SelfLink = ""
//	return subscription
//}
//
//func getNewSubscriptionWithUnknownConditions() *eventingv1alpha1.Subscription {
//	s := getNewSubscription()
//	s.Status.InitializeConditions()
//	return s
//}
//
//func getNewSubscriptionWithReferencesResolvedStatus() *eventingv1alpha1.Subscription {
//	s := getNewSubscriptionWithUnknownConditions()
//	s.Status.MarkReferencesResolved()
//	return s
//}
//
//func channelType() metav1.TypeMeta {
//	return metav1.TypeMeta{
//		APIVersion: eventingv1alpha1.SchemeGroupVersion.String(),
//		Kind:       "Channel",
//	}
//}
//
func containerSourceType() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: sourcesv1alpha1.SchemeGroupVersion.String(),
		Kind:       "ContainerSource",
	}
}

//
//func getK8sService() *corev1.Service {
//	return &corev1.Service{
//		TypeMeta: metav1.TypeMeta{
//			APIVersion: "v1",
//			Kind:       "Service",
//		},
//		ObjectMeta: metav1.ObjectMeta{
//			Namespace: testNS,
//			Name:      k8sServiceName,
//		},
//	}
//}

func om(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		SelfLink:  fmt.Sprintf("/apis/eventing/sources/v1alpha1/namespaces/%s/object/%s", namespace, name),
	}
}

//func feedObjectMeta(namespace, generateName string) metav1.ObjectMeta {
//	return metav1.ObjectMeta{
//		Namespace:    namespace,
//		GenerateName: generateName,
//		OwnerReferences: []metav1.OwnerReference{
//			getOwnerReference(true),
//		},
//	}
//}
//
//func getOwnerReference(blockOwnerDeletion bool) metav1.OwnerReference {
//	return metav1.OwnerReference{
//		APIVersion:         eventingv1alpha1.SchemeGroupVersion.String(),
//		Kind:               "Subscription",
//		Name:               subscriptionName,
//		Controller:         &trueVal,
//		BlockOwnerDeletion: &blockOwnerDeletion,
//	}
//}

func getTestResources() []runtime.Object {
	return []runtime.Object{
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": unsinkableAPIVersion,
				"kind":       unsinkableKind,
				"metadata": map[string]interface{}{
					"namespace": testNS,
					"name":      unsinkableName,
				},
			},
		}, &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": sinkableAPIVersion,
				"kind":       sinkableKind,
				"metadata": map[string]interface{}{
					"namespace": testNS,
					"name":      sinkableName,
				},
			},
		},
	}
}

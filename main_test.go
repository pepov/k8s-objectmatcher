/*
Copyright 2019 Banzai Cloud.

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
package objectmatch

import (
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/goph/emperror"
	"github.com/pkg/errors"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
)

var (
	integration   = flag.Bool("integration", false, "run integration tests")
	kubeconfig    = flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube/config"), "kubernetes config to use for tests")
	kubecontext   = flag.String("kubecontext", "", "kubernetes context to use in tests")
	keepnamespace = flag.Bool("keepnamespace", false, "keep the kubernetes namespace that was used for the tests")
	testContext   = &IntegrationTestContext{}
)

func TestMain(m *testing.M) {

	flag.Parse()

	if testing.Verbose() {
		klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
		klog.InitFlags(klogFlags)
		klogFlags.Set("v", "3")
	}

	if *integration {
		err := testContext.Setup()
		if err != nil {
			panic("Failed to setup test namespace")
		}
	}
	result := m.Run()
	if *integration {
		if !*keepnamespace {
			err := testContext.DeleteNamespace()
			if err != nil {
				panic("Failed to delete test namespace")
			}
		}
	}
	os.Exit(result)
}

type IntegrationTestContext struct {
	Client           kubernetes.Interface
	ExtensionsClient apiextension.Interface
	Namespace        string
}

func (ctx *IntegrationTestContext) CreateNamespace() error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "integration-",
		},
	}
	namespace, err := ctx.Client.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return err
	}
	ctx.Namespace = namespace.Name
	return nil
}

func (ctx *IntegrationTestContext) Setup() error {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: *kubecontext},
	).ClientConfig()
	if err != nil {
		return err
	}
	ctx.Client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return emperror.Wrap(err, "Failed to create kubernetes client")
	}
	ctx.ExtensionsClient, err = apiextension.NewForConfig(config)
	if err != nil {
		return emperror.Wrap(err, "Failed to create apiextensions client")
	}
	err = testContext.CreateNamespace()
	if err != nil {
		return emperror.Wrap(err, "Failed to create test namespace")
	}
	return err
}

func (ctx *IntegrationTestContext) DeleteNamespace() error {
	err := ctx.Client.CoreV1().Namespaces().Delete(ctx.Namespace, &metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	})
	return err
}

type TestItem struct {
	name         string
	object       metav1.Object
	shouldMatch  bool
	remoteChange func(interface{})
	localChange  func(interface{})
}

func NewTestMatch(name string, object metav1.Object) *TestItem {
	return &TestItem{
		name:        name,
		object:      object,
		shouldMatch: true,
	}
}
func NewTestDiff(name string, object metav1.Object) *TestItem {
	return &TestItem{
		name:        name,
		object:      object,
		shouldMatch: false,
	}
}

func (t *TestItem) withRemoteChange(extender func(interface{})) *TestItem {
	t.remoteChange = extender
	return t
}

func (t *TestItem) withLocalChange(extender func(interface{})) *TestItem {
	t.localChange = extender
	return t
}

func testMatchOnObjectv2(testItem *TestItem) error {
	newObject := testItem.object
	var existing metav1.Object
	var err error
	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: new(int64),
	}
	switch newObject.(type) {
	default:
		return emperror.With(errors.New("Unsupported type"), "type", reflect.TypeOf(newObject), "object", newObject)
	case *rbacv1.ClusterRole:
		existing, err = testContext.Client.RbacV1().ClusterRoles().Create(newObject.(*rbacv1.ClusterRole))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.RbacV1().ClusterRoles().Delete(existing.GetName(), deleteOptions)
		}()
	case *rbacv1.ClusterRoleBinding:
		existing, err = testContext.Client.RbacV1().ClusterRoleBindings().Create(newObject.(*rbacv1.ClusterRoleBinding))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.RbacV1().ClusterRoleBindings().Delete(existing.GetName(), deleteOptions)
		}()
	case *v1.Pod:
		existing, err = testContext.Client.CoreV1().Pods(newObject.GetNamespace()).Create(newObject.(*v1.Pod))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.CoreV1().Pods(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *v1.Service:
		existing, err = testContext.Client.CoreV1().Services(newObject.GetNamespace()).Create(newObject.(*v1.Service))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.CoreV1().Services(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *v1.ConfigMap:
		existing, err = testContext.Client.CoreV1().ConfigMaps(newObject.GetNamespace()).Create(newObject.(*v1.ConfigMap))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.CoreV1().ConfigMaps(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *v1beta1.CustomResourceDefinition:
		existing, err = testContext.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(newObject.(*v1beta1.CustomResourceDefinition))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.ExtensionsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(existing.GetName(), deleteOptions)
		}()
	case *appsv1.DaemonSet:
		existing, err = testContext.Client.AppsV1().DaemonSets(newObject.GetNamespace()).Create(newObject.(*appsv1.DaemonSet))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.AppsV1().DaemonSets(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *appsv1.Deployment:
		existing, err = testContext.Client.AppsV1().Deployments(newObject.GetNamespace()).Create(newObject.(*appsv1.Deployment))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.AppsV1().Deployments(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *v2beta1.HorizontalPodAutoscaler:
		existing, err = testContext.Client.AutoscalingV2beta1().HorizontalPodAutoscalers(newObject.GetNamespace()).Create(newObject.(*v2beta1.HorizontalPodAutoscaler))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.AutoscalingV2beta1().HorizontalPodAutoscalers(newObject.GetNamespace()).Delete(existing.GetName(), deleteOptions)
		}()
	case *admissionregistrationv1beta1.MutatingWebhookConfiguration:
		existing, err = testContext.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(newObject.(*admissionregistrationv1beta1.MutatingWebhookConfiguration))
		if err != nil {
			return emperror.WrapWith(err, "failed to create object", "object", newObject)
		}
		defer func() {
			testContext.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(existing.GetName(), deleteOptions)
		}()
	}

	if testItem.remoteChange != nil {
		testItem.remoteChange(existing)
	}

	if testItem.localChange != nil {
		testItem.localChange(newObject)
	}

	matched, err := New(klogr.New()).Match(existing, newObject)
	if err != nil {
		return err
	}

	if testItem.shouldMatch && !matched {
		return emperror.With(errors.New("Objects did not match"))
	}

	if !testItem.shouldMatch && matched {
		return emperror.With(errors.New("Objects matched although they should not"))
	}

	return nil
}

func standardObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		GenerateName: "test-",
		Namespace:    testContext.Namespace,
	}
}

func metaWithLabels(labels map[string]string) metav1.ObjectMeta {
	meta := standardObjectMeta()
	meta.Labels = labels
	return meta
}

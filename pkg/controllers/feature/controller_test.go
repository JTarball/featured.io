package feature

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	featurecontroller "github.com/featured.io/pkg/apis/feature/v1alpha1"
	"github.com/featured.io/pkg/generated/clientset/versioned/fake"
	informers "github.com/featured.io/pkg/generated/informers/externalversions"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client     *fake.Clientset
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	featureflagLister []*featurecontroller.FeatureFlag
	configmapLister   []*core.ConfigMap
	// Actions expected to happen on the client.
	kubeactions []kubetesting.Action
	actions     []kubetesting.Action
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}
	return f
}

func newFeatureFlag(name string, replicas *int32) *featurecontroller.FeatureFlag {
	return &featurecontroller.FeatureFlag{
		TypeMeta: metav1.TypeMeta{APIVersion: featurecontroller.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: featurecontroller.FeatureFlagSpec{
			ConfigMapName: fmt.Sprintf("%s-config", name),
			Replicas:      replicas,
		},
	}
}

func (f *fixture) newFeatureController() (*FeatureController, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
	f.client = fake.NewSimpleClientset(f.objects...)
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	i := informers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	c := NewFeatureController(f.kubeclient, f.client,
		k8sI.Core().V1().ConfigMaps(), i.Featurecontroller().V1alpha1().FeatureFlags())

	c.featureflagsSynced = alwaysReady
	c.configmapsSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}

	for _, f := range f.featureflagLister {
		i.Featurecontroller().V1alpha1().FeatureFlags().Informer().GetIndexer().Add(f)
	}

	for _, d := range f.configmapLister {
		fmt.Println("---- Adding configmap to lister ----")
		k8sI.Core().V1().ConfigMaps().Informer().GetIndexer().Add(d)
	}

	return c, i, k8sI
}

func (f *fixture) run(featureflagName string) {
	f.runFeatureController(featureflagName, true, false)
}

func (f *fixture) runExpectError(featureflagName string) {
	f.runFeatureController(featureflagName, true, true)
}

func (f *fixture) runFeatureController(featureflagName string, startInformers bool, expectError bool) {
	c, i, k8sI := f.newFeatureController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		i.Start(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(featureflagName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing featureflag: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing featureflag, got nil")
	}

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual kubetesting.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case kubetesting.CreateActionImpl:
		e, _ := expected.(kubetesting.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case kubetesting.UpdateActionImpl:
		e, _ := expected.(kubetesting.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case kubetesting.PatchActionImpl:
		e, _ := expected.(kubetesting.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []kubetesting.Action) []kubetesting.Action {
	ret := []kubetesting.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "featureflags") ||
				action.Matches("watch", "featureflags") ||
				action.Matches("list", "configmaps") ||
				action.Matches("watch", "configmaps")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectCreateConfigMapAction(d *core.ConfigMap) {
	f.kubeactions = append(f.kubeactions, kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, d.Namespace, d))
}

func (f *fixture) expectUpdateConfigMapAction(d *core.ConfigMap) {
	f.kubeactions = append(f.kubeactions, kubetesting.NewUpdateAction(schema.GroupVersionResource{Resource: "configmaps"}, d.Namespace, d))
}

func (f *fixture) expectUpdateFooStatusAction(featureflag *featurecontroller.FeatureFlag) {
	action := kubetesting.NewUpdateAction(schema.GroupVersionResource{Resource: "featureflags"}, featureflag.Namespace, featureflag)
	// TODO: Until #38113 is merged, we can't use Subresource
	//action.Subresource = "status"
	f.actions = append(f.actions, action)
}

func getKey(featureflag *featurecontroller.FeatureFlag, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(featureflag)
	if err != nil {
		t.Errorf("Unexpected error getting key for featureflag %v: %v", featureflag.Name, err)
		return ""
	}
	return key
}

// TestCreateDeployment tests that a configmap is created automatically if a new CRD FeatureFlag is created
func TestCreatesDeployment(t *testing.T) {
	f := newFixture(t)
	featureflag := newFeatureFlag("test", int32Ptr(1))

	f.featureflagLister = append(f.featureflagLister, featureflag)
	f.objects = append(f.objects, featureflag)

	expConfig := newConfigMap(featureflag)
	f.expectCreateConfigMapAction(expConfig)
	f.expectUpdateFooStatusAction(featureflag)

	f.run(getKey(featureflag, t))
}

// TestDoNothing tests that the controller takes no action if the configmap for the CRD FeatureFlag exists already
func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	featureflag := newFeatureFlag("test", int32Ptr(1))
	d := newConfigMap(featureflag)

	f.featureflagLister = append(f.featureflagLister, featureflag)
	f.objects = append(f.objects, featureflag)
	f.configmapLister = append(f.configmapLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.expectUpdateFooStatusAction(featureflag)

	f.run(getKey(featureflag, t))
}

// // TestUpdateConfig tests that a configmap can be updated
// func TestUpdateConfig(t *testing.T) {
// 	f := newFixture(t)
// 	featureflag := newFeatureFlag("test", int32Ptr(1))
// 	d := newConfigMap(featureflag)

// 	// Update replicas
// 	featureflag.Spec.Replicas = int32Ptr(2)
// 	expDeployment := newConfigMap(featureflag)

// 	f.featureflagLister = append(f.featureflagLister, featureflag)
// 	f.objects = append(f.objects, featureflag)
// 	f.configmapLister = append(f.configmapLister, d)
// 	f.kubeobjects = append(f.kubeobjects, d)

// 	f.expectUpdateFooStatusAction(featureflag)
// 	f.expectUpdateConfigMapAction(expDeployment)
// 	f.run(getKey(featureflag, t))
// }

func TestNotControlledByUs(t *testing.T) {
	f := newFixture(t)
	featureflag := newFeatureFlag("test", int32Ptr(1))
	d := newConfigMap(featureflag)

	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}

	f.featureflagLister = append(f.featureflagLister, featureflag)
	f.objects = append(f.objects, featureflag)
	f.configmapLister = append(f.configmapLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.runExpectError(getKey(featureflag, t))
}

func int32Ptr(i int32) *int32 { return &i }

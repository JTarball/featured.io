// Has it is a library
package feature_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubernetes "k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"

	"github.com/featured.io/pkg/controllers/feature"
)

var (
	configMapsGroup = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
)

func newConfigMapListAction(ns string, configMap *corev1.ConfigMap) kubetesting.ListActionImpl {
	return kubetesting.NewListAction(configMapsGroup, schema.GroupVersionKind{Kind: "ConfigMap", Version: "v1"}, ns, metav1.ListOptions{})
}

func newConfigMapUpdateAction(ns string, configMap *corev1.ConfigMap) kubetesting.UpdateActionImpl {
	return kubetesting.NewUpdateAction(configMapsGroup, ns, configMap)
}

func newConfigMapGetAction(ns, name string) kubetesting.GetActionImpl {
	return kubetesting.NewGetAction(configMapsGroup, ns, name)
}

func newConfigMapCreateAction(ns string, configMap *corev1.ConfigMap) kubetesting.CreateActionImpl {
	return kubetesting.NewCreateAction(configMapsGroup, ns, configMap)
}

func newConfigMapDeleteAction(ns string, name string) kubetesting.DeleteActionImpl {
	return kubetesting.NewDeleteAction(configMapsGroup, ns, name)
}

// TestConfigMapControlGetCreateOrUpdate tests the CreateOrUpdateConfigMap method
func TestConfigMapControlGetCreateOrUpdate(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testconfigmap1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name               string
		configMap          *corev1.ConfigMap
		getConfigMapResult *corev1.ConfigMap
		errorOnGet         error
		errorOnCreation    error
		expActions         []kubetesting.Action
		expectErr          bool
	}{
		{
			name:               "A new configmap should create a new configmap.",
			configMap:          testConfigMap,
			getConfigMapResult: nil,
			errorOnGet:         kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:    nil,
			expActions: []kubetesting.Action{
				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
				newConfigMapCreateAction(testns, testConfigMap),
			},
			expectErr: false,
		},
		{
			name:               "A new configmap should error when create a new configmap fails.",
			configMap:          testConfigMap,
			getConfigMapResult: nil,
			errorOnGet:         kubeerrors.NewNotFound(schema.GroupResource{}, ""),
			errorOnCreation:    errors.New("wanted error"),
			expActions: []kubetesting.Action{
				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
				newConfigMapCreateAction(testns, testConfigMap),
			},
			expectErr: true,
		},
		{
			name:               "An existent configmap should update the configmap.",
			configMap:          testConfigMap,
			getConfigMapResult: testConfigMap,
			errorOnGet:         nil,
			errorOnCreation:    nil,
			expActions: []kubetesting.Action{
				newConfigMapGetAction(testns, testConfigMap.ObjectMeta.Name),
				newConfigMapUpdateAction(testns, testConfigMap),
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Mock Kubernetes Client
			mcli := &kubernetes.Clientset{}
			mcli.AddReactor("get", "configmaps", func(action kubetesting.Action) (bool, runtime.Object, error) {
				return true, test.getConfigMapResult, test.errorOnGet
			})
			mcli.AddReactor("create", "configmaps", func(action kubetesting.Action) (bool, runtime.Object, error) {
				return true, nil, test.errorOnCreation
			})

			control := feature.NewConfigMapControl(mcli)
			_, err := control.CreateOrUpdateConfigMap(testns, test.configMap)

			if test.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expActions, mcli.Actions()) // Check calls to kubernetes client.
			}
		})
	}
}

// TestConfigMapControlDelete tests the Delete method
func TestConfigMapControlDelete(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testconfigmap1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name            string
		configMap       *corev1.ConfigMap
		errorOnGet      error
		errorOnCreation error
		expActions      []kubetesting.Action
		expectErr       bool
	}{
		{
			name:            "Delete a configmap.",
			configMap:       testConfigMap,
			errorOnGet:      nil,
			errorOnCreation: nil,
			expActions: []kubetesting.Action{
				newConfigMapDeleteAction(testns, testConfigMap.Name),
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Mock Kubernetes Client
			mcli := &kubernetes.Clientset{}
			mcli.AddReactor("delete", "configmaps", func(action kubetesting.Action) (bool, runtime.Object, error) {
				return true, nil, test.errorOnGet
			})

			control := feature.NewConfigMapControl(mcli)
			err := control.DeleteConfigMap(testns, test.configMap.Name)

			if test.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expActions, mcli.Actions()) // Check calls to kubernetes client.
			}
		})
	}
}

// TestConfigMapControlList tests the List method
func TestConfigMapControlList(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testconfigmap1",
			ResourceVersion: "10",
		},
	}

	testns := "testns"

	tests := []struct {
		name                string
		configMap           *corev1.ConfigMap
		listConfigMapResult *corev1.ConfigMapList
		errorOnGet          error
		errorOnCreation     error
		expActions          []kubetesting.Action
		expectErr           bool
	}{
		{
			name:                "Gather a list of configmaps where no items are returned.",
			listConfigMapResult: &corev1.ConfigMapList{},
			errorOnGet:          nil,
			errorOnCreation:     nil,
			expActions: []kubetesting.Action{
				newConfigMapListAction(testns, testConfigMap),
			},
			expectErr: false,
		},
		{
			name: "Gather a list of configmaps where items are returned.",
			listConfigMapResult: &corev1.ConfigMapList{
				Items: []corev1.ConfigMap{
					corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-configmap",
							Namespace:       testns,
							OwnerReferences: []metav1.OwnerReference{},
						},
						Data: map[string]string{
							"config.yml": "{\"vhosts\": [{\"name\": \"/backend\"}",
						},
					},
				},
			},
			errorOnGet:      nil,
			errorOnCreation: nil,
			expActions: []kubetesting.Action{
				newConfigMapListAction(testns, testConfigMap),
			},
			expectErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Mock Kubernetes Client
			mcli := &kubernetes.Clientset{}
			mcli.AddReactor("list", "configmaps", func(action kubetesting.Action) (bool, runtime.Object, error) {
				return true, test.listConfigMapResult, test.errorOnGet
			})

			control := feature.NewConfigMapControl(mcli)
			list, err := control.ListConfigMaps(testns)

			if test.expectErr {
				require.Error(t, err)
			} else {
				require.Equal(t, test.listConfigMapResult, list)
				require.NoError(t, err)
				require.Equal(t, test.expActions, mcli.Actions()) // Check calls to kubernetes client.
			}
		})
	}
}

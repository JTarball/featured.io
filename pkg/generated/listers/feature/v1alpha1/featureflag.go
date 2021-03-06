/*
Copyright 2020 Danvir Guram

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/featured.io/pkg/apis/feature/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// FeatureFlagLister helps list FeatureFlags.
// All objects returned here must be treated as read-only.
type FeatureFlagLister interface {
	// List lists all FeatureFlags in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.FeatureFlag, err error)
	// FeatureFlags returns an object that can list and get FeatureFlags.
	FeatureFlags(namespace string) FeatureFlagNamespaceLister
	FeatureFlagListerExpansion
}

// featureFlagLister implements the FeatureFlagLister interface.
type featureFlagLister struct {
	indexer cache.Indexer
}

// NewFeatureFlagLister returns a new FeatureFlagLister.
func NewFeatureFlagLister(indexer cache.Indexer) FeatureFlagLister {
	return &featureFlagLister{indexer: indexer}
}

// List lists all FeatureFlags in the indexer.
func (s *featureFlagLister) List(selector labels.Selector) (ret []*v1alpha1.FeatureFlag, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.FeatureFlag))
	})
	return ret, err
}

// FeatureFlags returns an object that can list and get FeatureFlags.
func (s *featureFlagLister) FeatureFlags(namespace string) FeatureFlagNamespaceLister {
	return featureFlagNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// FeatureFlagNamespaceLister helps list and get FeatureFlags.
// All objects returned here must be treated as read-only.
type FeatureFlagNamespaceLister interface {
	// List lists all FeatureFlags in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.FeatureFlag, err error)
	// Get retrieves the FeatureFlag from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.FeatureFlag, error)
	FeatureFlagNamespaceListerExpansion
}

// featureFlagNamespaceLister implements the FeatureFlagNamespaceLister
// interface.
type featureFlagNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all FeatureFlags in the indexer for a given namespace.
func (s featureFlagNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.FeatureFlag, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.FeatureFlag))
	})
	return ret, err
}

// Get retrieves the FeatureFlag from the indexer for a given namespace and name.
func (s featureFlagNamespaceLister) Get(name string) (*v1alpha1.FeatureFlag, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("featureflag"), name)
	}
	return obj.(*v1alpha1.FeatureFlag), nil
}

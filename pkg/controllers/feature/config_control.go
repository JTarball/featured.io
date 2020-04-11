package feature

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	log "github.com/sirupsen/logrus"
)

// ConfigMapControlInterface defines the interface that the
// ClusterController uses to create Configmaps. It is implemented as an
// interface to enable testing.
type ConfigMapControlInterface interface {
	GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error)
	CreateConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	UpdateConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error)
	CreateOrUpdateConfigMap(namespace string, np *corev1.ConfigMap) (*corev1.ConfigMap, error)
	DeleteConfigMap(namespace string, name string) error
	ListConfigMaps(namespace string) (*corev1.ConfigMapList, error)
}

// ConfigMapControl is the configMap service implementation using API calls to kubernetes.
type ConfigMapControl struct {
	kubeClient kubernetes.Interface
	logger     *log.Entry
}

// NewConfigMapControl creates a concrete implementation of the ConfigMapControlInterface.
func NewConfigMapControl(kubeClient kubernetes.Interface) ConfigMapControlInterface {

	logger := log.WithFields(log.Fields{
		"service": "k8s.configMap",
	})

	return &ConfigMapControl{
		kubeClient: kubeClient,
		logger:     logger,
	}
}

// GetConfigMap get a configmap resource given the name and namespace
func (p *ConfigMapControl) GetConfigMap(namespace string, name string) (*corev1.ConfigMap, error) {
	// TODO: Need to fix context
	configMap, err := p.kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configMap, err
}

// CreateConfigMap creates a configmap resource
func (p *ConfigMapControl) CreateConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	// TODO: Need to fix context and create options
	config, err := p.kubeClient.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		return config, err
	}
	p.logger.WithFields(log.Fields{"namespace": namespace, "configMap": configMap.Name}).Info("configMap created")
	return config, nil
}

// UpdateConfigMap updates a configmap resource
func (p *ConfigMapControl) UpdateConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	// TODO: Need to fix context and create options
	config, err := p.kubeClient.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
	if err != nil {
		return config, err
	}
	p.logger.WithField("namespace", namespace).WithField("configMap", configMap.Name).Infof("configMap updated")
	return config, nil
}

// CreateOrUpdateConfigMap updates a configmap resource
func (p *ConfigMapControl) CreateOrUpdateConfigMap(namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	storedConfigMap, err := p.GetConfigMap(namespace, configMap.Name)
	if err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return p.CreateConfigMap(namespace, configMap)
		}
		return storedConfigMap, err
	}

	// Already exists, need to Update.
	// Set the correct resource version to ensure we are on the latest version. This way the only valid
	// namespace is our spec(https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency),
	// we will replace the current namespace state.
	configMap.ResourceVersion = storedConfigMap.ResourceVersion
	return p.UpdateConfigMap(namespace, configMap)
}

// DeleteConfigMap deletes a configmap resource
func (p *ConfigMapControl) DeleteConfigMap(namespace string, name string) error {
	// TODO: Need to fix context and create options
	return p.kubeClient.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// ListConfigMaps lists all the configmaps for a given namespace
func (p *ConfigMapControl) ListConfigMaps(namespace string) (*corev1.ConfigMapList, error) {
	// TODO: Need to fix contextS
	return p.kubeClient.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
}

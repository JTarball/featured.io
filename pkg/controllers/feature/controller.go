package feature

// Inspired by
// https://github.com/oracle/mysql-operator/blob/master/pkg/controllers/cluster/controller.go
// https://github.com/kubernetes/sample-controller
// https://medium.com/speechmatics/how-to-write-kubernetes-custom-controllers-in-go-8014c4a04235

// Try and keep things in this file for now will likely have to abstract/modularise some of the function
// as the complexity increases

// We can identify two main tasks in the controller workflow:
//  - Use informers to keep track of add/update/delete events for the Kubernetes resources that we want to know about. “Keeping track” involves storing them in a local cache (thread-safe store) and also adding them to a workqueue.
//  - Consume items from the workqueue and process them.

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	samplev1alpha1 "github.com/featured.io/pkg/apis/feature/v1alpha1"
	clientset "github.com/featured.io/pkg/generated/clientset/versioned"
	samplescheme "github.com/featured.io/pkg/generated/clientset/versioned/scheme"
	informers "github.com/featured.io/pkg/generated/informers/externalversions/feature/v1alpha1"
	listers "github.com/featured.io/pkg/generated/listers/feature/v1alpha1"
)

const controllerAgentName = "feature-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a FeatureFlag is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a FeatureFlag fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Deployment already existing
	MessageResourceExists = "Resource %q already exists and is not managed by FeatureFlag"
	// MessageResourceSynced is the message used for an Event fired when a FeatureFlag
	// is synced successfully
	MessageResourceSynced = "FeatureFlag synced successfully"
)

// FeatureController is the controller implementation for Foo resources
type FeatureController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// featureclientset is a clientset for our own API group
	featureclientset clientset.Interface

	configmapsLister corelisters.ConfigMapLister
	configmapsSynced cache.InformerSynced
	// configmapControl enables control of ConfigMaps associated with FeatureFlag
	configmapControl ConfigMapControlInterface

	featureflagsLister listers.FeatureFlagLister
	featureflagsSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewFeatureController returns a new sample controller
func NewFeatureController(
	kubeclientset kubernetes.Interface,
	featureclientset clientset.Interface,
	configmapInformer coreinformers.ConfigMapInformer,
	featureflagInformer informers.FeatureFlagInformer) *FeatureController {

	// Create event broadcaster
	// Add sample-controller types to the default Kubernetes Scheme so Events can be
	// logged for sample-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &FeatureController{
		kubeclientset:      kubeclientset,
		featureclientset:   featureclientset,
		configmapsLister:   configmapInformer.Lister(),
		configmapsSynced:   configmapInformer.Informer().HasSynced,
		configmapControl:   NewConfigMapControl(kubeclientset),
		featureflagsLister: featureflagInformer.Lister(),
		featureflagsSynced: featureflagInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "FeatureFlags"),
		recorder:           recorder,
	}

	klog.Info("Setting up event handlers")

	// Set up an event handler for when FeatureFlag resources change
	featureflagInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueFeatureFlag,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueFeatureFlag(new)
		},
	})

	// Set up an event handler for when ConfigMap resources change. This
	// handler will lookup the owner of the given ConfigMap, and if it is
	// owned by a FeatureFlag resource will enqueue that FeatureFlag resource for
	// processing. This way, we don't need to implement custom logic for
	// handling ConfigMap resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	configmapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			// NOTE: My understanding this is not relevant for ConfigMap
			// however our control interface for configmap sets the resource version
			// so probably ok
			newCfg := new.(*corev1.ConfigMap)
			oldCfg := old.(*corev1.ConfigMap)

			if newCfg.ResourceVersion == oldCfg.ResourceVersion {
				// Periodic resync will send update events for all known Configmaps.
				return
			}

			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *FeatureController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting FeatureFlag controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.configmapsSynced, c.featureflagsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *FeatureController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *FeatureController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *FeatureController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the FeatureFlag resource with this namespace/name
	featureflag, err := c.featureflagsLister.FeatureFlags(namespace).Get(name)
	if err != nil {
		// The FeatureFlag resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("featureflag '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	configmapName := featureflag.Spec.ConfigMapName
	if configmapName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleError(fmt.Errorf("%s: configmap name must be specified", key))
		return nil
	}

	// Get the ConfigMap with the name specified in FeatureFlag.spec
	// NOTE: Looking at the listers doesnt hit the API
	// where as configmap, err := c.configmapControl.GetConfigMap(featureflag.Namespace, configmapName)
	// would therefore it is far more efficient to do this instead
	configmap, err := c.configmapsLister.ConfigMaps(featureflag.Namespace).Get(configmapName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		configmap, err = c.configmapControl.CreateConfigMap(featureflag.Namespace, newConfigMap(featureflag))
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If the ConfigMap is not controlled by this FeatureFlag resource, we should log
	// a warning to the event recorder and return error msg.
	if !metav1.IsControlledBy(configmap, featureflag) {
		msg := fmt.Sprintf(MessageResourceExists, configmap.Name)
		c.recorder.Event(featureflag, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	// // If this number of the replicas on the FeatureFlag resource is specified, and the
	// // number does not equal the current desired replicas on the Deployment, we
	// // should update the Deployment resource.
	// if foo.Spec.Replicas != nil && *foo.Spec.Replicas != *deployment.Spec.Replicas {
	// 	klog.V(4).Infof("Foo %s replicas: %d, deployment replicas: %d", name, *foo.Spec.Replicas, *deployment.Spec.Replicas)
	// 	deployment, err = c.kubeclientset.AppsV1().Deployments(foo.Namespace).Update(context.TODO(), newDeployment(foo), metav1.UpdateOptions{})
	// }

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err = c.updateFeatureFlagStatus(featureflag, configmap)
	if err != nil {
		return err
	}

	c.recorder.Event(featureflag, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *FeatureController) updateFeatureFlagStatus(featureflag *samplev1alpha1.FeatureFlag, configmap *corev1.ConfigMap) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	featureflagCopy := featureflag.DeepCopy()
	//featureflagCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.featureclientset.FeaturecontrollerV1alpha1().FeatureFlags(featureflag.Namespace).Update(context.TODO(), featureflagCopy, metav1.UpdateOptions{})
	return err
}

// enqueueFeatureFlag takes a FeatureFlag resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than FeatureFlag.
func (c *FeatureController) enqueueFeatureFlag(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the FeatureFlag resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that FeatureFlag resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *FeatureController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a FeatureFlag, we should not do anything more
		// with it.
		if ownerRef.Kind != "FeatureFlag" {
			return
		}

		foo, err := c.featureflagsLister.FeatureFlags(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned object '%s' of featureflag '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueFeatureFlag(foo)
		return
	}
}

// DANVIR: This needs refactoring
//
// newConfigMap creates a new ConfigMap for a FeatureFlag resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the FeatureFlag resource that 'owns' it.
func newConfigMap(featureflag *samplev1alpha1.FeatureFlag) *corev1.ConfigMap {

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      featureflag.Spec.ConfigMapName,
			Namespace: featureflag.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(featureflag, samplev1alpha1.SchemeGroupVersion.WithKind("FeatureFlag")),
			},
		},
		// Data: map[string]string{
		// 	ConfigMapConfKeyName: defaultSentinelConfig(redis.Name, redis.Spec.Sentinels.Quorum),
		// },
	}
}

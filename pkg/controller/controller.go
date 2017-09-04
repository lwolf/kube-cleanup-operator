package controller

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sync"
	"time"
	"k8s.io/client-go/pkg/api/v1"
	"fmt"
)

// NamespaceController watches the kubernetes api for changes to namespaces and
// creates a RoleBinding for that particular namespace.
type NamespaceController struct {
	namespaceInformer cache.SharedIndexInformer
	kclient           *kubernetes.Clientset
}

// NewNamespaceController creates a new NewNamespaceController
func NewNamespaceController(kclient *kubernetes.Clientset) *NamespaceController {
	namespaceWatcher := &NamespaceController{}

	// Create informer for watching Namespaces
	namespaceInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.Core().Namespaces().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.Core().Namespaces().Watch(options)
			},
		},
		&v1.Namespace{},
		3*time.Minute,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
	namespaceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: namespaceWatcher.doSomeMagic,
	})

	namespaceWatcher.kclient = kclient
	namespaceWatcher.namespaceInformer = namespaceInformer

	return namespaceWatcher
}

// Run starts the process for listening for namespace changes and acting upon those changes.
func (c *NamespaceController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group as we're about to execute a go function
	wg.Add(1)

	// Execute go function
	go c.namespaceInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

func (c *NamespaceController) doSomeMagic(obj interface{}) {
	namespaceObj := obj.(*v1.Namespace)
	namespaceName := namespaceObj.Name
	fmt.Println("~~~~~~~> Found namespace", namespaceName)
}
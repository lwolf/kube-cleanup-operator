package controller

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"log"
	"reflect"
	"sync"
	"time"
)

// PodController watches the kubernetes api for changes to Pods and
// delete completed Pods without specific annotation
type PodController struct {
	podInformer cache.SharedIndexInformer
	kclient     *kubernetes.Clientset
}

type CreatedByAnnotation struct {
	Kind       string
	ApiVersion string
	Reference  struct {
		Kind            string
		Namespace       string
		Name            string
		Uid             string
		ApiVersion      string
		ResourceVersion string
	}
}

// NewPodController creates a new NewPodController
func NewPodController(kclient *kubernetes.Clientset, opts map[string]string) *PodController {
	podWatcher := &PodController{}

	// Create informer for watching Namespaces
	podInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.CoreV1().Pods(opts["namespace"]).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.CoreV1().Pods(opts["namespace"]).Watch(options)
			},
		},
		&v1.Pod{},
		time.Second*30,
		cache.Indexers{},
	)
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(cur interface{}) {
			podWatcher.doTheMagic(cur)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				podWatcher.doTheMagic(cur)
			}
		},
	})

	podWatcher.kclient = kclient
	podWatcher.podInformer = podInformer

	return podWatcher
}

// Run starts the process for listening for pod changes and acting upon those changes.
func (c *PodController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	log.Printf("Listening for changes...")
	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group as we're about to execute a go function
	wg.Add(1)

	// Execute go function
	go c.podInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

func (c *PodController) doTheMagic(cur interface{}) {
	podObj := cur.(*v1.Pod)
	// Skip Pods in Running or Pending state
	if podObj.Status.Phase != "Succeeded" {
		return
	}
	var createdMeta CreatedByAnnotation
	json.Unmarshal([]byte(podObj.ObjectMeta.Annotations["kubernetes.io/created-by"]), &createdMeta)
	if createdMeta.Reference.Kind != "Job" {
		return
	}
	restartCounts := podObj.Status.ContainerStatuses[0].RestartCount
	if restartCounts == 0 {
		log.Printf("Going to delete pod '%s'", podObj.Name)
		// Delete Pod
		var po metav1.DeleteOptions
		c.kclient.CoreV1().Pods(podObj.Namespace).Delete(podObj.Name, &po)

		log.Printf("Going to delete job '%s'", createdMeta.Reference.Name)
		// Delete Job itself
		var jo metav1.DeleteOptions
		c.kclient.BatchV1Client.Jobs(createdMeta.Reference.Namespace).Delete(createdMeta.Reference.Name, &jo)
	}
}

package controller

import (
	"k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	job "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
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

// NewPodController creates a new NewPodController
func NewPodController(kclient *kubernetes.Clientset, opts map[string]string) *PodController {
	podWatcher := &PodController{}
	jobInformer := job.NewJobInformer(kclient, opts["namespace"], time.Second*30, cache.Indexers{})
	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	podWatcher.podInformer = jobInformer

	return podWatcher
}

// Run starts the process for listening for pod changes and acting upon those changes.
func (c *PodController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	log.Println("Listening for changes...")
	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group as we're about to execute a go function
	wg.Add(1)

	// Execute go function
	go c.podInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

func shouldDeleteJob(job *v1.Job) bool {
	return job.Status.CompletionTime != nil && ((job.Status.Succeeded > 0 && job.Status.Failed == 0) || (true))
}

func (c *PodController) doTheMagic(cur interface{}) {
	job := cur.(*v1.Job)
	// Skip Pods in Running or Pending state
	if !shouldDeleteJob(job) {
		return
	}
	log.Printf("Deleting job %s", job.ObjectMeta.Name)
	if err := c.kclient.Batch().Jobs("namespace").Delete(job.ObjectMeta.Name, &metav1.DeleteOptions{}); err != nil {
		log.Println(err)
	}
}

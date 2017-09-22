package main

import (
	"log"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	informers "k8s.io/client-go/informers/batch/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// JobController watches the kubernetes api for changes to Jobs and
// delete completed Jobs without specific annotation
type JobController struct {
	jobInformer cache.SharedIndexInformer
	kclient     *kubernetes.Clientset
}

// NewJobController creates a new NewJobController
func NewJobController(kclient *kubernetes.Clientset, opts map[string]string) *JobController {
	jobWatcher := &JobController{}
	jobInformer := informers.NewJobInformer(kclient, opts["namespace"], time.Second*30, cache.Indexers{})
	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(cur interface{}) {
			jobWatcher.maybeDeleteJob(cur)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				jobWatcher.maybeDeleteJob(cur)
			}
		},
	})

	jobWatcher.kclient = kclient
	jobWatcher.jobInformer = jobInformer

	return jobWatcher
}

// Run starts the process for listening for job changes and acting upon those changes.
func (c *JobController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {
	log.Infof("Listening for changes...")
	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group as we're about to execute a go function
	wg.Add(1)

	// Execute go function
	go c.jobInformer.Run(stopCh)

	// Wait till we receive a stop signal
	<-stopCh
}

func (c *JobController) maybeDeleteJob(cur interface{}) {
	job := cur.(*v1.Job)
	log.WithFields(log.Fields{
		"type":      "Job",
		"name":      job.ObjectMeta.Name,
		"namespace": job.ObjectMeta.Namespace,
	}).Infof("Found")
	if !shouldDeleteJob(job) {
		return
	}
	log.WithFields(log.Fields{
		"type":      "Job",
		"name":      job.ObjectMeta.Name,
		"namespace": job.ObjectMeta.Namespace,
	}).Infof("Deleting")
	if err := c.kclient.Batch().Jobs(job.ObjectMeta.Namespace).Delete(job.ObjectMeta.Name, &metav1.DeleteOptions{}); err != nil {
		log.WithFields(log.Fields{
			"type":      "Job",
			"name":      job.ObjectMeta.Name,
			"namespace": job.ObjectMeta.Namespace,
		}).Warn(err)
	}
}

func shouldDeleteJob(job *v1.Job) bool {
	return job.Status.CompletionTime != nil && ((job.Status.Succeeded > 0 && job.Status.Failed == 0) || (timeSinceCompletion(job) > (3 * 24 * time.Hour)))
}

func timeSinceCompletion(job *v1.Job) time.Duration {
	return metav1.Now().Time.Sub(job.Status.CompletionTime.Time)
}

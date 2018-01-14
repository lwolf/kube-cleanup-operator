package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"log"
	"reflect"
	"strconv"
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

	keepSuccessHours, _ := strconv.Atoi(opts["keepSuccessHours"])
	keepFailedHours, _ := strconv.Atoi(opts["keepFailedHours"])
	keepPendingHours, _ := strconv.Atoi(opts["keepPendingHours"])
	dryRun, _ := strconv.ParseBool(opts["dryRun"])
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
			podWatcher.doTheMagic(cur, keepSuccessHours, keepFailedHours, keepPendingHours, dryRun)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				podWatcher.doTheMagic(cur, keepSuccessHours, keepFailedHours, keepPendingHours, dryRun)
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

func (c *PodController) doTheMagic(cur interface{}, keepSuccessHours int, keepFailedHours int, keepPendingHours int, dryRun bool) {
	podObj := cur.(*v1.Pod)

	parentJobName := ""

	// Going all over the owners, looking for a job, usually there is only one owner
	for _, ow := range podObj.OwnerReferences {
		if ow.Kind == "Job" {
			parentJobName = ow.Name
		}
		var gopts metav1.GetOptions
		job,_ :=c.kclient.BatchV1Client.Jobs(podObj.Namespace).Get(ow.Name, gopts)
		log.Printf("The longth of jw is %d", len(job.OwnerReferences))
		for _, jw := range job.OwnerReferences {
			log.Printf("------  Job owner name: %s kind: %s  ------", jw.Name, jw.Kind)
		}


	}
	// If we couldn't extract the job's name, return
	if parentJobName == "" {
		return
	}

	executionTimeHours := c.getExecutionTimeHours(podObj)
	log.Printf("Checking pod %s with %s status that was executed %f hours ago", podObj.Name, podObj.Status.Phase, executionTimeHours)
	switch podObj.Status.Phase {
	case v1.PodSucceeded:
		if keepSuccessHours == 0 || (keepSuccessHours > 0 && executionTimeHours > float32(keepSuccessHours)) {
			c.deleteObjects(podObj, parentJobName, dryRun)
		}
	case v1.PodFailed:
		if keepFailedHours == 0 || (keepFailedHours > 0 && executionTimeHours > float32(keepFailedHours)) {
			c.deleteObjects(podObj, parentJobName, dryRun)
		}
	case v1.PodPending:
		if keepPendingHours > 0 && executionTimeHours > float32(keepPendingHours) {
			c.deleteObjects(podObj, parentJobName, dryRun)
		}
	default:
		return
	}
}

// method to calcualte the hours that passed since the pod's excecution end time
func (c *PodController) getExecutionTimeHours(podObj *v1.Pod) (executionTimeHours float32) {
	executionTimeHours = 0.0
	currentUnixTime := time.Now().Unix()
	podConditions := podObj.Status.Conditions
	var pc v1.PodCondition
	for _, pc = range podConditions {
		// Looking for the time when pod's condition "Ready" became "false" (equals end of execution)
		if pc.Type == v1.PodReady && pc.Status == v1.ConditionFalse {
			executionTimeUnix := pc.LastTransitionTime.Unix()
			executionTimeHours = (float32(currentUnixTime) - float32(executionTimeUnix)) / float32(3600)
		}
	}

	return
}

func (c *PodController) deleteObjects(podObj *v1.Pod, parentJobName string, dryRun bool) {
	// Delete Pod
	if !dryRun {
		log.Printf("Deleting pod '%s'", podObj.Name)
		var po metav1.DeleteOptions
		c.kclient.CoreV1().Pods(podObj.Namespace).Delete(podObj.Name, &po)
	} else {
		log.Printf("Pod '%s' would have been deleted", podObj.Name)
	}
	// Delete Job itself
	if !dryRun {
		log.Printf("Deleting job '%s'", parentJobName)
		var jo metav1.DeleteOptions
		c.kclient.BatchV1Client.Jobs(podObj.Namespace).Delete(parentJobName, &jo)
	} else {
		log.Printf("Job '%s' would have been deleted", parentJobName)
	}
	return
}

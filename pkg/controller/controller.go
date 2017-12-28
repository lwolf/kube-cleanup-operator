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
	"strconv"
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

	keepSuccessDays, _ := strconv.Atoi(opts["keepSuccessDays"])
	keepFailedDays, _ := strconv.Atoi(opts["keepFailedDays"])
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
			podWatcher.doTheMagic(cur, keepSuccessDays, keepFailedDays)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				podWatcher.doTheMagic(cur, keepSuccessDays, keepFailedDays)
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

func (c *PodController) doTheMagic(cur interface{}, keepSuccessDays int, keepFailedDays int) {
	podObj := cur.(*v1.Pod)
	// handle jobs only
	var createdMeta CreatedByAnnotation
	json.Unmarshal([]byte(podObj.ObjectMeta.Annotations["kubernetes.io/created-by"]), &createdMeta)
	if createdMeta.Reference.Kind != "Job" {
		return
	}
	// if restartCount is not 0, do not delete
	restartCounts := podObj.Status.ContainerStatuses[0].RestartCount
	if restartCounts != 0 {
		return
	}

	executionTimeDays := c.getExecutionTimeDays(podObj)
	log.Printf("Checking pod %s with %s status that was executed %f days ago", podObj.Name, podObj.Status.Phase, executionTimeDays)
	switch podObj.Status.Phase{
	case v1.PodSucceeded:
		if (keepSuccessDays == 0 || (keepSuccessDays > 0 && executionTimeDays > float32(keepSuccessDays))){
			c.deleteObjects(podObj, createdMeta)
		}
	case v1.PodFailed:
		if (keepFailedDays == 0 || (keepFailedDays > 0 && executionTimeDays > float32(keepFailedDays))){
			c.deleteObjects(podObj, createdMeta)
		}
	default:
		return
	}
}

// method to calcualte the days that passed since the pod's excecution end time
func (c *PodController) getExecutionTimeDays(podObj *v1.Pod) (executionTimeDays float32){
	executionTimeDays = 0.0
	currentUnixTime := time.Now().Unix()
	podConditions := podObj.Status.Conditions
	var pc v1.PodCondition
	for _, pc = range podConditions{
		// Looking for the time when pod's condition "Ready" became "false" (equals end of execution)
		if pc.Type == v1.PodReady && pc.Status == v1.ConditionFalse{
			executionTimeUnix := pc.LastTransitionTime.Unix()
			executionTimeDays = (float32(currentUnixTime) - float32(executionTimeUnix)) / float32(86400)
		}
	}

	return
}

func (c *PodController) deleteObjects(podObj *v1.Pod, createdMeta CreatedByAnnotation) {
	log.Printf("Going to delete pod '%s'", podObj.Name)
	// Delete Pod
	//var po metav1.DeleteOptions
	//c.kclient.CoreV1().Pods(podObj.Namespace).Delete(podObj.Name, &po)

	log.Printf("Going to delete job '%s'", createdMeta.Reference.Name)
	// Delete Job itself
	//var jo metav1.DeleteOptions
	//c.kclient.BatchV1Client.Jobs(createdMeta.Reference.Namespace).Delete(createdMeta.Reference.Name, &jo)

	return

}
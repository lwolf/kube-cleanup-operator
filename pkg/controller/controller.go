package controller

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/VictoriaMetrics/metrics"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func metricName(name string, namespace string) string {
	return fmt.Sprintf(`%s{namespace=%q}`, name, namespace)
}

const (
	resyncPeriod           = time.Second * 30
	podDeletedMetric       = "pods_deleted_total"
	podDeletedFailedMetric = "pods_deleted_failed_total"
	jobDeletedFailedMetric = "jobs_deleted_failed_total"
	jobDeletedMetric       = "jobs_deleted_total"
)

// Kleaner watches the kubernetes api for changes to Pods and Jobs and
// delete those according to configured timeouts
type Kleaner struct {
	podInformer cache.SharedIndexInformer
	jobInformer cache.SharedIndexInformer
	kclient     *kubernetes.Clientset

	deleteSuccessfulAfter  time.Duration
	deleteFailedAfter      time.Duration
	deletePendingAfter     time.Duration
	deleteOrphanedAfter    time.Duration
	deleteEvictedAfter     time.Duration
	deleteTerminatedAfter  time.Duration
	deleteTerminatingAfter time.Duration

	ignoreOwnedByCronjob bool

	labelSelector string

	dryRun bool
	ctx    context.Context
	stopCh <-chan struct{}
}

// NewKleaner creates a new NewKleaner
func NewKleaner(ctx context.Context, kclient *kubernetes.Clientset, namespace string, dryRun bool, deleteSuccessfulAfter,
	deleteFailedAfter, deletePendingAfter, deleteOrphanedAfter, deleteEvictedAfter time.Duration, deleteTerminatedAfter time.Duration, deleteTerminatingAfter time.Duration, ignoreOwnedByCronjob bool,
	labelSelector string,
	stopCh <-chan struct{}) *Kleaner {
	jobInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labelSelector
				return kclient.BatchV1().Jobs(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return kclient.BatchV1().Jobs(namespace).Watch(ctx, options)
			},
		},
		&batchv1.Job{},
		resyncPeriod,
		cache.Indexers{},
	)
	// Create informer for watching Namespaces
	podInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = labelSelector
				return kclient.CoreV1().Pods(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return kclient.CoreV1().Pods(namespace).Watch(ctx, options)
			},
		},
		&corev1.Pod{},
		resyncPeriod,
		cache.Indexers{},
	)
	kleaner := &Kleaner{
		dryRun:                 dryRun,
		kclient:                kclient,
		ctx:                    ctx,
		stopCh:                 stopCh,
		deleteSuccessfulAfter:  deleteSuccessfulAfter,
		deleteFailedAfter:      deleteFailedAfter,
		deletePendingAfter:     deletePendingAfter,
		deleteOrphanedAfter:    deleteOrphanedAfter,
		deleteEvictedAfter:     deleteEvictedAfter,
		deleteTerminatedAfter:  deleteTerminatedAfter,
		deleteTerminatingAfter: deleteTerminatingAfter,
		ignoreOwnedByCronjob:   ignoreOwnedByCronjob,
		labelSelector:          labelSelector,
	}
	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			if !reflect.DeepEqual(old, new) {
				kleaner.Process(new)
			}
		},
	})
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old, new interface{}) {
			if !reflect.DeepEqual(old, new) {
				kleaner.Process(new)
			}
		},
	})

	kleaner.podInformer = podInformer
	kleaner.jobInformer = jobInformer

	return kleaner
}

func (c *Kleaner) periodicCacheCheck() {
	ticker := time.NewTicker(2 * resyncPeriod)
	for {
		select {
		case <-c.stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			for _, job := range c.jobInformer.GetStore().List() {
				c.Process(job)
			}
			for _, obj := range c.podInformer.GetStore().List() {
				c.Process(obj)
			}
		}
	}
}

// Run starts the process for listening for pod changes and acting upon those changes.
func (c *Kleaner) Run() {
	log.Printf("Listening for changes...")

	go c.podInformer.Run(c.stopCh)
	go c.jobInformer.Run(c.stopCh)

	go c.periodicCacheCheck()

	<-c.stopCh
}

func (c *Kleaner) Process(obj interface{}) {
	switch t := obj.(type) {
	case *batchv1.Job:
		// skip jobs that are already in the deleting process
		if !t.DeletionTimestamp.IsZero() {
			return
		}
		if shouldDeleteJob(t, c.deleteSuccessfulAfter, c.deleteFailedAfter, c.ignoreOwnedByCronjob) {
			c.DeleteJob(t)
		}
	case *corev1.Pod:
		pod := t

		if c.deleteTerminatingAfter > 0 && shouldDeleteTerminatingPod(t, c.deleteOrphanedAfter, c.deletePendingAfter, c.deleteEvictedAfter, c.deleteTerminatingAfter, c.deleteSuccessfulAfter, c.deleteFailedAfter) {
			c.DeletePodWithForce(t)
		} else {
			// skip pods that are already in the deleting process
			return
		}
		// skip pods related to jobs created by cronjobs if `ignoreOwnedByCronjob` is set
		if c.ignoreOwnedByCronjob && podRelatedToCronJob(pod, c.jobInformer.GetStore()) {
			return
		}
		// normal cleanup flow
		if shouldDeletePod(t, c.deleteOrphanedAfter, c.deletePendingAfter, c.deleteEvictedAfter, c.deleteTerminatedAfter, c.deleteSuccessfulAfter, c.deleteFailedAfter) {
			c.DeletePod(t)
		}
	}
}

func (c *Kleaner) DeleteJob(job *batchv1.Job) {
	if c.dryRun {
		log.Printf("dry-run: Job '%s:%s' would have been deleted", job.Namespace, job.Name)
		return
	}
	log.Printf("Deleting job '%s/%s'", job.Namespace, job.Name)
	propagation := metav1.DeletePropagationForeground
	jo := metav1.DeleteOptions{PropagationPolicy: &propagation}
	if err := c.kclient.BatchV1().Jobs(job.Namespace).Delete(c.ctx, job.Name, jo); ignoreNotFound(err) != nil {
		log.Printf("failed to delete job '%s:%s': %v", job.Namespace, job.Name, err)
		metrics.GetOrCreateCounter(metricName(jobDeletedFailedMetric, job.Namespace)).Inc()
		return
	}
	metrics.GetOrCreateCounter(metricName(jobDeletedMetric, job.Namespace)).Inc()
}

func (c *Kleaner) DeletePod(pod *corev1.Pod) {
	if c.dryRun {
		log.Printf("dry-run: Pod '%s:%s' would have been deleted", pod.Namespace, pod.Name)
		return
	}
	log.Printf("Deleting pod '%s/%s'", pod.Namespace, pod.Name)
	var po metav1.DeleteOptions
	if err := c.kclient.CoreV1().Pods(pod.Namespace).Delete(c.ctx, pod.Name, po); ignoreNotFound(err) != nil {
		log.Printf("failed to delete pod '%s:%s': %v", pod.Namespace, pod.Name, err)
		metrics.GetOrCreateCounter(metricName(podDeletedFailedMetric, pod.Namespace)).Inc()
		return
	}
	metrics.GetOrCreateCounter(metricName(podDeletedMetric, pod.Namespace)).Inc()
}

// In Case If Pod(s) is Stuck in Terminating state just in case to delete with force
// Not goood way to fid the root cause of the issue why pod is stuck in terminating state
func (c *Kleaner) DeletePodWithForce(pod *corev1.Pod) {
	if c.dryRun {
		log.Printf("dry-run: Pod '%s:%s' would have been deleted", pod.Namespace, pod.Name)
		return
	}
	log.Printf("Deleting terminating pod with force '%s/%s'", pod.Namespace, pod.Name)
	var po metav1.DeleteOptions
	gracePeriodSeconds := int64(0)
	po.GracePeriodSeconds = &gracePeriodSeconds
	if err := c.kclient.CoreV1().Pods(pod.Namespace).Delete(c.ctx, pod.Name, po); ignoreNotFound(err) != nil {
		log.Printf("failed to delete pod '%s:%s': %v", pod.Namespace, pod.Name, err)
		metrics.GetOrCreateCounter(metricName(podDeletedFailedMetric, pod.Namespace)).Inc()
		return
	}
	metrics.GetOrCreateCounter(metricName(podDeletedMetric, pod.Namespace)).Inc()
}

package controller

import (
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createJob(ownedByCronJob bool, completed time.Time, active, succeeded, failed int32, conditions []batchv1.JobCondition, annotations map[string]string) *batchv1.Job {
	ts := metav1.NewTime(completed)
	job := batchv1.Job{
		Spec: batchv1.JobSpec{},
		Status: batchv1.JobStatus{
			CompletionTime: &ts,
			Active:         active,
			Succeeded:      succeeded,
			Failed:         failed,
			Conditions:     conditions,
		},
	}
	if ownedByCronJob {
		job.ObjectMeta = metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob"},
			},
		}
	}
	if len(annotations) != 0 {
		job.Annotations = annotations
	}
	return &job
}

func TestKleaner_DeleteJob(t *testing.T) {
	ts := time.Now()
	testCases := map[string]struct {
		jobSpec     *batchv1.Job
		successful  time.Duration
		failed      time.Duration
		ignoreCron  bool
		expected    bool
		annotations bool
	}{
		"jobs owned by cronjobs should be ignored": {
			jobSpec:     createJob(true, ts.Add(-time.Minute), 0, 0, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  true,
			expected:    false,
			annotations: false,
		},
		"jobs owned by cronjobs should be deleted": {
			jobSpec:     createJob(true, ts.Add(-time.Minute), 0, 0, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    false,
			annotations: false,
		},
		"jobs with active pods should not be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 1, 0, 0, []batchv1.JobCondition{}, map[string]string{}), // job.Status.Active > 0
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    false,
			annotations: false,
		},
		"expired successful jobs should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
		"expired successful jobs with annotations respect and empty annotations should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: true,
		},
		"expired successful jobs with annotations respect and disabled != true should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{annotationDisabled: "not true"}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: true,
		},
		"expired successful jobs with annotations respect and disabled == true should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{annotationDisabled: "true"}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    false,
			annotations: true,
		},
		"non-expired successful jobs should not be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Minute * 2,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    false,
			annotations: false,
		},
		"non-expired successful jobs with annotations respect and successful overridden should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 1, 0, []batchv1.JobCondition{}, map[string]string{annotationDeleteSuccessfulAfter: "1s"}),
			successful:  time.Minute * 2,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: true,
		},
		"expired failed jobs should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 0, 1, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
		"non-expired failed jobs should not be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 0, 1, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Minute * 2,
			ignoreCron:  false,
			expected:    false,
			annotations: false,
		},
		"non-expired failed jobs with annotations respect and failed overridden should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 0, 0, 1, []batchv1.JobCondition{}, map[string]string{annotationDeleteFailedAfter: "1s"}),
			successful:  time.Second,
			failed:      time.Minute * 2,
			ignoreCron:  false,
			expected:    true,
			annotations: true,
		},
		"failed (based on JobCondition) but not marked as failed jobs should be deleted": {
			jobSpec: createJob(false, time.Time{}, 0, 0, 0, []batchv1.JobCondition{
				batchv1.JobCondition{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastProbeTime:      metav1.NewTime(ts),
					LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute)),
					Reason:             "DeadlineExceeded",
					Message:            "Job was active longer than specified deadline",
				},
			}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
		"successful but 'active' jobs should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 1, 1, 0, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
		"failed but 'active' jobs should be deleted": {
			jobSpec:     createJob(false, ts.Add(-time.Minute), 1, 0, 1, []batchv1.JobCondition{}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
		"failed (based on JobCondition) but 'active' jobs should be deleted": {
			jobSpec: createJob(false, ts.Add(-time.Minute), 1, 0, 0, []batchv1.JobCondition{
				batchv1.JobCondition{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastProbeTime:      metav1.NewTime(ts),
					LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute)),
					Reason:             "DeadlineExceeded",
					Message:            "Job was active longer than specified deadline",
				},
			}, map[string]string{}),
			successful:  time.Second,
			failed:      time.Second,
			ignoreCron:  false,
			expected:    true,
			annotations: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := shouldDeleteJob(tc.jobSpec, tc.successful, tc.failed, tc.ignoreCron, tc.annotations)
			if result != tc.expected {
				t.Fatalf("failed, expected %v, got %v", tc.expected, result)
			}
		})
	}
}

package controller

import (
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createJob(ownedByCronJob bool, completed time.Time, active, succeeded, failed int32) *batchv1.Job {
	ts := metav1.NewTime(completed)
	job := batchv1.Job{
		Spec: batchv1.JobSpec{},
		Status: batchv1.JobStatus{
			CompletionTime: &ts,
			Active:         active,
			Succeeded:      succeeded,
			Failed:         failed,
		},
	}
	if ownedByCronJob {
		job.ObjectMeta = metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob"},
			},
		}
	}
	return &job
}

func TestKleaner_DeleteJob(t *testing.T) {
	ts := time.Now()
	testCases := map[string]struct {
		jobSpec    *batchv1.Job
		successful time.Duration
		failed     time.Duration
		ignoreCron bool
		expected   bool
	}{
		"jobs owned by cronjobs should be ignored": {
			jobSpec:    createJob(true, ts.Add(-time.Minute), 0, 0, 0),
			successful: time.Second,
			failed:     time.Second,
			ignoreCron: true,
			expected:   false,
		},
		"jobs owned by cronjobs should be deleted": {
			jobSpec:    createJob(true, ts.Add(-time.Minute), 0, 0, 0),
			successful: time.Second,
			failed:     time.Second,
			ignoreCron: false,
			expected:   false,
		},
		"jobs with active pods should not be deleted": {
			jobSpec:    createJob(false, ts.Add(-time.Minute), 1, 0, 0), // job.Status.Active > 0
			successful: time.Second,
			failed:     time.Second,
			ignoreCron: false,
			expected:   false,
		},
		"expired successful jobs should be deleted": {
			jobSpec:    createJob(false, ts.Add(-time.Minute), 0, 1, 0),
			successful: time.Second,
			failed:     time.Second,
			ignoreCron: false,
			expected:   true,
		},
		"non-expired successful jobs should not be deleted": {
			jobSpec:    createJob(false, ts.Add(-time.Minute), 0, 1, 0),
			successful: time.Minute * 2,
			failed:     time.Second,
			ignoreCron: false,
			expected:   false,
		},
		"expired failed jobs should be deleted": {
			jobSpec:    createJob(false, ts.Add(-time.Minute), 0, 0, 1),
			successful: time.Second,
			failed:     time.Second,
			ignoreCron: false,
			expected:   true,
		},
		"non-expired failed jobs should not be deleted": {
			jobSpec:    createJob(false, ts.Add(-time.Minute), 0, 0, 1),
			successful: time.Second,
			failed:     time.Minute * 2,
			ignoreCron: false,
			expected:   false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := shouldDeleteJob(tc.jobSpec, tc.successful, tc.failed, tc.ignoreCron)
			if result != tc.expected {
				t.Fatalf("failed, expected %v, got %v", tc.expected, result)
			}
		})
	}
}

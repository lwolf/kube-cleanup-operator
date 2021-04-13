package controller

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKleaner_DeletePod(t *testing.T) {
	ts := time.Now()
	testCases := map[string]struct {
		podSpec     *corev1.Pod
		orphaned    time.Duration
		pending     time.Duration
		evicted     time.Duration
		successful  time.Duration
		failed      time.Duration
		expected    bool
		annotations bool
	}{
		"expired orphaned pods should be deleted": {
			podSpec: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute * 2)),
						},
					},
				},
			},
			orphaned:    time.Minute,
			pending:     0,
			evicted:     0,
			successful:  0,
			failed:      0,
			expected:    true,
			annotations: false,
		},
		"non expired orphaned pods should not be deleted": {
			podSpec: &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute)),
						},
					},
				},
			},
			orphaned:    time.Minute * 5,
			pending:     0,
			evicted:     0,
			successful:  0,
			failed:      0,
			expected:    false,
			annotations: false,
		},
		"expired, PodSucceeded owned by Job should be deleted": {
			podSpec: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Job",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute * 2)),
						},
					},
				},
			},
			orphaned:    0,
			pending:     0,
			evicted:     0,
			successful:  time.Minute,
			failed:      0,
			expected:    true,
			annotations: false,
		},
		"expired, PodFailed owned by Job should be deleted": {
			podSpec: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "Job",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute * 2)),
						},
					},
				},
			},
			orphaned:    0,
			pending:     0,
			evicted:     0,
			successful:  0,
			failed:      time.Minute,
			expected:    true,
			annotations: false,
		},
		"evicted pods should be deleted": {
			podSpec: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase:  corev1.PodFailed,
					Reason: "Evicted",
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodReady,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute * 2)),
						},
					},
				},
			},
			orphaned:    0,
			pending:     0,
			evicted:     time.Hour,
			successful:  0,
			failed:      0,
			expected:    true,
			annotations: false,
		},
		"expired pending pods should be deleted": {
			podSpec: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:               corev1.PodScheduled,
							Status:             corev1.ConditionFalse,
							LastTransitionTime: metav1.NewTime(ts.Add(-time.Minute * 2)),
						},
					},
				},
			},
			orphaned:    0,
			pending:     time.Minute,
			evicted:     0,
			successful:  0,
			failed:      0,
			expected:    true,
			annotations: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := shouldDeletePod(tc.podSpec, tc.orphaned, tc.pending, tc.evicted, tc.successful, tc.failed, tc.annotations)
			if result != tc.expected {
				t.Fatalf("failed, expected %v, got %v", tc.expected, result)
			}
		})
	}
}

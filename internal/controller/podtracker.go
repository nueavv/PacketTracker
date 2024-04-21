package controller

import (
	networkv1alpha1 "github.com/nueavv/PacketTracker/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// PodImage is the image used for the tracker Pod
	// TODO 수정해야함
	PodImage      = "nginx:latest"
	PodSuffixName = "-tracker"
	Version       = "v1"
)

func (r *PodTrackerReconciler) podForPodTracker(tr *networkv1alpha1.PodTracker, td *corev1.Pod) (*corev1.Pod, error) {
	// Define a new Pod object
	pod := newPod(tr, td)
	// Set PodTracker instance as the owner and controller
	if err := ctrl.SetControllerReference(tr, pod, r.Scheme); err != nil {
		return nil, err
	}
	return pod, nil
}

func newPod(tr *networkv1alpha1.PodTracker, td *corev1.Pod) *corev1.Pod {
	labels := labelsForPodTracker(tr.Name, td.Name)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      td.Name + PodSuffixName,
			Namespace: td.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			NodeName: td.Spec.NodeName,
			Containers: []corev1.Container{
				{
					Name:  "tracker-container",
					Image: PodImage,
				},
			},
		},
	}
}

func labelsForPodTracker(tracker_name string, tracking_pod_name string) map[string]string {
	return map[string]string{"podtracker": tracker_name, "tracker_version": Version, "tracked_pod": tracking_pod_name}
}

func hasTrackerPod(trPods *corev1.PodList, tdPodName string) bool {
	for _, trPod := range trPods.Items {
		if trPod.Name == tdPodName+PodSuffixName {
			return true
		}
	}
	return false
}

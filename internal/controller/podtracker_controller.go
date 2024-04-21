/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkv1alpha1 "github.com/nueavv/PacketTracker/api/v1alpha1"
)

const (
	// PodTrackerFinalizer is the finalizer for the PodTracker
	PodTrackerFinalizer = "finalizer.podtracker.network.io"
)

// PodTrackerReconciler reconciles a PodTracker object
type PodTrackerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers/finalizers,verbs=update
func (r *PodTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling PodTracker")

	tr := &networkv1alpha1.PodTracker{}
	err := r.Get(ctx, req.NamespacedName, tr)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PodTracker resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PodTracker")
		return ctrl.Result{}, err
	}

	// tracked 파드 셀렉터 변경
	logger.Info("PodTracker resource found", "PodTracker", tr.Name)
	tdSelector, err := metav1.LabelSelectorAsSelector(&tr.Spec.PodSelector)
	if err != nil {
		logger.Error(err, "Failed to get PodSelector")
		return ctrl.Result{}, err
	}

	tdListOptions := &client.ListOptions{
		LabelSelector: tdSelector,
	}

	// tracked 파드 리스트 가져오기
	tdPods := &corev1.PodList{}
	err = r.List(ctx, tdPods, tdListOptions)
	if err != nil {
		logger.Error(err, "Failed to list Pods")
		return ctrl.Result{}, err
	}

	// tracker 파드 리스트 가져오기
	trLabels := labels.Set(map[string]string{"podtracker": tr.Name})
	trListOptions := &client.ListOptions{
		LabelSelector: trLabels.AsSelector(),
	}

	trPods := &corev1.PodList{}
	err = r.List(ctx, trPods, trListOptions)
	if err != nil {
		logger.Error(err, "Failed to list Tracker Pods")
		return ctrl.Result{}, err
	}

	if len(tdPods.Items) == 0 {
		logger.Info("No Tracking Pods found")
		if len(trPods.Items) > 0 {
			// TODO
			for _, trPod := range trPods.Items {
				if err := r.Delete(ctx, &trPod); err != nil {
					logger.Error(err, "Failed to delete tracker Pod", "PodName", trPod.Name)
					return ctrl.Result{}, err
				}
				logger.Info("Tracker Pod deleted", "PodName", trPod.Name)
			}
		}
		return ctrl.Result{}, nil
	}

	// tracker pod 생성
	var tdPodNames []string
	for _, tdPod := range tdPods.Items {
		logger.Info(tdPod.Name)
		if !hasTrackerPod(trPods, tdPod.Name) {
			trPod, err := r.podForPodTracker(tr, &tdPod)
			if err != nil {
				logger.Error(err, "Failed to create a new Pod", "PodName", tdPod.Name)
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, trPod); err != nil {
				logger.Error(err, "Failed to create a new Pod", "PodName", tdPod.Name)
				return ctrl.Result{}, err
			}
			logger.Info("Pod created", "PodName", trPod.Name)
		}
		tdPodNames = append(tdPodNames, tdPod.Name+PodSuffixName)
	}

	logger.Info("Number of Tracker Pods:", "Count", len(trPods.Items))
	for _, trPod := range trPods.Items {
		if contains(tdPodNames, trPod.Name) {
			continue
		}

		if err := r.Delete(ctx, &trPod); err != nil {
			logger.Error(err, "Failed to delete tracker Pod", "PodName", trPod.Name)
			return ctrl.Result{}, err
		}
		logger.Info("Tracker Pod deleted", "PodName", trPod.Name)
	}

	return ctrl.Result{}, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
// https://github.com/kubernetes-sigs/controller-runtime/issues/1049
// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder#Builder.Owns
func (r *PodTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.PodTracker{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

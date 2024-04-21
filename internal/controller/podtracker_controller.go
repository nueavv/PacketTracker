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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	networkv1alpha1 "github.com/nueavv/PacketTracker/api/v1alpha1"
)

// PodTrackerReconciler reconciles a PodTracker object
type PodTrackerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=pods,verbs=create;delete;get;list;patch;update;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=network.tracker.io,resources=podtrackers/finalizers,verbs=update
func (r *PodTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling PodTracker")

	podtracker := &networkv1alpha1.PodTracker{}
	err := r.Get(ctx, req.NamespacedName, podtracker)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("PodTracker resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get PodTracker")
		return ctrl.Result{}, err
	}

	
	logger.Info("PodTracker resource found", "PodTracker", podtracker.Name)
	podSelector, err := metav1.LabelSelectorAsSelector(&podtracker.Spec.PodSelector)
	if err != nil {
		logger.Error(err, "Failed to get PodSelector")
		return ctrl.Result{}, err
	}
	listOptions := &client.ListOptions{
		LabelSelector: podSelector,
	}

	// tracked 파드 리스트 가져오기
	trackedPods := &corev1.PodList{}
	err = r.List(ctx, trackedPods, listOptions)
	
	if err != nil {
		logger.Error(err, "Failed to list Pods")
		return ctrl.Result{}, err
	}

	trackerPods := &corev1.PodList{}
	trackerlistOptions := &client.ListOptions{	
		LabelSelector: podSelector,
	}
	trackerlistOptions.LabelSelector.MatchLabels["role"] = "tracker"
	err = r.List(ctx, trackerPods, trackerlistOptions)
	if err != nil {
		logger.Error(err, "Failed to list Tracker Pods")
		return ctrl.Result{}, err
	}

	if len(trackedPods.Items) == 0 {
		logger.Info("No Pod resources found")
		if len(trackerPods)	> 0 {
			// TODO: tracker pod 삭제
		}
		return ctrl.Result{}, nil
	} else {
		for _, pod := range trackedPods.Items {
			logger.Info(pod.Name)
			// Process each pod here
			logger.Info(pod.Spec.NodeName)
			// TODO : tracker pod 생성
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
// https://github.com/kubernetes-sigs/controller-runtime/issues/1049
// https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/builder#Builder.Owns
func (r *PodTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.PodTracker{}).
		// Owns(&cored.Pod{}).
		Complete(r)
}

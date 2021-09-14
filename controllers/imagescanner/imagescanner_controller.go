/*
Copyright 2021.

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

package imagescanner

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

// ImageScannerReconciler reconciles a ImageScanner object
type ImageScannerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagescanners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagescanners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagescanners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageScanner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ImageScannerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)
	job := &eraserv1alpha1.ImageJob{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "imagejob-",
		},
		Spec: eraserv1alpha1.ImageJobSpec{
			JobTemplate: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:            "collector",
							Image:           "aldaircoronel/collector:latest",
							ImagePullPolicy: corev1.PullAlways,
						},
					},
					ServiceAccountName: "collector-controller-manager",
				},
			},
			ImageListName: req.NamespacedName.Name,
		},
	}
	err := r.Create(ctx, job)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// your logic here

	return ctrl.Result{RequeueAfter: time.Minute, Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageScannerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return nil
}

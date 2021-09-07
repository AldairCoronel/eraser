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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// ImageScannerReconciler reconciles a ImageScanner object
type ImageScannerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// ImageScannerReconciler reconciles a ImageScanner object
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
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
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here

	return ctrl.Result{}, nil
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {

	c, err := controller.New("imagescanner-controller", mgr, controller.Options{
		Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &eraserv1alpha1.ImageScanner{}},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

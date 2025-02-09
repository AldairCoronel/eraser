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

package imagejob

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	dockerPath     = "/run/dockershim.sock"
	containerdPath = "/run/containerd/containerd.sock"
	crioPath       = "/run/crio/crio.sock"
	docker         = "docker"
	containerd     = "containerd"
	crio           = "cri-o"
	apiPath        = "apis/eraser.sh/v1alpha1"
	namespace      = "eraser-system"
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

// ImageJobReconciler reconciles a ImageJob object
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagejob-controller", mgr, controller.Options{
		Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ImageJob
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to pods created by ImageJob (eraser pods)
	err = c.Watch(&source.Kind{Type: &v1.Pod{}}, &handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageJob{}, IsController: true})
	if err != nil {
		return err
	}

	return nil
}

func checkNodeFitness(pod *v1.Pod, node *v1.Node) bool {
	nodeInfo := framework.NewNodeInfo()
	_ = nodeInfo.SetNode(node)

	insufficientResource := noderesources.Fits(pod, nodeInfo)

	if len(insufficientResource) != 0 {
		log.Println("Pod does not fit: ", insufficientResource)
		return false
	}

	return true
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;create;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagestatuses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return ctrl.Result{}, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return ctrl.Result{}, err
	}

	imageJob := &eraserv1alpha1.ImageJob{}
	err = r.Get(ctx, req.NamespacedName, imageJob)
	if err != nil {
		imageJob.Status.Phase = eraserv1alpha1.PhaseFailed
		updateJobStatus(ctx, clientset, *imageJob)
		return ctrl.Result{}, err
	}

	if imageJob.Status.Phase == "" {
		nodes := &v1.NodeList{}
		err := r.List(ctx, nodes)
		if err != nil {
			return ctrl.Result{}, err
		}

		imageJob.Status = eraserv1alpha1.ImageJobStatus{
			Desired:   len(nodes.Items),
			Succeeded: 0,
			Failed:    0,
			Phase:     eraserv1alpha1.PhaseRunning,
		}

		updateJobStatus(ctx, clientset, *imageJob)

		for _, n := range nodes.Items {
			n := n
			nodeName := n.Name
			runtime := n.Status.NodeInfo.ContainerRuntimeVersion
			runtimeName := strings.Split(runtime, ":")[0]
			mountPath := getMountPath(runtimeName)
			if mountPath == "" {
				log.Println("Incompatible runtime on node ", nodeName)
				continue
			}

			givenImage := imageJob.Spec.JobTemplate.Spec.Containers[0]
			image := v1.Container{
				Args:            append(givenImage.Args, "--runtime="+runtimeName),
				VolumeMounts:    []v1.VolumeMount{{MountPath: mountPath, Name: runtimeName + "-sock-volume"}},
				Image:           givenImage.Image,
				Name:            givenImage.Name,
				ImagePullPolicy: givenImage.ImagePullPolicy,
				Env:             []v1.EnvVar{{Name: "NODE_NAME", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}}},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"cpu":    resource.MustParse("7m"),
						"memory": resource.MustParse("25Mi"),
					},
					Limits: v1.ResourceList{
						"cpu":    resource.MustParse("8m"),
						"memory": resource.MustParse("30Mi"),
					},
				},
			}

			givenPodSpec := imageJob.Spec.JobTemplate.Spec
			podSpec := v1.PodSpec{
				RestartPolicy:      givenPodSpec.RestartPolicy,
				ServiceAccountName: givenPodSpec.ServiceAccountName,
				Containers:         []v1.Container{image},
				NodeName:           nodeName,
				Volumes:            []v1.Volume{{Name: runtimeName + "-sock-volume", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: mountPath}}}},
			}

			podName := image.Name + "-" + nodeName
			pod := &v1.Pod{
				TypeMeta: metav1.TypeMeta{},
				Spec:     podSpec,
				ObjectMeta: metav1.ObjectMeta{Namespace: "eraser-system",
					Name:            podName,
					Labels:          map[string]string{"name": image.Name},
					OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(imageJob, imageJob.GroupVersionKind())}},
			}

			fitness := checkNodeFitness(pod, &n)

			if fitness {
				err = r.Create(ctx, pod)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}

	} else if imageJob.Status.Phase == eraserv1alpha1.PhaseRunning {
		// get eraser pods
		podList := &v1.PodList{}
		err := r.List(ctx, podList, &client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{"name": imageJob.Spec.JobTemplate.Spec.Containers[0].Name})})
		if err != nil {
			return ctrl.Result{}, err
		}

		failed := 0
		success := 0

		// if all pods are complete, job is complete
		if podsComplete(podList.Items) {
			// get status of pods
			for _, p := range podList.Items {
				if p.Status.Phase == v1.PodSucceeded {
					success++
				} else {
					failed++
				}
			}

			imageJob.Status = eraserv1alpha1.ImageJobStatus{
				Desired:   imageJob.Status.Desired,
				Succeeded: success,
				Failed:    failed,
				Phase:     eraserv1alpha1.PhaseCompleted,
			}

			updateJobStatus(ctx, clientset, *imageJob)

			// transfer results from imageStatus objects to imageList
			statusList := &eraserv1alpha1.ImageStatusList{}
			err = r.List(ctx, statusList)
			if err != nil {
				return ctrl.Result{}, err
			}

			var nodeResult []eraserv1alpha1.NodeResult

			for _, s := range statusList.Items {
				nodeResult = append(nodeResult, eraserv1alpha1.NodeResult{
					Name:   s.Result.Node,
					Images: s.Result.Results,
				})
			}

			imageList := &eraserv1alpha1.ImageList{}
			err = r.Get(ctx, types.NamespacedName{Name: imageJob.Spec.ImageListName}, imageList)
			if err != nil {
				return ctrl.Result{}, err
			}
			updateImageListStatus(ctx, clientset, nodeResult, *imageList)
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageJob{}).
		Complete(r)
}

func getMountPath(runtimeName string) string {
	switch runtimeName {
	case docker:
		return dockerPath
	case containerd:
		return containerdPath
	case crio:
		return crioPath
	default:
		return ""
	}
}

func podsComplete(lst []v1.Pod) bool {
	for _, pod := range lst {
		if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodPending {
			return false
		}
	}
	return true
}

func updateImageListStatus(ctx context.Context, clientset *kubernetes.Clientset, nodeResult []eraserv1alpha1.NodeResult, imageList eraserv1alpha1.ImageList) error {
	imageList.Status = eraserv1alpha1.ImageListStatus{
		Timestamp: &metav1.Time{Time: time.Now()},
		Node:      nodeResult,
	}

	body, err := json.Marshal(imageList)
	if err != nil {
		return err
	}

	// update imagelist object
	_, err = clientset.RESTClient().Put().
		AbsPath(apiPath).
		Name(imageList.Name).
		Resource("imagelists").
		SubResource("status").
		Body(body).DoRaw(ctx)

	if err != nil {
		return err
	}

	return nil
}

func updateJobStatus(ctx context.Context, clientset *kubernetes.Clientset, imageJob eraserv1alpha1.ImageJob) error {
	body, err := json.Marshal(imageJob)
	if err != nil {
		return err
	}

	// update imageJob object
	_, err = clientset.RESTClient().Put().
		AbsPath(apiPath).
		Name(imageJob.Name).
		Resource("imagejobs").
		SubResource("status").
		Body(body).DoRaw(ctx)

	if err != nil {
		return err
	}

	return nil
}

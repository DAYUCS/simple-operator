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

package controllers

import (
	"context"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	simplev1alpha1 "github.com/DAYUCS/simple-operator/api/v1alpha1"
)

// SimpleReconciler reconciles a Simple object
type SimpleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=simple.eximbills.com,resources=simples,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=simple.eximbills.com,resources=simples/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=simple.eximbills.com,resources=simples/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Simple object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *SimpleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Simple Service object if it exists
	simple := &simplev1alpha1.Simple{}
	err := r.Get(ctx, req.NamespacedName, simple)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Simple Servie resource not found. Ignoring since object must be deleted")
			// Exit reconciliation as the object has been deleted
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Simple Servie")
		// Requeue reconciliation as we were unable to fetch the object
		return ctrl.Result{}, err
	}

	// Fetch the Deployment object if it exists
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: simple.Name, Namespace: simple.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.deploymentForSimple(simple)
		logger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure deployment replicas is the same as the Simple Service size
	size := simple.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Ask to requeue after 1 minute in order to give enough time for the
		// pods be created on the cluster side and the operand be able
		// to do the next update step accurately.
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Fetch pods to get their names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(simple.Namespace),
		client.MatchingLabels(labelsForSimple(simple.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "Failed to list pods", "Simple.Namespace", simple.Namespace, "Simple.Name", simple.Name)
		return ctrl.Result{}, err
	}

	// Update Simple Service nodes with pod names
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, simple.Status.Nodes) {
		simple.Status.Nodes = podNames
		err := r.Status().Update(ctx, simple)
		if err != nil {
			logger.Error(err, "Failed to update Simple Service status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&simplev1alpha1.Simple{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *SimpleReconciler) deploymentForSimple(m *simplev1alpha1.Simple) *appsv1.Deployment {
	ls := labelsForSimple(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "biandayu/simple-service:v1.2.1",
						Name:  "simple",
						Env: []corev1.EnvVar{{
							Name:  "UPSTREAM",
							Value: "https://www.thecocktaildb.com/",
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "http1",
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep

}

func labelsForSimple(name string) map[string]string {
	return map[string]string{"app": "simple", "simple_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

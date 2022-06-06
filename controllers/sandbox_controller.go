/*
Copyright 2022.

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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	mydomainv1alpha1 "github.com/MahnoorAsghar/Sandbox-Operator/api/v1alpha1"
)

// SandboxReconciler reconciles a Sandbox object
type SandboxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.domain,resources=sandboxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.domain,resources=sandboxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=sandboxes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Sandbox object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *SandboxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := ctrllog.FromContext(ctx)

	log.Info("SandBox Reconciler called")

	// Fetch sandbox object
	sandbox := &mydomainv1alpha1.Sandbox{}
	err := r.Get(ctx, req.NamespacedName, sandbox)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request
			return ctrl.Result{Requeue: false}, err
		}
		return ctrl.Result{}, err
	}

	// If sandbox is marked for deletion, delete its namespace
	if sandbox.GetDeletionTimestamp() != nil {
		log.Info("Deletion timestamp found for sandbox")
		r.deleteNamespace(req, ctx, sandbox)
		return ctrl.Result{Requeue: false}, err
	}

	// Update the Sandbox Status field
	if sandbox.Status.Name == "" {
		sandbox.Status.Name = sandbox.Spec.Name

		err = r.Status().Update(ctx, sandbox)
		if err != nil {
			log.Info("Sandbox status could not be updated.")
			return ctrl.Result{Requeue: false}, err
		}

		r.createNamespace(ctx, req, sandbox)
	}

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) createNamespace(ctx context.Context, req ctrl.Request, sandbox *mydomainv1alpha1.Sandbox) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Creating Namespace")

	// Get the sandbox resource for which the namespace is to be created
	err := r.Get(ctx, req.NamespacedName, sandbox)
	if err != nil {
		log.Info("Cannot get sandbox")
	}

	// Construct a namespace object
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s%s", "ns-", strings.ToLower(sandbox.Spec.Name)),
		},
	}
	log.WithValues("Namespace", nsSpec)

	// Create the namespace
	err = client.Client.Create(r.Client, ctx, nsSpec)

	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Info("Namespace already exists")
		} else {
			log.Info("Can not create new namespace")
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SandboxReconciler) deleteNamespace(req ctrl.Request, ctx context.Context, sandBox *mydomainv1alpha1.Sandbox) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Deleting Namespace")

	ns := &corev1.Namespace{}

	log.WithValues("Logging for namespace", ns)

	// Fetch the namespace to be deleted
	err := r.Client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s%s", "ns-", strings.ToLower(sandBox.Spec.Name))}, ns)
	if err != nil {
		log.Info("Namespace not available")
		return ctrl.Result{}, err
	}

	// Delete the namespace
	err = r.Client.Delete(ctx, ns)
	if err != nil {
		log.Info("Unable to delete the namespace: ", ns.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SandboxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.Sandbox{}).
		Complete(r)
}

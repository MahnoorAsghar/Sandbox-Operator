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
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	mydomainv1alpha1 "github.com/MahnoorAsghar/Sandbox-Operator/api/v1alpha1"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=my.domain,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.domain,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.domain,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// Fetch the user instance
	var user = &mydomainv1alpha1.User{}
	err := r.Get(ctx, req.NamespacedName, user)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("User resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}

	if user.GetDeletionTimestamp() != nil {
		log.Info("Deletion timestamp found for user " + req.Name)
		return r.deleteSandbox(ctx, user, req)
	}

	// If Status field is empty, add the Status info
	if user.Status.Name == "" {
		user.Status.Name = user.Spec.Name
		user.Status.SandBoxCount = user.Spec.SandBoxCount

		err = r.Status().Update(ctx, user)
		if err != nil {
			log.Info("Unable to update User Status.", "User name", user.Name)
		}
		r.createSandboxes(ctx, user)
	} else if user.Spec.Name != user.Status.Name || user.Spec.SandBoxCount != user.Status.SandBoxCount {
		r.updateUser(ctx, user, req)
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) createNamespace(ctx context.Context, sandboxName string) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Creating Namespace")

	// Construct a namespace object
	nsSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s%s", "ns-", sandboxName),
		},
	}
	log.WithValues("Namespace", nsSpec)

	// Create the namespace
	err := client.Client.Create(r.Client, ctx, nsSpec)

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

func (r *UserReconciler) createSandboxes(ctx context.Context, user *mydomainv1alpha1.User) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Creating Sandbox(es).", "User", user.Spec.Name)

	var initial int = user.Status.SandBoxCount

	for i := initial; i < user.Spec.SandBoxCount; i++ {

		sandboxName := fmt.Sprintf("%s%s%s%d", "sb-", strings.ToLower(user.Spec.Name), "-", i)
		result, err := r.createNamespace(ctx, sandboxName)

		if err != nil {
			fmt.Println("result: ", result)
			return ctrl.Result{}, err
		}

		sandbox := &mydomainv1alpha1.Sandbox{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sandboxName,
				Namespace: fmt.Sprintf("%s%s", "ns-", sandboxName),
			},

			Spec: mydomainv1alpha1.SandboxSpec{
				Name: fmt.Sprintf("%s%s%s%d", "SB-", user.Spec.Name, "-", i),
				Type: "T1",
			},
		}

		err = r.Client.Create(ctx, sandbox)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				log.Info("Sandbox already exists")
			} else {
				log.Info("Error occured while creating sandbox")
			}
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *UserReconciler) updateUser(ctx context.Context, user *mydomainv1alpha1.User, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Updating user.", "User", user.Spec.Name)

	if user.Spec.Name != user.Status.Name {
		log.Info("User name can not be updated.")
	}

	// Change in SandBoxCount
	if user.Spec.SandBoxCount < user.Status.SandBoxCount {
		log.Info("Can not decrease the SandBoxCount!")
	} else if user.Spec.SandBoxCount > user.Status.SandBoxCount {
		log.Info("Creating new sandboxes.")
		r.createSandboxes(ctx, user)
		user.Status.SandBoxCount = user.Spec.SandBoxCount
	}

	// Update the Status field of the User
	r.Status().Update(ctx, user)

	// Try to get the sandbox. if it fails, throw an error.
	sandbox := &mydomainv1alpha1.Sandbox{}
	err := r.Get(ctx, req.NamespacedName, sandbox)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Done. Log and exit!
	log.Info("Updated user details.", "User: ", user.Spec.Name)
	return ctrl.Result{}, nil
}

func (r *UserReconciler) deleteSandbox(ctx context.Context, user *mydomainv1alpha1.User, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Deleting sandboxes for user")

	sandBox := &mydomainv1alpha1.Sandbox{}
	userName := user.Spec.Name

	for i := 1; i <= user.Spec.SandBoxCount; i++ {
		err := r.Client.Get(ctx,
			types.NamespacedName{Namespace: req.Namespace,
				Name: fmt.Sprintf("%s%s%s%d", "sb-", strings.ToLower(user.Spec.Name), "-", i)},
			sandBox)
		if err != nil {
			log.Info("No sandbox to delete for this user", "User: ", user.Spec.Name)
			return ctrl.Result{}, err

		}
		if strings.Contains(sandBox.Spec.Name, userName) {
			r.Client.Delete(ctx, sandBox)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mydomainv1alpha1.User{}).
		Complete(r)
}

/*func userPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			userObj, err := e.Object.(*mydomainv1alpha1.User)
			sandBoxCount := (*userObj).Spec.SandBoxCount
			fmt.Printf("count: %v\n\n", sandBoxCount)
			fmt.Printf("err: %v\n", err)
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
	}
}*/

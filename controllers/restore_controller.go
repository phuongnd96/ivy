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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ivyv1alpha1 "github.com/phuongnd96/ivy/api/v1alpha1"
	"github.com/phuongnd96/ivy/pkg/k8s"
	"github.com/phuongnd96/ivy/pkg/sql"
	"github.com/spf13/viper"
)

// RestoreReconciler reconciles a Restore object
type RestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ivy.dev,resources=restores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ivy.dev,resources=restores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ivy.dev,resources=restores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Restore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	defer log.Info("End reconcile")
	var err error
	var backUpId string
	viper.SetConfigFile("config.yaml")
	viper.ReadInConfig()
	foundCR := &ivyv1alpha1.Restore{}
	err = r.Get(ctx, req.NamespacedName, foundCR)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			log.Info("Restore not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Restore")
		return ctrl.Result{}, err
	}
	remoteRestConfig, err := k8s.GetGKEClientsetWithTimeout(foundCR.Spec.TargetProjectId, foundCR.Spec.TargetCluster, viper.GetString("Region"))
	if err != nil {
		log.Error(err, "Get remoteRestConfig")
	}
	remoteClientSet, err := kubernetes.NewForConfig(remoteRestConfig)
	if err != nil {
		log.Error(err, "Get remoteClientSet")
	}
	if foundCR.Spec.BackupId != "" {
		backUpId = foundCR.Spec.BackupId
	} else {
		backUpId = foundCR.Name
	}
	if err = sql.InstanceFromBackUp(ctx, remoteClientSet, foundCR.Name, foundCR.Spec.TargetNamespace, "8.0.30", backUpId, "sql-from-backup@dev-org.iam.gserviceaccount.com"); err != nil {
		log.Error(err, "Boost instance from backup")
	}
	const finalizer = "restore.ivy.dev/finalizer"
	// Check if the Restore instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isRestoreMarkedToBeDeleted := foundCR.GetDeletionTimestamp() != nil
	if isRestoreMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(foundCR, finalizer) {
			// Run finalization logic for finalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeSQLRestore(ctx, remoteClientSet, foundCR.Spec.TargetNamespace, foundCR.Name); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(foundCR, finalizer)
			err := r.Update(ctx, foundCR)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(foundCR, finalizer) {
		controllerutil.AddFinalizer(foundCR, finalizer)
		err = r.Update(ctx, foundCR)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ivyv1alpha1.Restore{}).
		Complete(r)
}

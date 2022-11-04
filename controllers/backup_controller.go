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

	"google.golang.org/api/storage/v1"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/sqladmin/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ivyv1alpha1 "github.com/phuongnd96/ivy/api/v1alpha1"
	"github.com/phuongnd96/ivy/pkg/sql"
	"github.com/spf13/viper"
)

// BackupReconciler reconciles a Backup object
type BackupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=ivy.dev,resources=backups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ivy.dev,resources=backups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ivy.dev,resources=backups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Backup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	defer log.Info("End reconcile")
	var err error
	viper.SetConfigFile("config.yaml")
	viper.ReadInConfig()
	foundCR := &ivyv1alpha1.Backup{}
	err = r.Get(ctx, req.NamespacedName, foundCR)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			log.Info("Backup not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Backup")
		return ctrl.Result{}, err
	}
	if foundCR.Status.ObservedGeneration < foundCR.GetObjectMeta().GetGeneration() {
		c, err := google.DefaultClient(ctx, sqladmin.CloudPlatformScope, storage.CloudPlatformScope)
		if err != nil {
			log.Error(err, "Authenticate to GCP")
			foundCR.Status.Status = viper.GetString("BackUpFailedStatus")
			if err = r.Status().Update(ctx, foundCR, &client.UpdateOptions{}); err != nil {
				log.Error(err, "Update backup resource status")
			}
			return ctrl.Result{}, err
		}
		// if foundCR.Status.Status != viper.GetString("BackUpSuccessStatus") {
		// 	if err = sql.NewBackUp(ctx, foundCR.Spec.ProjectID, foundCR.Spec.SQLSourceInstance, c); err != nil {
		// 		log.Error(err, "NewBackUp")
		// 		foundCR.Status.Status = viper.GetString("BackUpFailedStatus")
		// 		if err = r.Status().Update(ctx, foundCR, &client.UpdateOptions{}); err != nil {
		// 			log.Error(err, "Update backup resource status")
		// 		}
		// 		return ctrl.Result{}, err
		// 	}
		// }
		if foundCR.Status.Status != viper.GetString("BackUpSuccessStatus") {
			if err = sql.ExportToGCS(ctx, foundCR.Spec.ProjectID, foundCR.Spec.SQLSourceInstance, c, foundCR.Spec.ExcludedDatabases, foundCR.Spec.GCSBucket, foundCR.Name); err != nil {
				log.Error(err, "Export Backup to GCS")
				return ctrl.Result{}, err
			}
		}
		if err = sql.WaitForInstanceReady(foundCR.Spec.ProjectID, foundCR.Spec.SQLSourceInstance, c); err != nil {
			log.Error(err, "WaitForInstanceReady")
			foundCR.Status.Status = viper.GetString("BackUpFailedStatus")
			if err = r.Status().Update(ctx, foundCR, &client.UpdateOptions{}); err != nil {
				log.Error(err, "Update backup resource status")
			}
			return ctrl.Result{}, err
		}
		foundCR.Status.Status = viper.GetString("BackUpSuccessStatus")
		if err = r.Status().Update(ctx, foundCR, &client.UpdateOptions{}); err != nil {
			log.Error(err, "Update backup resource status")
			return ctrl.Result{}, err
		}
	}
	// observed generate < universe spec generate on cluster, update
	foundCR.Status.ObservedGeneration = foundCR.GetObjectMeta().GetGeneration()
	r.Status().Update(ctx, foundCR, &client.UpdateOptions{})
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ivyv1alpha1.Backup{}).
		Complete(r)
}

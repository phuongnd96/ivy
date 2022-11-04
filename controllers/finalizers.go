package controllers

import (
	"context"
	"net/http"

	"github.com/phuongnd96/ivy/pkg/bigtable"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

func (r *RestoreReconciler) finalizeSQLRestore(ctx context.Context, clientset kubernetes.Interface, namespace string, name string) error {
	var err error
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err = clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	_, err = clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err = clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.Info("Successfully finalized Restore")
	return nil
}

func (r *BigTableBackUpReconciler) finalizeBigTableBackup(ctx context.Context, project string, instance string, cluster string, sourceTableId string, backupId string, client *http.Client) error {
	if err := bigtable.CleanBackup(ctx, project, instance, cluster, backupId, client); err != nil {
		return err
	}
	return nil
}

func (r *BigTableRestoreReconciler) finalizeBigTableRestore(ctx context.Context, project string, instance string, cluster string, tableId string, client *http.Client) error {
	table := bigtable.NewBigtable(project, instance, cluster, tableId)
	if err := bigtable.CleanTableRestore(ctx, *table, client); err != nil {
		return err
	}
	return nil
}

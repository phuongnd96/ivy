package sql

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/phuongnd96/ivy/helper/poll"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/strings/slices"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func InstanceFromBackUp(ctx context.Context, clientset kubernetes.Interface, name string, namespace string, version string, backupId string, backupServiceAccountEmail string) error {
	sqlInstanceK8sSa := "sql-from-backup"
	var err error
	_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		if _, err = clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, sqlInstanceK8sSa, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		serviceAccount := &v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sqlInstanceK8sSa,
				Namespace: namespace,
				Annotations: map[string]string{
					"iam.gke.io/gcp-service-account": backupServiceAccountEmail,
				},
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	_, err = clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Port: 3306,
					},
				},
				Selector: map[string]string{
					"app":  "mysql",
					"name": name,
				},
			},
		}
		_, err = clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"storage": resource.MustParse("250Gi"),
					},
				},
			},
		}
		_, err = clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	downloadBackUpCommand := fmt.Sprintf(`gsutil cp gs://%s /var`, backupId)
	object := backupId[strings.LastIndex(backupId, "/")+1:]
	restoreCommand := fmt.Sprintf(`cd /var && gunzip -c %s > /docker-entrypoint-initdb.d/restore.sql`, object)
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":  "mysql",
						"name": name,
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: "Recreate",
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":  "mysql",
							"name": name,
						},
					},
					Spec: v1.PodSpec{
						ServiceAccountName: sqlInstanceK8sSa,
						InitContainers: []v1.Container{
							{
								Name:  "init",
								Image: "google/cloud-sdk:404.0.0",
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "data",
										MountPath: "/var",
									},
								},
								Command: []string{
									"sh", "-c", downloadBackUpCommand,
								},
							},
						},
						Containers: []v1.Container{
							{
								Name:  "mysql",
								Image: "mysql:" + version,
								Lifecycle: &v1.Lifecycle{
									PostStart: &v1.LifecycleHandler{
										Exec: &v1.ExecAction{
											Command: []string{"sh", "-c", restoreCommand},
										},
									},
								},
								Env: []v1.EnvVar{
									{
										Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
										Value: "true",
									},
								},
								Ports: []v1.ContainerPort{
									{
										ContainerPort: 3306,
										Name:          "mysql",
									},
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "data",
										MountPath: "/var",
									},
								},
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "data",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: name,
									},
								},
							},
						},
					},
				},
			},
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func NewBackUp(ctx context.Context, projectId string, instanceId string, client *http.Client) error {
	// sqladminService, err := sqladmin.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, authToken)))
	sqladminService, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	_, err = sqladminService.BackupRuns.Insert(projectId, instanceId, &sqladmin.BackupRun{}).Do()
	if err != nil {
		return err
	}
	return nil
}

func ExportToGCS(ctx context.Context, projectId string, instanceId string, client *http.Client, excludedDatabases []string, bucketName string, objectName string) error {
	var dbExportList []string
	sqladminService, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	resp, err := sqladminService.Databases.List(projectId, instanceId).Do()
	if err != nil {
		return err
	}
	for _, db := range resp.Items {
		if !slices.Contains(excludedDatabases, db.Name) {
			dbExportList = append(dbExportList, db.Name)
		}
	}
	exportUri := fmt.Sprintf("gs://%s/%s.sql.gz", bucketName, objectName)
	_, err = sqladminService.Instances.Export(projectId, instanceId, &sqladmin.InstancesExportRequest{
		ExportContext: &sqladmin.ExportContext{
			FileType:  "SQL",
			Databases: dbExportList,
			Uri:       exportUri,
		},
	}).Context(ctx).Do()
	if err != nil {
		return err
	}
	return nil
}

func WaitForInstanceReady(projectId string, instanceId string, client *http.Client) error {
	viper.SetConfigFile("config.yaml")
	viper.ReadInConfig()
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("GlobalResourceTimeOut")*time.Minute)
	defer cancel()
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		log.WithFields(log.Fields{
			"instanceId": instanceId,
		}).Info("Waiting for instance ready")
		sqladminService, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return false, err
		}
		b, _ := sqladminService.BackupRuns.List(projectId, instanceId).Do()
		log.WithFields(log.Fields{
			"status": b.Items[0].Status,
		}).Info("Last backup")
		if b.Items[0].Status == viper.GetString("BackUpSuccessStatus") {
			return true, nil
		}
		return false, nil
	})
}

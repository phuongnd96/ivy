package gcs

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// func GetLatestObjectWithPrefix(ctx context.Context, prefix string, bucket string) (*storage.Object, error) {
// 	t, _ := auth.GetToken(ctx)
// 	storageService, err := storage.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
// 	if err != nil {
// 		return nil, err
// 	}
// 	_, err = storageService.Buckets.Get(bucket).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	objList, err := storageService.Objects.List(bucket).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var objListWithFilter *storage.Objects
// 	regex, _ := regexp.Compile("^" + prefix + ".+")
// 	for _, obj := range objList.Items {
// 		if regex.MatchString(obj.Name) {
// 			objListWithFilter.Items = append(objListWithFilter.Items, obj)
// 		}
// 	}
// 	sort.Slice(objListWithFilter.Items[:], func(i, j int) bool {
// 		return objListWithFilter.Items[i].Updated < objListWithFilter.Items[j].Updated
// 	})
// 	return objListWithFilter.Items[len(objListWithFilter.Items)], nil
// }

func NewBucketWithServiceAccount(ctx context.Context, bucket string, clientset kubernetes.Interface, restConfig *rest.Config, name string, namespace string, scheme *runtime.Scheme, owner metav1.Object) error {
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "storage.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "storagebuckets",
	}
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		b := &unstructured.Unstructured{}
		b.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "storage.cnrm.cloud.google.com/v1beta1",
			"kind":       "StorageBucket",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"annotations": map[string]interface{}{
					"cnrm.cloud.google.com/force-destroy": "true",
				},
			},
			"spec": map[string]interface{}{
				"lifecycleRule": []string{},
				"location":      viper.GetString("Region"),
			},
		})
		err = ctrl.SetControllerReference(owner, b, scheme)
		if err != nil {
			return err
		}
		_, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, b, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	gvr = schema.GroupVersionResource{
		Group:    "iam.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "iamserviceaccounts",
	}
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		b := &unstructured.Unstructured{}
		b.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "iam.cnrm.cloud.google.com/v1beta1",
			"kind":       "IAMServiceAccount",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"displayName": "Service account to access bucket" + name,
			},
		})
		err = ctrl.SetControllerReference(owner, b, scheme)
		if err != nil {
			return err
		}
		_, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, b, metav1.CreateOptions{})
		if err != nil {
			return err
		}

	}
	gvr = schema.GroupVersionResource{
		Group:    "storage.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "storagebucketaccesscontrols",
	}
	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Create storagebucketaccesscontrols")
		b := &unstructured.Unstructured{}
		b.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "storage.cnrm.cloud.google.com/v1beta1",
			"kind":       "StorageBucketAccessControl",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"bucketRef": map[string]interface{}{
					"name": name,
				},
				"entity": "user-" + name + "@" + viper.GetString("GCP_PROJECT_ID"),
				"role":   "OWNER",
			},
		})
		err = ctrl.SetControllerReference(owner, b, scheme)
		if err != nil {
			log.Error(err, "Set owner ref")
		}
		_, err := dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, b, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

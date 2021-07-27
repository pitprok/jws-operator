package webserver

import (
	"context"
	"fmt"
	"sort"

	webserversv1alpha1 "github.com/web-servers/jws-operator/pkg/apis/webservers/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	// rbac "rbac.authorization.k8s.io/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func isOpenShift(c *rest.Config) bool {
	var err error
	var dcclient *discovery.DiscoveryClient
	dcclient, err = discovery.NewDiscoveryClientForConfig(c)
	if err != nil {
		log.Info("isOpenShift discovery.NewDiscoveryClientForConfig has encountered a problem")
		return false
	}
	apiList, err := dcclient.ServerGroups()
	if err != nil {
		log.Info("isOpenShift client.ServerGroups has encountered a problem")
		return false
	}
	for _, v := range apiList.Groups {
		log.Info(v.Name)
		if v.Name == "route.openshift.io" {

			log.Info("route.openshift.io was found in apis, platform is OpenShift")
			return true
		}
	}
	return false
}

func (r *ReconcileWebServer) setDefaultValues(webServer *webserversv1alpha1.WebServer) *webserversv1alpha1.WebServer {

	if webServer.Spec.WebImage != nil && webServer.Spec.WebImage.WebApp != nil {
		webApp := webServer.Spec.WebImage.WebApp
		if webApp.Name == "" {
			log.Info("WebServer.Spec.Image.WebApp.Name is not set, setting value to 'ROOT'")
			webApp.Name = "ROOT"
		}
		if webApp.DeployPath == "" {
			log.Info("WebServer.Spec.Image.WebApp.DeployPath is not set, setting value to '/deployments/'")
			webApp.DeployPath = "/deployments/"
		}
		if webApp.ApplicationSizeLimit == "" {
			log.Info("WebServer.Spec.Image.WebApp.ApplicationSizeLimit is not set, setting value to '1Gi'")
			webApp.ApplicationSizeLimit = "1Gi"
		}

		if webApp.Builder.ApplicationBuildScript == "" {
			log.Info("WebServer.Spec.Image.WebApp.Builder.ApplicationBuildScript is not set, generating default build script")
			webApp.Builder.ApplicationBuildScript = generateWebAppBuildScript(webServer)
		}
	}

	return webServer

}

func generateWebAppBuildScript(webServer *webserversv1alpha1.WebServer) string {
	webApp := webServer.Spec.WebImage.WebApp
	webAppWarFileName := webApp.Name + ".war"
	webAppSourceRepositoryURL := webApp.SourceRepositoryURL
	webAppSourceRepositoryRef := webApp.SourceRepositoryRef
	webAppSourceRepositoryContextDir := webApp.SourceRepositoryContextDir

	return fmt.Sprintf(`
		webAppWarFileName=%s;
		webAppSourceRepositoryURL=%s;
		webAppSourceRepositoryRef=%s;
		webAppSourceRepositoryContextDir=%s;

		# Some pods don't have root privileges, so the build takes place in /tmp
		cd tmp;

		# Create a custom .m2 repo in a location where no root privileges are required
		mkdir -p /tmp/.m2/repo;

		# Create custom maven settings that change the location of the .m2 repo
		echo '<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"' >> /tmp/.m2/settings.xml
		echo 'xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.0.0 https://maven.apache.org/xsd/settings-1.0.0.xsd">' >> /tmp/.m2/settings.xml
		echo '<localRepository>/tmp/.m2/repo</localRepository>' >> /tmp/.m2/settings.xml
		echo '</settings>' >> /tmp/.m2/settings.xml

		if [ -z ${webAppSourceRepositoryURL} ]; then
			echo "Need an URL like https://github.com/jfclere/demo-webapp.git";
			exit 1;
		fi;

		git clone ${webAppSourceRepositoryURL};
		if [ $? -ne 0 ]; then
			echo "Can't clone ${webAppSourceRepositoryURL}";
			exit 1;
		fi;

		# Get the name of the source code directory
		DIR=$(echo ${webAppSourceRepositoryURL##*/});
		DIR=$(echo ${DIR%%.*});

		cd ${DIR};

		if [ ! -z ${webAppSourceRepositoryRef} ]; then
			git checkout ${webAppSourceRepositoryRef};
		fi;

		if [ ! -z ${webAppSourceRepositoryContextDir} ]; then
			cd ${webAppSourceRepositoryContextDir};
		fi;

		# Builds the webapp using the custom maven settings
		mvn clean install -gs /tmp/.m2/settings.xml;
		if [ $? -ne 0 ]; then
			echo "mvn install failed please check the pom.xml in ${webAppSourceRepositoryURL}";
			exit 1;
		fi

		# Copies the resulting war to the mounted persistent volume
		cp target/*.war /mnt/${webAppWarFileName};`,
		webAppWarFileName,
		webAppSourceRepositoryURL,
		webAppSourceRepositoryRef,
		webAppSourceRepositoryContextDir,
	)
}

func (r *ReconcileWebServer) getWebServer(request reconcile.Request) (*webserversv1alpha1.WebServer, error) {
	webServer := &webserversv1alpha1.WebServer{}
	err := r.client.Get(context.TODO(), request.NamespacedName, webServer)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("WebServer resource not found. Ignoring since object must have been deleted")
			return webServer, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get WebServer resource")
		return webServer, err
	}
	return webServer, nil
}

func (r *ReconcileWebServer) createResource(webServer *webserversv1alpha1.WebServer, resource runtime.Object, resourceKind string, resourceName string, resourceNamespace string) (ctrl.Result, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: resourceName, Namespace: resourceNamespace}, resource)
	if err != nil && errors.IsNotFound(err) {
		// Create a new resource
		log.Info("Creating a new "+resourceKind, resourceKind+".Namespace", resourceNamespace, resourceKind+".Name", resourceName)
		err = r.client.Create(context.TODO(), resource)
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create a new "+resourceKind, resourceKind+".Namespace", resourceNamespace, resourceKind+".Name", resourceName)
			return reconcile.Result{}, err
		}
		// Resource created successfully - return and requeue
		return ctrl.Result{Requeue: true}, err
	} else if err != nil {
		log.Error(err, "Failed to get "+resourceKind)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, err
}

// updateWebServerStatus updates status of the WebServer resource.
func updateWebServerStatus(webServer *webserversv1alpha1.WebServer, client client.Client) error {
	log.Info("Updating the status of WebServer")

	if err := client.Status().Update(context.Background(), webServer); err != nil {
		log.Error(err, "Failed to update the status of WebServer")
		return err
	}

	log.Info("The status of WebServer was updated successfully")
	return nil
}

// sortPodListByName sorts the pod list by number in the name
//  expecting the format which the StatefulSet works with which is `<podname>-<number>`
func sortPodListByName(podList *corev1.PodList) *corev1.PodList {
	sort.SliceStable(podList.Items, func(i, j int) bool {
		return podList.Items[i].ObjectMeta.Name < podList.Items[j].ObjectMeta.Name
	})
	return podList
}

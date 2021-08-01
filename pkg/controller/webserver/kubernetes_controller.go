package webserver

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	webserversv1alpha1 "github.com/web-servers/jws-operator/pkg/apis/webservers/v1alpha1"

	kbappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbac "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func generateObjectMeta(webServer *webserversv1alpha1.WebServer, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: webServer.Namespace,
		Labels: map[string]string{
			"application": webServer.Spec.ApplicationName,
		},
	}
}

func (r *ReconcileWebServer) generateRoutingService(webServer *webserversv1alpha1.WebServer) *corev1.Service {

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Service",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "ui",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
			Selector: map[string]string{
				"deploymentConfig": webServer.Spec.ApplicationName,
				"WebServer":        webServer.Name,
			},
		},
	}

	controllerutil.SetControllerReference(webServer, service, r.scheme)
	return service
}

func (r *ReconcileWebServer) generateServiceForDNS(webServer *webserversv1alpha1.WebServer) *corev1.Service {

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Service",
		},
		ObjectMeta: generateObjectMeta(webServer, "webserver-"+webServer.Name),
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
			Selector: map[string]string{
				"application": webServer.Spec.ApplicationName,
			},
		},
	}

	controllerutil.SetControllerReference(webServer, service, r.scheme)
	return service
}

func (r *ReconcileWebServer) generateRoleBinding(webServer *webserversv1alpha1.WebServer) *rbac.RoleBinding {
	rolebinding := &rbac.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1beta",
			Kind:       "RoleBinding",
		},
		ObjectMeta: generateObjectMeta(webServer, "webserver-"+webServer.Name),
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "view",
		},
		Subjects: []rbac.Subject{{
			Kind: "ServiceAccount",
			Name: "default",
		}},
	}

	controllerutil.SetControllerReference(webServer, rolebinding, r.scheme)
	return rolebinding
}

func (r *ReconcileWebServer) generateConfigMapForDNS(webServer *webserversv1alpha1.WebServer) *corev1.ConfigMap {

	cmap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: generateObjectMeta(webServer, "webserver-"+webServer.Name),
		Data:       r.generateCommandForServerXml(),
	}

	controllerutil.SetControllerReference(webServer, cmap, r.scheme)
	return cmap
}

func (r *ReconcileWebServer) generatePersistentVolumeClaim(webServer *webserversv1alpha1.WebServer) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s.io/api/apps/v1",
			Kind:       "PersistentVolumeClaimVolumeSource",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse(webServer.Spec.WebImage.WebApp.ApplicationSizeLimit),
				},
			},
		},
	}

	controllerutil.SetControllerReference(webServer, pvc, r.scheme)
	return pvc
}

func (r *ReconcileWebServer) generateBuildPod(webServer *webserversv1alpha1.WebServer) *corev1.Pod {
	name := webServer.Spec.ApplicationName + "-build"
	objectMeta := generateObjectMeta(webServer, name)
	objectMeta.Labels["WebServer"] = webServer.Name
	terminationGracePeriodSeconds := int64(60)
	pod := &corev1.Pod{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			RestartPolicy:                 "OnFailure",
			Volumes: []corev1.Volume{
				{
					Name: "app-volume",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: webServer.Spec.ApplicationName},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "war",
					Image: webServer.Spec.WebImage.WebApp.Builder.Image,
					Command: []string{
						"/bin/sh",
						"-c",
					},
					Args: []string{
						webServer.Spec.WebImage.WebApp.Builder.ApplicationBuildScript,
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "app-volume",
							MountPath: "/mnt",
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(webServer, pod, r.scheme)
	return pod
}

func (r *ReconcileWebServer) generateDeployment(webServer *webserversv1alpha1.WebServer) *kbappsv1.Deployment {

	replicas := int32(1)
	podTemplateSpec := r.generatePodTemplate(webServer, webServer.Spec.WebImage.ApplicationImage)
	deployment := &kbappsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s.io/api/apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
		Spec: kbappsv1.DeploymentSpec{
			Strategy: kbappsv1.DeploymentStrategy{
				Type: kbappsv1.RecreateDeploymentStrategyType,
			},
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deploymentConfig": webServer.Spec.ApplicationName,
					"WebServer":        webServer.Name,
				},
			},
			Template: podTemplateSpec,
		},
	}

	controllerutil.SetControllerReference(webServer, deployment, r.scheme)
	return deployment
}

func (r *ReconcileWebServer) generatePodTemplate(webServer *webserversv1alpha1.WebServer, image string) corev1.PodTemplateSpec {
	objectMeta := generateObjectMeta(webServer, webServer.Spec.ApplicationName)
	objectMeta.Labels["deploymentConfig"] = webServer.Spec.ApplicationName
	objectMeta.Labels["WebServer"] = webServer.Name
	var health *webserversv1alpha1.WebServerHealthCheckSpec = &webserversv1alpha1.WebServerHealthCheckSpec{}
	if webServer.Spec.WebImage != nil {
		health = webServer.Spec.WebImage.WebServerHealthCheck
	} else {
		health = webServer.Spec.WebImageStream.WebServerHealthCheck
	}
	terminationGracePeriodSeconds := int64(60)
	return corev1.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			Containers: []corev1.Container{{
				Name:            webServer.Spec.ApplicationName,
				Image:           image,
				ImagePullPolicy: "Always",
				ReadinessProbe:  generateReadinessProbe(webServer, health),
				LivenessProbe:   generateLivenessProbe(webServer, health),
				Ports: []corev1.ContainerPort{{
					Name:          "jolokia",
					ContainerPort: 8778,
					Protocol:      corev1.ProtocolTCP,
				}, {
					Name:          "http",
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				}},
				Env:          r.generateEnvVars(webServer),
				VolumeMounts: generateVolumeMounts(webServer),
			}},
			Volumes: generateVolumes(webServer),
		},
	}
}

// generateLivenessProbe returns a custom probe if the serverLivenessScript string is defined and not empty in the Custom Resource.
// Otherwise, it uses the default /health Valve via curl.
//
// If defined, serverLivenessScript must be a shell script that
// complies to the Kubernetes probes requirements and use the following format
// shell -c "command"
func generateLivenessProbe(webServer *webserversv1alpha1.WebServer, health *webserversv1alpha1.WebServerHealthCheckSpec) *corev1.Probe {
	livenessProbeScript := ""
	if health != nil {
		livenessProbeScript = health.ServerLivenessScript
	}
	if livenessProbeScript != "" {
		return generateCustomProbe(webServer, livenessProbeScript)
	} else {
		/* Use the default one */
		return &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(8080),
				},
			},
		}
	}
}

// generateReadinessProbe returns a custom probe if the serverReadinessScript string is defined and not empty in the Custom Resource.
// Otherwise, it uses the default /health Valve via curl.
//
// If defined, serverReadinessScript must be a shell script that
// complies to the Kubernetes probes requirements and use the following format
// shell -c "command"
func generateReadinessProbe(webServer *webserversv1alpha1.WebServer, health *webserversv1alpha1.WebServerHealthCheckSpec) *corev1.Probe {
	readinessProbeScript := ""
	if health != nil {
		readinessProbeScript = health.ServerReadinessScript
	}
	if readinessProbeScript != "" {
		return generateCustomProbe(webServer, readinessProbeScript)
	} else {
		/* Use the default one */
		return &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(8080),
				},
			},
		}
	}
}

func generateCustomProbe(webServer *webserversv1alpha1.WebServer, probeScript string) *corev1.Probe {
	// If the script has the following format: shell -c "command"
	// we create the slice ["shell", "-c", "command"]
	probeScriptSlice := make([]string, 0)
	pos := strings.Index(probeScript, "\"")
	if pos != -1 {
		probeScriptSlice = append(strings.Split(probeScript[0:pos], " "), probeScript[pos:])
	} else {
		probeScriptSlice = strings.Split(probeScript, " ")
	}
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: probeScriptSlice,
			},
		},
	}
}

// Create the env for the pods we are starting.
func (r *ReconcileWebServer) generateEnvVars(webServer *webserversv1alpha1.WebServer) []corev1.EnvVar {
	value := "webserver-" + webServer.Name
	if r.useKUBEPing && webServer.Spec.UseSessionClustering {
		value = webServer.Namespace
	}
	env := []corev1.EnvVar{
		{
			Name:  "KUBERNETES_NAMESPACE",
			Value: value,
		},
	}
	if webServer.Spec.UseSessionClustering {
		// Add parameter USE_SESSION_CLUSTERING
		env = append(env, corev1.EnvVar{
			Name:  "ENV_FILES",
			Value: "/test/my-files/test.sh",
		})
	}
	return env
}

// Create the VolumeMounts
func generateVolumeMounts(webServer *webserversv1alpha1.WebServer) []corev1.VolumeMount {
	var volm []corev1.VolumeMount
	if webServer.Spec.UseSessionClustering {
		volm = append(volm, corev1.VolumeMount{
			Name:      "webserver-" + webServer.Name,
			MountPath: "/test/my-files",
		})
	}
	if webServer.Spec.WebImage != nil && webServer.Spec.WebImage.WebApp != nil {
		webAppWarFileName := webServer.Spec.WebImage.WebApp.Name + ".war"
		volm = append(volm, corev1.VolumeMount{
			Name:      "app-volume",
			MountPath: webServer.Spec.WebImage.WebApp.DeployPath + webAppWarFileName,
			SubPath:   webAppWarFileName,
		})
	}
	return volm
}

// Create the Volumes
func generateVolumes(webServer *webserversv1alpha1.WebServer) []corev1.Volume {
	var vol []corev1.Volume
	if webServer.Spec.UseSessionClustering {
		vol = append(vol, corev1.Volume{
			Name: "webserver-" + webServer.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "webserver-" + webServer.Name,
					},
				},
			},
		})
	}
	if webServer.Spec.WebImage != nil && webServer.Spec.WebImage.WebApp != nil {
		vol = append(vol, corev1.Volume{
			Name: "app-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: webServer.Spec.ApplicationName,
					ReadOnly:  true,
				},
			},
		})
	}
	return vol
}

// create the shell script to modify server.xml
//
func (r *ReconcileWebServer) generateCommandForServerXml() map[string]string {
	cmd := make(map[string]string)
	if r.useKUBEPing {
		cmd["test.sh"] = "FILE=`find /opt -name server.xml`\n" +
			"grep -q MembershipProvider ${FILE}\n" +
			"if [ $? -ne 0 ]; then\n" +
			"  sed -i '/cluster.html/a        <Cluster className=\"org.apache.catalina.ha.tcp.SimpleTcpCluster\" channelSendOptions=\"6\">\\n <Channel className=\"org.apache.catalina.tribes.group.GroupChannel\">\\n <Membership className=\"org.apache.catalina.tribes.membership.cloud.CloudMembershipService\" membershipProviderClassName=\"org.apache.catalina.tribes.membership.cloud.KubernetesMembershipProvider\"/>\\n </Channel>\\n </Cluster>\\n' ${FILE}\n" +
			"fi\n"
	} else {
		cmd["test.sh"] = "FILE=`find /opt -name server.xml`\n" +
			"grep -q MembershipProvider ${FILE}\n" +
			"if [ $? -ne 0 ]; then\n" +
			"  sed -i '/cluster.html/a        <Cluster className=\"org.apache.catalina.ha.tcp.SimpleTcpCluster\" channelSendOptions=\"6\">\\n <Channel className=\"org.apache.catalina.tribes.group.GroupChannel\">\\n <Membership className=\"org.apache.catalina.tribes.membership.cloud.CloudMembershipService\" membershipProviderClassName=\"org.apache.catalina.tribes.membership.cloud.DNSMembershipProvider\"/>\\n </Channel>\\n </Cluster>\\n' ${FILE}\n" +
			"fi\n"
	}
	return cmd
}

// getPodList lists pods which belongs to the Web server
// the pods are differentiated based on the selectors
func getPodList(r *ReconcileWebServer, webServer *webserversv1alpha1.WebServer) (*corev1.PodList, error) {
	podList := &corev1.PodList{}

	listOpts := []client.ListOption{
		client.InNamespace(webServer.Namespace),
		client.MatchingLabels(generateLabelsForWeb(webServer)),
	}
	err := r.client.List(context.TODO(), podList, listOpts...)

	if err == nil {
		// sorting pods by number in the name
		sortPodListByName(podList)
	}
	return podList, err
}

// generateLabelsForWeb return a map of labels that are used for identification
//  of objects belonging to the particular WebServer instance
func generateLabelsForWeb(webServer *webserversv1alpha1.WebServer) map[string]string {
	labels := map[string]string{
		"deploymentConfig": webServer.Spec.ApplicationName,
		"WebServer":        webServer.Name,
	}
	// labels["app.kubernetes.io/name"] = webServer.Name
	// labels["app.kubernetes.io/managed-by"] = os.Getenv("LABEL_APP_MANAGED_BY")
	// labels["app.openshift.io/runtime"] = os.Getenv("LABEL_APP_RUNTIME")
	if webServer.Labels != nil {
		for labelKey, labelValue := range webServer.Labels {
			log.Info("labels: ", labelKey, " : ", labelValue)
			labels[labelKey] = labelValue
		}
	}
	return labels
}

// getPodStatus returns the pod names of the array of pods passed in
func getPodStatus(pods []corev1.Pod) ([]webserversv1alpha1.PodStatus, bool) {
	var requeue = false
	var podStatuses []webserversv1alpha1.PodStatus
	for _, pod := range pods {
		podState := webserversv1alpha1.PodStateFailed

		switch pod.Status.Phase {
		case corev1.PodPending:
			podState = webserversv1alpha1.PodStatePending
		case corev1.PodRunning:
			podState = webserversv1alpha1.PodStateActive
		}

		podStatuses = append(podStatuses, webserversv1alpha1.PodStatus{
			Name:  pod.Name,
			PodIP: pod.Status.PodIP,
			State: podState,
		})
		if pod.Status.PodIP == "" {
			requeue = true
		}
	}
	if requeue {
		log.Info("Some pods don't have an IP address yet, reconciliation requeue scheduled")
	}
	return podStatuses, requeue
}

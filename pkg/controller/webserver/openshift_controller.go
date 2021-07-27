package webserver

import (
	webserversv1alpha1 "github.com/web-servers/jws-operator/pkg/apis/webservers/v1alpha1"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// rbac "rbac.authorization.k8s.io/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileWebServer) generateImageStream(webServer *webserversv1alpha1.WebServer) *imagev1.ImageStream {

	imageStream := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "image.openshift.io/v1",
			Kind:       "ImageStream",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
	}

	controllerutil.SetControllerReference(webServer, imageStream, r.scheme)
	return imageStream
}

func (r *ReconcileWebServer) generateBuildConfig(webServer *webserversv1alpha1.WebServer) *buildv1.BuildConfig {

	buildConfig := &buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "build.openshift.io/v1",
			Kind:       "BuildConfig",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Type: "Git",
					Git: &buildv1.GitBuildSource{
						URI: webServer.Spec.WebImageStream.WebSources.SourceRepositoryURL,
						Ref: webServer.Spec.WebImageStream.WebSources.SourceRepositoryRef,
					},
					ContextDir: webServer.Spec.WebImageStream.WebSources.ContextDir,
				},
				Strategy: buildv1.BuildStrategy{
					Type: "Source",
					SourceStrategy: &buildv1.SourceBuildStrategy{
						Env:       generateEnvBuild(webServer),
						ForcePull: true,
						From: corev1.ObjectReference{
							Kind:      "ImageStreamTag",
							Namespace: webServer.Spec.WebImageStream.ImageStreamNamespace,
							Name:      webServer.Spec.WebImageStream.ImageStreamName + ":latest",
						},
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: webServer.Spec.ApplicationName + ":latest",
					},
				},
			},
			Triggers: generateBuildTriggerPolicy(webServer),
		},
	}

	controllerutil.SetControllerReference(webServer, buildConfig, r.scheme)
	return buildConfig
}

// Create the env for the maven build
func generateEnvBuild(webServer *webserversv1alpha1.WebServer) []corev1.EnvVar {
	var env []corev1.EnvVar
	sources := webServer.Spec.WebImageStream.WebSources
	if sources != nil {
		params := sources.WebSourcesParams
		if params != nil {
			if params.MavenMirrorURL != "" {
				env = append(env, corev1.EnvVar{
					Name:  "MAVEN_MIRROR_URL",
					Value: params.MavenMirrorURL,
				})
			}
			if params.ArtifactDir != "" {
				env = append(env, corev1.EnvVar{
					Name:  "ARTIFACT_DIR",
					Value: params.ArtifactDir,
				})
			}
		}
	}
	return env
}

// Create the BuildTriggerPolicy
func generateBuildTriggerPolicy(webServer *webserversv1alpha1.WebServer) []buildv1.BuildTriggerPolicy {
	env := []buildv1.BuildTriggerPolicy{
		{
			Type:        "ImageChange",
			ImageChange: &buildv1.ImageChangeTrigger{},
		},
		{
			Type: "ConfigChange",
		},
	}
	sources := webServer.Spec.WebImageStream.WebSources
	if sources != nil {
		params := sources.WebSourcesParams
		if params != nil {
			if params.GithubWebhookSecret != "" {
				env = append(env, buildv1.BuildTriggerPolicy{
					Type: "GitHub",
					GitHubWebHook: &buildv1.WebHookTrigger{
						Secret: params.GithubWebhookSecret,
					},
				})
			}
			if params.GenericWebhookSecret != "" {
				env = append(env, buildv1.BuildTriggerPolicy{
					Type: "Generic",
					GenericWebHook: &buildv1.WebHookTrigger{
						Secret: params.GenericWebhookSecret,
					},
				})
			}
		}
	}
	return env
}

func (r *ReconcileWebServer) generateDeploymentConfig(webServer *webserversv1alpha1.WebServer, imageStreamName string, imageStreamNamespace string) *appsv1.DeploymentConfig {

	replicas := int32(1)
	podTemplateSpec := r.generatePodTemplate(webServer, webServer.Spec.ApplicationName)
	deploymentConfig := &appsv1.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.openshift.io/v1",
			Kind:       "DeploymentConfig",
		},
		ObjectMeta: generateObjectMeta(webServer, webServer.Spec.ApplicationName),
		Spec: appsv1.DeploymentConfigSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyTypeRecreate,
			},
			Triggers: []appsv1.DeploymentTriggerPolicy{{
				Type: appsv1.DeploymentTriggerOnImageChange,
				ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
					Automatic:      true,
					ContainerNames: []string{webServer.Spec.ApplicationName},
					From: corev1.ObjectReference{
						Kind:      "ImageStreamTag",
						Name:      imageStreamName + ":latest",
						Namespace: imageStreamNamespace,
					},
				},
			},
				{
					Type: appsv1.DeploymentTriggerOnConfigChange,
				}},
			Replicas: replicas,
			Selector: map[string]string{
				"deploymentConfig": webServer.Spec.ApplicationName,
				"WebServer":        webServer.Name,
			},
			Template: &podTemplateSpec,
		},
	}

	controllerutil.SetControllerReference(webServer, deploymentConfig, r.scheme)
	return deploymentConfig
}

func (r *ReconcileWebServer) generateRoute(webServer *webserversv1alpha1.WebServer) *routev1.Route {
	objectMeta := generateObjectMeta(webServer, webServer.Spec.ApplicationName)
	objectMeta.Annotations = map[string]string{
		"description": "Route for application's http service.",
	}
	route := &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
		ObjectMeta: objectMeta,
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Name: webServer.Spec.ApplicationName,
			},
		},
	}

	controllerutil.SetControllerReference(webServer, route, r.scheme)
	return route
}

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	"time"

	"github.com/google/uuid"
	"github.com/nifetency/nife.io/api/model"
	"github.com/nifetency/nife.io/helper"
	cluster_details "github.com/nifetency/nife.io/internal/cluster_info"
	"github.com/nifetency/nife.io/internal/decode"
	organizationInfo "github.com/nifetency/nife.io/internal/organization_info"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	awsService "github.com/nifetency/nife.io/pkg/aws"
	_helper "github.com/nifetency/nife.io/pkg/helper"
	commonService "github.com/nifetency/nife.io/pkg/helper"

	// v1 "k8s.io/api/admissionregistration/v1"
	// v1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"

	// v1beta1 "k8s.io/api/extensions/v1beta1"
	ingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

// Deployment creates a new deployent object if it doesn't exists

func Deployment(orgSlug string, clientset *kubernetes.Clientset, internalPort int32, imageName, appName, secretName string, envArgs map[string]string, storage model.Requirement, regId string, replicas int) (string, string, error) {
	deploymentId := ""
	containerId := ""
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return "", "", err
	}
	usedContainerPort := make([]int32, len(pods.Items))
	// Checks if port is already In Use
	for _, pod := range pods.Items {
		if pod.Spec.Containers != nil {
			if pod.Spec.Containers[0].Ports != nil {
				usedContainerPort = append(usedContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
			}
			// if pod.Spec.Containers[1].Ports != nil {
			// 	usedContainerPort = append(usedContainerPort, pod.Spec.Containers[0].Ports[0].ContainerPort)
			// }
		}
	}

	internalPort = CheckPortInUse(usedContainerPort, internalPort)
	// Create deployment function call
	PrivateRegistry1, err := secretregistry.GetSecretDetails(regId, "")

	if err != nil {
		return "", "", err
	}
	if imageName == "mysql:5.6" || imageName == "postgres:10.1" {
		deploymentId, containerId, err = CreateDeploymentMySql(orgSlug, clientset, internalPort, imageName, appName, secretName, envArgs, storage, PrivateRegistry1, replicas)

	} else {
		deploymentId, containerId, err = CreateDeployment(orgSlug, clientset, internalPort, imageName, appName, secretName, envArgs, storage, replicas)
	}
	return deploymentId, containerId, err
}

func CreateConfigMap(clientset *kubernetes.Clientset, registry model.GetUserSecret, orgSlug, appName string) error {
	var postgres string
	if registry.RegistryType == nil {
		postgres = ""
	} else {
		postgres = *registry.RegistryType
	}
	if postgres == "postgres" {

		Password := decode.DePwdCode(*registry.PassWord)

		configMap := &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *registry.Name,
				Namespace: orgSlug,
				Labels: map[string]string{
					"app": appName,
				},
			},
			Data: map[string]string{
				"POSTGRES_DB":       *registry.RegistryName,
				"POSTGRES_USER":     *registry.UserName,
				"POSTGRES_PASSWORD": Password,
			},
		}

		// Method to create configmap
		configMaping, err := clientset.CoreV1().ConfigMaps(orgSlug).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		log.Printf("Created ConfigMap %q.\n", configMaping.GetObjectMeta().GetName())
	}
	return nil

}

func CreatePersistentVolume(clientset *kubernetes.Clientset, name, orgSlug string, PrivateRegistry model.GetUserSecret) error {
	var mySql string
	if PrivateRegistry.RegistryType == nil {
		mySql = ""
	} else {
		mySql = *PrivateRegistry.RegistryType
	}

	volumes, err := GetVolumeByAppName(name)
	if err != nil {
		return err
	}
	if mySql == "mysql" {
		persistentVolumeClient := clientset.CoreV1().PersistentVolumes()
		persistentVolumeClaimClient := clientset.CoreV1().PersistentVolumeClaims(orgSlug)

		PersistentVolume := apiv1.PersistentVolume{

			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
				Labels: map[string]string{
					"type": "local",
				},
			},

			Spec: apiv1.PersistentVolumeSpec{
				// VolumeMode: v1.PersistentVolumeMode(),

				StorageClassName: "manual",
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Capacity: apiv1.ResourceList{
					apiv1.ResourceName(apiv1.ResourceStorage): resource.MustParse("2Gi"),
				},
				PersistentVolumeSource: apiv1.PersistentVolumeSource{
					HostPath: &apiv1.HostPathVolumeSource{
						Path: "/mnt/data/" + name,
					},
				},
			},
		}
		log.Println("Creating Persistent Volume...")
		results, err := persistentVolumeClient.Create(context.TODO(), &PersistentVolume, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		// err = persistentVolumeClient.Delete(context.TODO(), name, metav1.DeleteOptions{})

		PersistentVolumeClaim := apiv1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
			},
			Spec: apiv1.PersistentVolumeClaimSpec{
				StorageClassName: &results.Spec.StorageClassName,
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					apiv1.ReadWriteOnce,
				},
				Resources: apiv1.ResourceRequirements{
					Requests: results.Spec.Capacity,
				},
			},
		}

		log.Println("Creating Persistent Volume Claim...")

		results1, err := persistentVolumeClaimClient.Create(context.TODO(), &PersistentVolumeClaim, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		fmt.Println(results1)
	} else if mySql == "postgres" {
		persistentVolumeClient := clientset.CoreV1().PersistentVolumes()
		persistentVolumeClaimClient := clientset.CoreV1().PersistentVolumeClaims(orgSlug)

		PersistentVolume := apiv1.PersistentVolume{

			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
				Labels: map[string]string{
					"type": "local",
					"app":  name,
				},
			},

			Spec: apiv1.PersistentVolumeSpec{
				// VolumeMode: v1.PersistentVolumeMode(),

				StorageClassName: "manual",
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					"ReadWriteMany",
				},
				Capacity: apiv1.ResourceList{
					apiv1.ResourceName(apiv1.ResourceStorage): resource.MustParse("2Gi"),
				},
				PersistentVolumeSource: apiv1.PersistentVolumeSource{
					HostPath: &apiv1.HostPathVolumeSource{
						Path: "/mnt/data/" + name,
					},
				},
			},
		}
		log.Println("Creating Persistent Volume...")
		results, err := persistentVolumeClient.Create(context.TODO(), &PersistentVolume, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		PersistentVolumeClaim := apiv1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
				Labels: map[string]string{
					"app": name,
				},
			},
			Spec: apiv1.PersistentVolumeClaimSpec{
				StorageClassName: &results.Spec.StorageClassName,
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					apiv1.ReadWriteMany,
				},
				Resources: apiv1.ResourceRequirements{
					Requests: results.Spec.Capacity,
				},
			},
		}

		log.Println("Creating Persistent Volume Claim...")

		results1, err := persistentVolumeClaimClient.Create(context.TODO(), &PersistentVolumeClaim, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		fmt.Println(results1)
	} else if volumes != nil {

		var volumePath string
		if volumes.Path == nil {
			volumePath = "/mnt/data/" + name
		} else {
			volumePath = *volumes.Path
		}

		persistentVolumeClient := clientset.CoreV1().PersistentVolumes()
		persistentVolumeClaimClient := clientset.CoreV1().PersistentVolumeClaims(orgSlug)
		volumeSize := *volumes.Size + "Gi"

		PersistentVolume := apiv1.PersistentVolume{

			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolume"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
				Labels: map[string]string{
					"type": "local",
				},
			},

			Spec: apiv1.PersistentVolumeSpec{
				// VolumeMode: v1.PersistentVolumeMode(),

				StorageClassName: "manual",
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					"ReadWriteOnce",
				},
				Capacity: apiv1.ResourceList{
					apiv1.ResourceName(apiv1.ResourceStorage): resource.MustParse(volumeSize),
				},
				PersistentVolumeSource: apiv1.PersistentVolumeSource{
					HostPath: &apiv1.HostPathVolumeSource{
						// Path: "/mnt/data/" + name,
						Path: volumePath,
					},
				},
			},
		}
		log.Println("Creating Persistent Volume...")
		results, err := persistentVolumeClient.Create(context.TODO(), &PersistentVolume, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		// err = persistentVolumeClient.Delete(context.TODO(), name, metav1.DeleteOptions{})

		PersistentVolumeClaim := apiv1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{Kind: "PersistentVolumeClaim"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: orgSlug,
			},
			Spec: apiv1.PersistentVolumeClaimSpec{
				StorageClassName: &results.Spec.StorageClassName,
				AccessModes: []apiv1.PersistentVolumeAccessMode{
					apiv1.ReadWriteOnce,
				},
				Resources: apiv1.ResourceRequirements{
					Requests: results.Spec.Capacity,
				},
			},
		}

		log.Println("Creating Persistent Volume Claim...")

		results1, err := persistentVolumeClaimClient.Create(context.TODO(), &PersistentVolumeClaim, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		fmt.Println(results1)
	}
	return nil
}

// CreateDeployment creates a new deployment

func CreateDeployment(orgSlug string, clientset *kubernetes.Clientset, internalPort int32, imageName, appName, secretName string, envArgs map[string]string, storage model.Requirement, replicas int) (string, string, error) {
	deploymentsClient := clientset.AppsV1().Deployments(orgSlug)
	var deployment *appsv1.Deployment
	deploymentId := uuid.NewString()

	volumes, err := GetVolumeByAppName(appName)
	if err != nil {
		return "", "", nil
	}

	//-----------------deployment volume----------
	if volumes != nil {

		resourceRequirement := CreateStorageResource(storage)

		var containerEnv []apiv1.EnvVar
		for n, v := range envArgs {
			containerEnv = append(containerEnv, apiv1.EnvVar{Name: n, Value: v})
		}

		// deploymentId := uuid.NewString()
		// deploymentsClient := clientset.AppsV1().Deployments(orgSlug)

		ContainerSpecVolume := NewContainerSpecVolume(appName, imageName, int(internalPort), containerEnv, resourceRequirement, volumes)
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: deploymentId,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: helper.Int32Ptr(int32(replicas)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": appName,
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": appName,
						},
					},
					Spec: apiv1.PodSpec{

						Containers: ContainerSpecVolume,
						Volumes: []apiv1.Volume{
							apiv1.Volume{
								Name: appName,
								VolumeSource: apiv1.VolumeSource{
									PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
										ClaimName: appName,
									},
								},
							},
						},

						ImagePullSecrets: []apiv1.LocalObjectReference{
							{
								Name: secretName,
							},
						},
					},
				},
			},
		}
		//----------------

	} else {

		resourceRequirement := CreateStorageResource(storage)

		var containerEnv []apiv1.EnvVar
		for n, v := range envArgs {
			containerEnv = append(containerEnv, apiv1.EnvVar{Name: n, Value: v})
		}

		// deploymentId := uuid.NewString()
		// deploymentsClient := clientset.AppsV1().Deployments(orgSlug)

		ContainerSpec := NewContainerSpec(appName, imageName, int(internalPort), containerEnv, resourceRequirement)
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: deploymentId,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: helper.Int32Ptr(int32(replicas)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": appName,
					},
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": appName,
						},
					},
					Spec: apiv1.PodSpec{

						Containers: ContainerSpec,

						ImagePullSecrets: []apiv1.LocalObjectReference{
							{
								Name: secretName,
							},
						},
					},
				},
			},
		}
	}
	// Create Deployment
	log.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		return "", "", err
	}
	log.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

	options := metav1.ListOptions{
		LabelSelector: "app=" + appName,
	}

	time.Sleep(time.Second * 5)

	podList, _ := clientset.CoreV1().Pods(orgSlug).List(context.TODO(), options)

	time.Sleep(time.Second * 5)

	containerId := ""
	// List() returns a pointer to slice, derefernce it, before iterating

	length := len(podList.Items)

	loop := 0
	for _, podInfo := range (*podList).Items {

		containerId = podInfo.Name
		fmt.Println("containerId", containerId)

		checkId := strings.Contains(containerId, deploymentId)
		if !checkId {
			if loop != length {
				loop += 1
				continue
			} else {
				return "", "", fmt.Errorf("SomeThing Wrong In ContainerId")
			}
		} else {
			break
		}
	}
	if containerId == "" {
		return "", "", fmt.Errorf("containerId must not be empty")
	}
	return deploymentId, containerId, nil
}

func CreateDeploymentMySql(orgSlug string, clientset *kubernetes.Clientset, internalPort int32, imageName, appName, secretName string, envArgs map[string]string, storage model.Requirement, PrivateRegistry1 model.GetUserSecret, replicas int) (string, string, error) {

	// resourceRequirement := CreateStorageResource(storage)
	if imageName == "mysql:5.6" {
		var containerEnv []apiv1.EnvVar
		for n, v := range envArgs {
			containerEnv = append(containerEnv, apiv1.EnvVar{Name: n, Value: v})
		}

		deploymentId := uuid.NewString()
		deploymentsClient := clientset.AppsV1().Deployments(orgSlug)

		// ContainerSpec := NewContainerSpec(appName, imageName, int(internalPort), containerEnv, resourceRequirement)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentId,
				Namespace: orgSlug,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: helper.Int32Ptr(int32(replicas)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": appName,
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RecreateDeploymentStrategyType,
				},
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": appName,
						},
					},
					Spec: apiv1.PodSpec{

						Containers: []apiv1.Container{
							{
								Name:  appName,
								Image: imageName,
								Env: []apiv1.EnvVar{
									{
										Name: "MYSQL_ROOT_PASSWORD",
										ValueFrom: &apiv1.EnvVarSource{
											SecretKeyRef: &apiv1.SecretKeySelector{
												LocalObjectReference: apiv1.LocalObjectReference{
													Name: secretName,
												},
												Key: "password",
											},
										},
									},
								},
								Ports: []apiv1.ContainerPort{
									{
										ContainerPort: 3306,
										Name:          appName,
									},
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      appName + "nife",
										MountPath: "/var/lib/" + appName,
									},
								},
							},
						},
						Volumes: []apiv1.Volume{
							apiv1.Volume{
								Name: appName + "nife",
								VolumeSource: apiv1.VolumeSource{
									PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
										ClaimName: appName,
									},
								},
							},
						},
					},
				},
			},
		}

		// Create Deployment
		log.Println("Creating deployment...")
		result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			return "", "", err
		}
		log.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

		options := metav1.ListOptions{
			LabelSelector: "app=" + appName,
		}

		time.Sleep(time.Second * 5)

		podList, _ := clientset.CoreV1().Pods(orgSlug).List(context.TODO(), options)

		time.Sleep(time.Second * 5)

		containerId := ""
		// List() returns a pointer to slice, derefernce it, before iterating

		length := len(podList.Items)

		loop := 0
		for _, podInfo := range (*podList).Items {

			containerId = podInfo.Name
			fmt.Println("containerId", containerId)

			checkId := strings.Contains(containerId, deploymentId)
			if !checkId {
				if loop != length {
					loop += 1
					continue
				} else {
					return "", "", fmt.Errorf("SomeThing Wrong In ContainerId")
				}
			} else {
				break
			}
		}
		if containerId == "" {
			return "", "", fmt.Errorf("containerId must not be empty")
		}

		return deploymentId, containerId, nil
	} else if imageName == "postgres:10.1" {
		var containerEnv []apiv1.EnvVar
		for n, v := range envArgs {
			containerEnv = append(containerEnv, apiv1.EnvVar{Name: n, Value: v})
		}

		deploymentId := uuid.NewString()
		deploymentsClient := clientset.AppsV1().Deployments(orgSlug)

		// ContainerSpec := NewContainerSpec(appName, imageName, int(internalPort), containerEnv, resourceRequirement)
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentId,
				Namespace: orgSlug,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: helper.Int32Ptr(int32(replicas)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": appName,
					},
				},
				// Strategy: appsv1.DeploymentStrategy{
				// 	Type: appsv1.RecreateDeploymentStrategyType,
				// },
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": appName,
						},
					},
					Spec: apiv1.PodSpec{

						Containers: []apiv1.Container{
							{
								Name:            appName,
								Image:           imageName,
								ImagePullPolicy: "IfNotPresent",
								Ports: []apiv1.ContainerPort{
									{
										ContainerPort: 5432,
										Name:          appName,
									},
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      appName + "nife",
										MountPath: "/var/lib/postgresql/data/" + appName,
									},
								},
								EnvFrom: []apiv1.EnvFromSource{
									{
										ConfigMapRef: &apiv1.ConfigMapEnvSource{
											LocalObjectReference: apiv1.LocalObjectReference{
												Name: *PrivateRegistry1.Name,
											},
										},
									},
								},
							},
						},
						Volumes: []apiv1.Volume{
							apiv1.Volume{
								Name: appName + "nife",
								VolumeSource: apiv1.VolumeSource{
									PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
										ClaimName: appName,
									},
								},
							},
						},
					},
				},
			},
		}

		// Create Deployment
		log.Println("Creating deployment...")
		result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
		if err != nil {
			return "", "", err
		}
		log.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())

		options := metav1.ListOptions{
			LabelSelector: "app=" + appName,
		}

		time.Sleep(time.Second * 5)

		podList, _ := clientset.CoreV1().Pods(orgSlug).List(context.TODO(), options)

		time.Sleep(time.Second * 5)

		containerId := ""
		// List() returns a pointer to slice, derefernce it, before iterating

		length := len(podList.Items)

		loop := 0
		for _, podInfo := range (*podList).Items {

			containerId = podInfo.Name
			fmt.Println("containerId", containerId)

			checkId := strings.Contains(containerId, deploymentId)
			if !checkId {
				if loop != length {
					loop += 1
					continue
				} else {
					return "", "", fmt.Errorf("SomeThing Wrong In ContainerId")
				}
			} else {
				break
			}
		}
		if containerId == "" {
			return "", "", fmt.Errorf("containerId must not be empty")
		}
		return deploymentId, containerId, nil

	}
	return "", "", nil
}

func NewContainerSpec(appName, image string, internalPort int, env []apiv1.EnvVar, resourceRequirement apiv1.ResourceRequirements) (conSpec []apiv1.Container) {
	new := apiv1.Container{}
	new.Name = appName
	new.Image = image
	new.Ports = []apiv1.ContainerPort{
		{
			Name:          "http",
			Protocol:      apiv1.ProtocolTCP,
			HostPort:      int32(internalPort),
			ContainerPort: int32(internalPort),
		},
	}
	new.Env = env

	new.Resources = resourceRequirement
	conSpec = append(conSpec, new)
	return conSpec
}
func NewContainerSpecVolume(appName, image string, internalPort int, env []apiv1.EnvVar, resourceRequirement apiv1.ResourceRequirements, volumekube *model.DuploVolumeInput) (conSpec []apiv1.Container) {
	new := apiv1.Container{}
	new.Name = appName
	new.Image = image
	new.Ports = []apiv1.ContainerPort{
		{
			Name:          "http",
			Protocol:      apiv1.ProtocolTCP,
			HostPort:      int32(internalPort),
			ContainerPort: int32(internalPort),
		},
	}
	new.Env = env

	new.Resources = resourceRequirement
	new.VolumeMounts = []apiv1.VolumeMount{
		apiv1.VolumeMount{
			Name:      appName,
			MountPath: *volumekube.Path,
		},
	}

	conSpec = append(conSpec, new)
	return conSpec
}

func CreateStorageResource(storage model.Requirement) (result apiv1.ResourceRequirements) {
	units := ""
	resourceRequirement := apiv1.ResourceRequirements{}

	fmt.Println("entered1")

	if *storage.LimitRequirement.CPU != "" {
		units = os.Getenv("DEFAULT_MEMORY_UNITS")
		resourceRequirement = apiv1.ResourceRequirements{
			Requests: apiv1.ResourceList{
				"cpu":    resource.MustParse(*storage.RequestRequirement.CPU),
				"memory": resource.MustParse(*storage.RequestRequirement.Memory + units),
			},
			Limits: apiv1.ResourceList{
				"cpu":    resource.MustParse(*storage.LimitRequirement.CPU),
				"memory": resource.MustParse(*storage.LimitRequirement.Memory + units),
			},
		}

		fmt.Println("entered2")

	}

	return resourceRequirement

}

// Service fetches list of service and create one if not exists
func Services(clientset *kubernetes.Clientset, internalPort int32, imageName, appName, serviceName, orgSlug string, clusterDetails cluster_details.ClusterDetail, userId string, reDeployBool bool) (string, string, int32, error) {
	// Create a new service which returns hostname and port
	var hostname string
	var port int32
	var err error
	if imageName == "mysql:5.6" || imageName == "postgres:10.1" {
		serviceName, hostname, port, err = CreateServiceMySql(clientset, internalPort, imageName, appName, serviceName, orgSlug, clusterDetails)
		if err != nil && hostname != "" {
			return "", "", 0, err
		}
	} else {

		if reDeployBool {
			serviceName, hostname, port, err = UpdateService(clientset, internalPort, imageName, appName, serviceName, orgSlug, clusterDetails, userId)
			if err != nil && hostname != "" {
				return "", "", 0, err
			}
		} else {

			serviceName, hostname, port, err = CreateService(clientset, internalPort, imageName, appName, serviceName, orgSlug, clusterDetails, userId)
			if err != nil && hostname != "" {
				return "", "", 0, err
			}
			if err != nil {
				return "", "", 0, err
			}
		}
		if clusterDetails.ClusterType == "byoh" {
			appDetails, err := GetApp(appName, userId)
			if err != nil {
				return "", "", 0, err
			}
			externalPort, _ := _helper.GetExternalPort(appDetails.Config.Definition)
			ingHostName, err := CreateIngress(clientset, int32(externalPort), serviceName, appName, orgSlug, clusterDetails)
			if err != nil {
				return "", "", 0, err
			}
			hostname = ingHostName

		}

	}
	return serviceName, hostname, port, nil

}

func CreateIngress(clientset *kubernetes.Clientset, externalPort int32, serviceName, appName, orgSlug string, clusterDetails cluster_details.ClusterDetail) (string, error) {
	var annotations map[string]string
	if clusterDetails.Region_code == "BYOH-10" {
		annotations = map[string]string{
			"kubernetes.io/ingress.class": "traefik",
		}
	} else {
		annotations = map[string]string{
			"kubernetes.io/ingress.class": "nginx",
		}
	}

	ingressName := appName + "-ingress"
	hostUrl := appName + "." + *clusterDetails.ExternalBaseAddress
	pathType := "Prefix"

	ing := ingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ingressName,
			Namespace:   orgSlug,
			Annotations: annotations,
		},
		Spec: ingv1.IngressSpec{
			Rules: []ingv1.IngressRule{
				ingv1.IngressRule{
					Host: hostUrl,
					IngressRuleValue: ingv1.IngressRuleValue{
						HTTP: &ingv1.HTTPIngressRuleValue{
							Paths: []ingv1.HTTPIngressPath{
								ingv1.HTTPIngressPath{
									Path:     "/",
									PathType: (*ingv1.PathType)(&pathType),
									Backend: ingv1.IngressBackend{
										Service: &ingv1.IngressServiceBackend{
											Name: serviceName,
											Port: ingv1.ServiceBackendPort{
												Number: externalPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ingressss, err := clientset.NetworkingV1().Ingresses(orgSlug).Create(context.Background(), &ing, metav1.CreateOptions{})
	if err != nil {
		log.Println(err)
		return "", err
	}
	ingrs, err := clientset.NetworkingV1().Ingresses(orgSlug).Get(context.Background(), ingressss.Name, metav1.GetOptions{})
	if err != nil {
		log.Println(err)
		return "", err
	}
	var hostName string
	for _, k := range ingrs.Spec.Rules {
		fmt.Println(k.Host)
		hostName = k.Host
	}

	return hostName, nil
}

// CreateService creates a new service for the deployment
func CreateService(clientset *kubernetes.Clientset, internalPort int32, imageName, appName, serviceName, orgSlug string, clusterDetails cluster_details.ClusterDetail, userId string) (string, string, int32, error) {
	if *&clusterDetails.ClusterType == "byoh" {
		app, err := GetApp(appName, userId)
		if err != nil {
			return "", "", 0, err
		}

		externalPort, err := commonService.GetExternalPort(app.Config.Definition)

		ser := &apiv1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind: "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
			},
			Spec: apiv1.ServiceSpec{
				Type: apiv1.ServiceTypeClusterIP,
				Ports: []apiv1.ServicePort{
					apiv1.ServicePort{
						Port:       int32(externalPort),
						TargetPort: intstr.IntOrString{IntVal: internalPort},
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
			},
		}

		// Creating service
		log.Println("Creating service...")
		serviceCreate, err := clientset.CoreV1().Services(orgSlug).Create(context.TODO(), ser, metav1.CreateOptions{})
		if err != nil {
			return "", "", 0, err
		}
		log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())

		var port int32
		hostname := ""
		ipAddres := ""

		if *clusterDetails.ProviderType == "gcp" || *clusterDetails.ProviderType == "azure" {
			hostname = ipAddres
		}

		return serviceCreate.GetObjectMeta().GetName(), hostname, port, nil
	} else {
		// Get APP
		app, err := GetApp(appName, userId)
		if err != nil {
			return "", "", 0, err
		}

		externalPort, err := commonService.GetExternalPort(app.Config.Definition)

		ser := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
			},
			Spec: apiv1.ServiceSpec{
				Ports: []apiv1.ServicePort{
					{
						Name:       appName,
						Protocol:   apiv1.ProtocolTCP,
						Port:       int32(externalPort),
						TargetPort: intstr.IntOrString{IntVal: internalPort},
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
				Type: apiv1.ServiceTypeLoadBalancer,
			},
		}

		// Creating service
		log.Println("Creating service...")
		serviceCreate, err := clientset.CoreV1().Services(orgSlug).Create(context.TODO(), ser, metav1.CreateOptions{})
		if err != nil {
			return "", "", 0, err
		}
		log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())

		// Goroutine function to get hostname and port
		// portCh := make(chan int32)
		// hostnameCh := make(chan string)
		// ipCh := make(chan string)
		// go func() {
		var port int32
		hostname := ""
		ipAddres := ""
		// for {
		// 	serviceGet, err := clientset.CoreV1().Services(orgSlug).Get(context.TODO(), serviceCreate.GetObjectMeta().GetName(), metav1.GetOptions{})
		// 	fmt.Println("error in loop", err)
		// 	if err != nil {
		// 		break
		// 	}

		// 	if len(serviceGet.Status.LoadBalancer.Ingress) > 0 {
		// 		port = serviceGet.Spec.Ports[0].Port
		// 		ipAddres = serviceGet.Status.LoadBalancer.Ingress[0].IP
		// 		hostname = serviceGet.Status.LoadBalancer.Ingress[0].Hostname
		// 		break
		// 	}
		// 	time.Sleep(time.Second * 5)
		// }
		maxAttemptEnv := os.Getenv("MAX_ATTEMPTS_TO_FETCH_LB_URL")
		maxAttempts, _ := strconv.Atoi(maxAttemptEnv)
		checkLBUrl := true
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			serviceGet, err := clientset.CoreV1().Services(orgSlug).Get(context.TODO(), serviceCreate.GetObjectMeta().GetName(), metav1.GetOptions{})
			fmt.Println("error in loop", err)
			if err != nil {
				break
			}

			if len(serviceGet.Status.LoadBalancer.Ingress) > 0 {
				port = serviceGet.Spec.Ports[0].Port
				ipAddres = serviceGet.Status.LoadBalancer.Ingress[0].IP
				hostname = serviceGet.Status.LoadBalancer.Ingress[0].Hostname
				break
			}

			if attempt == maxAttempts {
				fmt.Println("Maximum attempts reached")
				checkLBUrl = false
				break
			}

			time.Sleep(time.Second * 5)
		}
		// }()
		if !checkLBUrl {
			deploymentId := strings.ReplaceAll(serviceName, "service-", "")

			deploymentList, _ := clientset.AppsV1().Deployments(orgSlug).List(context.TODO(), metav1.ListOptions{})
			for _, item := range deploymentList.Items {
				if deploymentId == item.Name {
					err := clientset.AppsV1().Deployments(orgSlug).Delete(context.TODO(), deploymentId, metav1.DeleteOptions{})
					if err != nil {
						return "", "", 0, err
					}
				}
			}

			err := clientset.CoreV1().Services(orgSlug).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
			if err != nil {
				return "", "", 0, err
			}

			return "", "", 0, fmt.Errorf("Maximum attempts reached while waiting for service creation. The cluster is currently full and unable to accommodate new services. Please contact your administrator for assistance.")
		}
		if *clusterDetails.ProviderType == "gcp" || *clusterDetails.ProviderType == "azure" {
			hostname = ipAddres
		}
		return serviceCreate.GetObjectMeta().GetName(), hostname, port, nil
	}
}

func UpdateService(clientset *kubernetes.Clientset, internalPort int32, imageName, appName, serviceName, orgSlug string, clusterDetails cluster_details.ClusterDetail, userId string) (string, string, int32, error) {

	var getDeploymentId string
	getServiceDet, err := _helper.GetAppReleaseByVersion(appName)
	if err != nil {
		return "", "", 0, err
	}

	getDeploymentdet, err := _helper.GetDeploymentIdByReleaseId(appName, getServiceDet.Id)
	if err != nil {
		return "", "", 0, err
	}
	for _, depId := range *getDeploymentdet {
		getDeploymentId = "service-" + depId.Deployment_id
	}

	if *&clusterDetails.ClusterType == "byoh" {
		app, err := GetApp(appName, userId)
		if err != nil {
			return "", "", 0, err
		}

		externalPort, err := commonService.GetExternalPort(app.Config.Definition)

		ser := &apiv1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind: "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: getDeploymentId,
			},
			Spec: apiv1.ServiceSpec{
				Type: apiv1.ServiceTypeClusterIP,
				Ports: []apiv1.ServicePort{
					apiv1.ServicePort{
						Port:       int32(externalPort),
						TargetPort: intstr.IntOrString{IntVal: internalPort},
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
			},
		}

		// Creating service
		log.Println("Creating service...")
		serviceCreate, err := clientset.CoreV1().Services(orgSlug).Create(context.TODO(), ser, metav1.CreateOptions{})
		if err != nil {
			return "", "", 0, err
		}
		log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())

		var port int32
		hostname := ""
		ipAddres := ""

		if *clusterDetails.ProviderType == "gcp" || *clusterDetails.ProviderType == "azure" {
			hostname = ipAddres
		}

		return serviceCreate.GetObjectMeta().GetName(), hostname, port, nil
	} else {
		// Get APP
		app, err := GetApp(appName, userId)
		if err != nil {
			return "", "", 0, err
		}

		externalPort, err := commonService.GetExternalPort(app.Config.Definition)

		ser := &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: getDeploymentId,
			},
			Spec: apiv1.ServiceSpec{
				Ports: []apiv1.ServicePort{
					{
						Name:       appName,
						Protocol:   apiv1.ProtocolTCP,
						Port:       int32(externalPort),
						TargetPort: intstr.IntOrString{IntVal: internalPort},
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
				Type: apiv1.ServiceTypeLoadBalancer,
			},
		}

		// Creating service
		log.Println("Creating service...")
		// serviceCreate, err := clientset.CoreV1().Services(orgSlug).Create(context.TODO(), ser, metav1.CreateOptions{})
		// if err != nil {
		// 	return "", "", 0, err
		// }
		// log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())
		serviceCreate, err := clientset.CoreV1().Services(orgSlug).Update(context.TODO(), ser, metav1.UpdateOptions{})
		if err != nil {
			return "", "", 0, err
		}
		log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())

		// Goroutine function to get hostname and port
		// portCh := make(chan int32)
		// hostnameCh := make(chan string)
		// ipCh := make(chan string)
		// go func() {
		var port int32
		hostname := ""
		ipAddres := ""
		maxAttemptEnv := os.Getenv("MAX_ATTEMPTS_TO_FETCH_LB_URL")
		maxAttempts, _ := strconv.Atoi(maxAttemptEnv)
		checkLBUrl := true
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			serviceGet, err := clientset.CoreV1().Services(orgSlug).Get(context.TODO(), serviceCreate.GetObjectMeta().GetName(), metav1.GetOptions{})
			fmt.Println("error in loop", err)
			if err != nil {
				break
			}

			if len(serviceGet.Status.LoadBalancer.Ingress) > 0 {
				port = serviceGet.Spec.Ports[0].Port
				ipAddres = serviceGet.Status.LoadBalancer.Ingress[0].IP
				hostname = serviceGet.Status.LoadBalancer.Ingress[0].Hostname
				break
			}

			if attempt == maxAttempts {
				fmt.Println("Maximum attempts reached")
				checkLBUrl = false
				break
			}

			time.Sleep(time.Second * 5)
		}
		// }()
		if !checkLBUrl {
			return "", "", 0, fmt.Errorf("Maximum attempts reached while waiting for service creation. The cluster is currently full and unable to accommodate new services. Please contact your administrator for assistance.")
		}

		if *clusterDetails.ProviderType == "gcp" || *clusterDetails.ProviderType == "azure" {
			hostname = ipAddres
		}
		return serviceCreate.GetObjectMeta().GetName(), hostname, port, nil
	}
}

func CreateServiceMySql(clientset *kubernetes.Clientset, internalPort int32, imageName, appName, serviceName, orgSlug string, clusterDetails cluster_details.ClusterDetail) (string, string, int32, error) {

	// Get APP
	// app, err := GetApp(appName)
	// if err != nil {
	// 	return "", "", 0, err
	// }
	var portss int
	var typees string
	if imageName == "postgres:10.1" {
		portss = 5432
		typees = string(apiv1.ServiceTypeNodePort)
	} else if imageName == "mysql:5.6" {
		portss = 3306
		typees = string(apiv1.ServiceTypeClusterIP)

	}
	var ser *apiv1.Service
	// externalPort, err := commonService.GetExternalPort(app.Config.Definition)
	if imageName == "mysql:5.6" {

		ser = &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: orgSlug,
			},
			Spec: apiv1.ServiceSpec{
				Type: apiv1.ServiceType(typees),
				Ports: []apiv1.ServicePort{
					{
						Port:       int32(portss),
						TargetPort: intstr.IntOrString{IntVal: int32(3306)},
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
			},
		}
	} else if imageName == "postgres:10.1" {
		ser = &apiv1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: orgSlug,
				Labels: map[string]string{
					"app": serviceName,
				},
			},
			Spec: apiv1.ServiceSpec{
				// Type: apiv1.ServiceType(typees),
				Type: apiv1.ServiceTypeNodePort,
				Ports: []apiv1.ServicePort{
					{
						Port: int32(5432),
					},
				},
				Selector: map[string]string{
					"app": appName,
				},
			},
		}
	}
	// Creating service
	log.Println("Creating service...")
	serviceCreate, err := clientset.CoreV1().Services(orgSlug).Create(context.TODO(), ser, metav1.CreateOptions{})
	if err != nil {
		return "", "", 0, err
	}
	log.Printf("Created Service %q.\n", serviceCreate.GetObjectMeta().GetName())

	// Goroutine function to get hostname and port
	// portCh := make(chan int32)
	// hostnameCh := make(chan string)
	// ipCh := make(chan string)
	// go func() {
	var port int32
	hostname := ""
	// ipAddres := ""
	// for {
	// 	serviceGet, err := clientset.CoreV1().Services(orgSlug).Get(context.TODO(), serviceCreate.GetObjectMeta().GetName(), metav1.GetOptions{})
	// 	fmt.Println("error in loop", err)
	// 	if err != nil {
	// 		break
	// 	}

	// 	if len(serviceGet.Status.LoadBalancer.Ingress) > 0 {
	// 		port = serviceGet.Spec.Ports[0].Port
	// 		ipAddres = serviceGet.Status.LoadBalancer.Ingress[0].IP
	// 		hostname = serviceGet.Status.LoadBalancer.Ingress[0].Hostname
	// 		break
	// 	}
	// 	time.Sleep(time.Second * 5)
	// }
	// }()

	// if *clusterDetails.ProviderType == "gcp" {
	// 	hostname = ipAddres
	// }

	return serviceCreate.GetObjectMeta().GetName(), hostname, port, nil
}

// GetURL return the hostname and port of the exposed service
// func GetURL(clientset *kubernetes.Clientset, input model.DeployInput) (string, int32, error) {
// 	services, err := listService(clientset)
// 	if err != nil {
// 		return "", 0, nil
// 	}
// 	var hostname string
// 	var port int32
// 	for _, service := range services.Items {
// 		if string(service.Name) == *input.ServiceName {
// 			if len(service.Spec.Ports) > 0 {
// 				port = service.Spec.Ports[0].Port
// 			}
// 			if len(service.Status.LoadBalancer.Ingress) > 0 {
// 				hostname = service.Status.LoadBalancer.Ingress[0].Hostname
// 			}
// 			return hostname, port, nil
// 		}
// 	}
// 	return "", 0, nil
// }

// ListService fetch list of service in default namespace
func listService(clientset *kubernetes.Clientset, org *model.Organization) (*apiv1.ServiceList, error) {
	serviceList, err := clientset.CoreV1().Services(*org.Slug).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return serviceList, nil
	}
	return serviceList, nil
}

func CreateSecret(clientset *kubernetes.Clientset, registry model.GetUserSecret, orgSlug string) error {

	// key := []byte(os.Getenv("ENCRYPTION_KEY"))

	// encrypt base64 crypto to original value
	// decrypted := commonService.Decrypt(key, *registry.PassWord)

	Password := decode.DePwdCode(*registry.PassWord)

	dockerCred := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s","email":"%s"}}}`, *registry.URL, *registry.UserName, Password, "")
	fmt.Println(dockerCred)
	sec := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: *registry.Name,
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte(dockerCred),
		},
		Type: "kubernetes.io/dockerconfigjson",
	}
	// Method to create Secret
	secCreate, err := clientset.CoreV1().Secrets(orgSlug).Create(context.TODO(), sec, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("Created Secret %q.\n", secCreate.GetObjectMeta().GetName())
	return nil
}

func CreateSecretMysql(clientset *kubernetes.Clientset, registry model.GetUserSecret, orgSlug string) error {

	// key := []byte(os.Getenv("ENCRYPTION_KEY"))

	// encrypt base64 crypto to original value
	// decrypted := commonService.Decrypt(key, *registry.PassWord)

	Password := decode.DePwdCode(*registry.PassWord)

	// dockerCred := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s","email":"%s"}}}`, *registry.URL, *registry.UserName, Password, "")
	// fmt.Println(dockerCred)
	sec := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      *registry.Name,
			Namespace: orgSlug,
		},
		Type: "kubernetes.io/basic-auth",
		StringData: map[string]string{
			"password": Password,
		},
	}
	// Method to create Secret
	secCreate, err := clientset.CoreV1().Secrets(orgSlug).Create(context.TODO(), sec, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	log.Printf("Created Secret %q.\n", secCreate.GetObjectMeta().GetName())
	return nil
}

// Secret function lists all secrets created and create a new if one does not exists.
func Secret(clientset *kubernetes.Clientset, registry model.GetUserSecret, orgSlug string) error {

	var organizationSlug string
	fmt.Println(registry.OrganizationID)
	if registry.OrganizationID != nil {
		slug, err := GetOrganizationById(*registry.OrganizationID)
		if err != nil {
			log.Println(err)
			return err
		}
		organizationSlug = *slug.Slug
	} else {
		organizationSlug = orgSlug
	}
	secretList, err := ListSecrets(clientset, organizationSlug)
	if err != nil {

		return err
	}

	for _, secret := range secretList.Items {
		if secret.GetObjectMeta().GetName() == *registry.Name {
			return nil
		}
	}

	var registryType string
	if registry.RegistryType == nil {
		registryType = ""
	} else {
		registryType = *registry.RegistryType
	}

	if registryType == "mysql" {
		err = CreateSecretMysql(clientset, registry, organizationSlug)
		if err != nil {
			return err
		}
	} else {
		err = CreateSecret(clientset, registry, organizationSlug)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListService fetch list of service in default namespace
func ListSecrets(clientset *kubernetes.Clientset, orgSlug string) (*apiv1.SecretList, error) {
	secretList, err := clientset.CoreV1().Secrets(orgSlug).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		return secretList, err
	}
	return secretList, nil
}
func DeleteSecrets(clientset *kubernetes.Clientset, orgSlug, secretName string) error {
	err := clientset.CoreV1().Secrets(orgSlug).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func SuspendResumePods(clusterDetails cluster_details.ClusterDetail, appID, appStatus string, scale int32, org *model.Organization) (string, error) {
	statusToCheck := appStatus
	updateAppStatus := "running"
	if statusToCheck == updateAppStatus {
		updateAppStatus = "suspended"
	}
	getDeployment, getDeploymentErr := commonService.GetDeploymentsRecordSingle(appID, clusterDetails.Region_code, statusToCheck)
	if getDeploymentErr != nil {
		return "", getDeploymentErr
	}
	println("UnDeploy ConfigPath", clusterDetails.Cluster_config_path)

	var fileSavePath string
	if clusterDetails.Cluster_config_path == "" {
		k8sPath := "k8s_config/" + clusterDetails.Region_code
		err := os.Mkdir(k8sPath, 0755)
		if err != nil {
			return "", err
		}

		fileSavePath = k8sPath + "/config"

		_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterDetails.ClusterConfigURL, fileSavePath)
		if err != nil {
			return "", err
		}
		clusterDetails.Cluster_config_path = "./k8s_config/" + clusterDetails.Region_code + "/config"
	}

	clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)

	if err != nil {
		helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
		return "", err
	}
	helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)

	deploymentName := getDeployment.Deployment_id
	s, err := clientset.AppsV1().Deployments(*org.Slug).GetScale(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	sc := *s
	sc.Spec.Replicas = scale

	_, err = clientset.AppsV1().Deployments(*org.Slug).UpdateScale(context.TODO(), deploymentName, &sc, metav1.UpdateOptions{})
	if err != nil {
		return "", err
	}
	_ = UpdateDeploymentsRecord(updateAppStatus, appID, getDeployment.Deployment_id, time.Now())
	err = UpdateAppStatus(appID)
	if err != nil {
		return "", err
	}
	return "", nil
}

func DeleteDeployment(clientset *kubernetes.Clientset, deploymentId string, org *model.Organization, redeployordelete bool, appId string) error {

	deploymentList, _ := clientset.AppsV1().Deployments(*org.Slug).List(context.TODO(), metav1.ListOptions{})
	for _, item := range deploymentList.Items {
		if deploymentId == item.Name {
			err := clientset.AppsV1().Deployments(*org.Slug).Delete(context.TODO(), deploymentId, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	if !redeployordelete {
		var deployId string
		getServiceDet, err := _helper.GetAppReleaseByVersion(appId)
		if err != nil {
			return err
		}

		getDeploymentdet, err := _helper.GetDeploymentIdByReleaseId(appId, getServiceDet.Id)
		if err != nil {
			return err
		}
		for _, depId := range *getDeploymentdet {
			deployId = depId.Deployment_id
		}
		deploymentId = deployId
	}

	if !redeployordelete {
		serviceName := "service-" + deploymentId
		serviceList, _ := listService(clientset, org)
		for _, service := range serviceList.Items {
			if string(service.Name) == serviceName {
				err := clientset.CoreV1().Services(*org.Slug).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func GetRegistryInfo() (model.GetUserSecret, error) {
	query := `select id, name, organization_id, registry_type, username, password, url, key_file_content, registry_name from organization_secrets where is_default = 1`

	selDB, err := database.Db.Query(query)
	if err != nil {
		return model.GetUserSecret{}, err
	}
	defer selDB.Close()

	var secrets model.GetUserSecret
	selDB.Next()
	err = selDB.Scan(&secrets.ID, &secrets.Name, &secrets.OrganizationID, &secrets.RegistryType, &secrets.UserName, &secrets.PassWord, &secrets.URL, &secrets.KeyFileContent, &secrets.RegistryName)
	if err != nil {
		return model.GetUserSecret{}, err
	}

	return model.GetUserSecret{
		ID:             secrets.ID,
		OrganizationID: secrets.OrganizationID,
		Name:           secrets.Name,
		RegistryType:   secrets.RegistryType,
		UserName:       secrets.UserName,
		PassWord:       secrets.PassWord,
		URL:            secrets.URL,
		KeyFileContent: secrets.KeyFileContent,
		RegistryName:   secrets.RegistryName,
	}, nil

}

func Deploy(appId, imageName, regId, orgSlug string, internalPort int32, clusterDetails cluster_details.ClusterDetail, envArgs string, storage model.Requirement, userId string, redeployBool bool) (*model.DeployOutput, error) {
	var deployOutput model.DeployOutput
	statusToCheck := "running"
	// hostname := appId + "." + os.Getenv("HOST_NAME")     // DNS route53
	hostEndpoint := os.Getenv("HOST_NAME_CLB")
	hostname := appId + "." + hostEndpoint

	envArgString := make(map[string]string)
	if envArgs != "" && envArgs != "{}" {

		err := json.Unmarshal([]byte(envArgs), &envArgString)
		if err != nil {
			return nil, err
		}
	}
	appDeployment, getDeploymentErr := commonService.GetDeploymentsRecordSingle(appId, *&clusterDetails.Region_code, statusToCheck)
	if getDeploymentErr != nil {
		return nil, getDeploymentErr
	} else if appDeployment.Id == "" {
		println("ConfigPath", clusterDetails.Cluster_config_path)

		var fileSavePath string
		if clusterDetails.Cluster_config_path == "" {
			k8sPath := "k8s_config/" + clusterDetails.Region_code
			err := os.Mkdir(k8sPath, 0755)
			if err != nil {
				return nil, err
			}

			fileSavePath = k8sPath + "/config"

			_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterDetails.ClusterConfigURL, fileSavePath)
			if err != nil {
				return nil, err
			}
			clusterDetails.Cluster_config_path = "./k8s_config/" + clusterDetails.Region_code + "/config"
		}

		clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)
		if err != nil {
			helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
			return &deployOutput, err
		}

		helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
		PrivateReg, err := secretregistry.GetSecretDetails(regId, "")

		if err != nil {
			return &deployOutput, err
		}
		if regId != "" {
			if *PrivateReg.RegistryType == "PAT" {
				regId = ""
			}
		}

		registryName := ""
		if regId != "" { // return &deployOutput, fmt.Errorf("Secret Registry Id should not be empty")

			PrivateRegistry, err := secretregistry.GetSecretDetails(regId, "")

			if err != nil {
				return &deployOutput, err
			}

			registryName = *PrivateRegistry.Name

			err = Secret(clientset, PrivateRegistry, "")
			if err != nil {
				return &deployOutput, err
			}
		} else {
			name := "nife123"
			pass := "9adcfa74a65e2c6b065a6eefe87eaef5"
			url := "https://index.docker.io/v1/"
			secretName := "nife"
			registry := model.GetUserSecret{
				UserName: &name,
				PassWord: &pass,
				URL:      &url,
				Name:     &secretName,
			}
			registryName = *registry.Name

			err = Secret(clientset, registry, orgSlug)
			if err != nil {
				return &deployOutput, err
			}

		}
		PrivateRegistry1, err := secretregistry.GetSecretDetails(regId, "")

		if err != nil {
			return &deployOutput, err
		}
		if regId == "" {
			PrivateRegistry1.RegistryType = nil
		}

		err = CreateConfigMap(clientset, PrivateRegistry1, orgSlug, appId)
		if err != nil {
			return &deployOutput, err
		}

		err = CreatePersistentVolume(clientset, appId, orgSlug, PrivateRegistry1)
		if err != nil {
			return &deployOutput, err
		}

		appReplicas, err := _helper.GetAppReplicas(appId)
		if err != nil {
			return nil, err
		}
		// Deploying pods using deployment object

		deploymentId, containerId, err := Deployment(orgSlug, clientset, internalPort, imageName, appId, registryName, envArgString, storage, regId, appReplicas)

		if err != nil {
			return &deployOutput, err
		}

		serviceName := "service-" + deploymentId
		// Creating service for pods using service object

		serviceName, elbURL, _, err := Services(clientset, internalPort, imageName, appId, serviceName, orgSlug, *&clusterDetails, userId, redeployBool)

		if err != nil {
			return &deployOutput, err
		}

		podsIp, err := clientset.CoreV1().Pods(orgSlug).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return &deployOutput, err
		}
		time.Sleep(5 * time.Second)

		var ipPod string
		for _, pod := range podsIp.Items {
			pod, _ := clientset.CoreV1().Pods(orgSlug).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			ipPod = pod.Status.PodIP
		}

		//if isdefault true then APP TABLE IMAGE NAME AND PORT UPDATE
		if clusterDetails.IsDefault == 1 {
			var hostName string
			if imageName == "mysql:5.6" || imageName == "postgres:10.1" {
				hostName = ipPod
			} else if clusterDetails.ClusterType == "byoh" {
				hostName = "http://" + elbURL
				hostname = elbURL
			} else {
				// hostName = fmt.Sprintf("http://%s", hostname)         // DNS route53
				hostName = fmt.Sprintf("https://%s", hostname)

			}

			err := UpdateApp(appId, imageName, fmt.Sprintf("%v", internalPort), hostName, envArgs, containerId)
			if err != nil {
				return &deployOutput, err
			}
		}

		// podsIp, err := clientset.CoreV1().Pods(orgSlug).List(context.TODO(), metav1.ListOptions{})
		// if err != nil {
		// 	return &deployOutput, err
		// }
		// time.Sleep(5 * time.Second)

		// var ipPod string
		// for _, pod := range podsIp.Items {
		// 	pod, _ := clientset.CoreV1().Pods(orgSlug).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		// 	ipPod = pod.Status.PodIP
		// }

		checkWorkLoad, err := CheckWorkloadForApp(appId)
		if err != nil {
			return &deployOutput, err
		}
		if imageName == "mysql:5.6" || imageName == "postgres:10.1" {
			//deployOutput.DeploymentName = deploymentName
			deployOutput.ServiceName = *PrivateRegistry1.Name
			deployOutput.HostName = *PrivateRegistry1.UserName
			deployOutput.URL = ipPod
			deployOutput.ID = deploymentId
			deployOutput.ContainerID = &containerId
			return &deployOutput, nil
		} else {
			//deployOutput.DeploymentName = deploymentName
			deployOutput.ServiceName = serviceName
			deployOutput.LoadBalanceURL = &elbURL
			deployOutput.HostName = hostname
			if checkWorkLoad != "" {
				// deployOutput.URL = fmt.Sprintf("http://%s", checkWorkLoad+"."+hostname)  //DNS route53
				if clusterDetails.ClusterType == "byoh" {
					deployOutput.URL = fmt.Sprintf("http://%s", hostname)
				} else {
					deployOutput.URL = fmt.Sprintf("https://%s", hostname)
				}
			} else {
				if clusterDetails.ClusterType == "byoh" {
					deployOutput.URL = fmt.Sprintf("http://%s", hostname)
				} else {
					deployOutput.URL = fmt.Sprintf("https://%s", hostname)
				}
			}
			deployOutput.ID = deploymentId
			deployOutput.ContainerID = &containerId
			return &deployOutput, nil
		}
	} else {
		exitDeployment := true
		return &model.DeployOutput{ExistDeployment: &exitDeployment, ReleaseID: &appDeployment.Release_id}, nil
	}
}

func UpdateApp(appId, imageName, port, hostname, envArgs, containerId string) error {
	statement, err := database.Db.Prepare("UPDATE app set imageName = ?, port = ?, hostname = ? , deployed = ? , envArgs = ? where name = ?")
	if err != nil {
		return err
	}

	if envArgs == "{}" {
		envArgs = ""
	}
	defer statement.Close()
	_, err = statement.Exec(imageName, port, hostname, true, envArgs, appId)
	if err != nil {
		return err
	}
	return nil
}

func UnDeploy(appId string, clusterDetails cluster_details.ClusterDetail, hostName string, org *model.Organization, userId string, redeployordelete bool) (*model.App, error) {

	statusToCheck := "running"
	getDeployment, getDeploymentErr := commonService.GetDeploymentsRecordSingle(appId, clusterDetails.Region_code, statusToCheck)
	if getDeploymentErr != nil {
		return nil, getDeploymentErr
	}

	println("UnDeploy ConfigPath", clusterDetails.Cluster_config_path)

	var fileSavePath string
	if clusterDetails.Cluster_config_path == "" {
		k8sPath := "k8s_config/" + clusterDetails.Region_code
		err := os.Mkdir(k8sPath, 0755)
		if err != nil {
			return nil, err
		}

		fileSavePath = k8sPath + "/config"

		_, err = organizationInfo.GetFileFromPrivateS3kubeconfigs(*clusterDetails.ClusterConfigURL, fileSavePath)
		if err != nil {
			return nil, err
		}
		clusterDetails.Cluster_config_path = "./k8s_config/" + clusterDetails.Region_code + "/config"
	}

	clientset, err := helper.LoadK8SConfig(clusterDetails.Cluster_config_path)

	if err != nil {
		helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)
		return &model.App{}, nil
	}
	helper.DeletedSourceFile("k8s_config/" + clusterDetails.Region_code)

	appDetails, err := GetApp(appId, userId)
	if err != nil {
		return &model.App{}, err
	}

	if *appDetails.ImageName != "mysql:5.6" && *appDetails.ImageName != "postgres:10.1" && clusterDetails.ClusterType != "byoh" {

		if len(appDetails.Regions) == 1 {
			// err = CreateOrDeleteDNSRecord(appId, getHostName(hostName), getDeployment.App_Url, clusterDetails.Region_code, *clusterDetails.ProviderType, true, userId)
			// if err != nil {
			// 	// return &model.App{}, err
			// 	fmt.Println("")
			// }

			_, err = DeleteCLBRoute(appId)
			if err != nil {
				// return nil, err
				fmt.Println("")
			}

		}
	}

	if clusterDetails.ClusterType == "byoh" {
		ingressName := appId + "-ingress"
		err = DeleteIngress(clientset, ingressName, *appDetails.Organization.Slug)
		if err != nil {
			return &model.App{}, err
		}
	}

	err = DeleteDeployment(clientset, getDeployment.Deployment_id, org, redeployordelete, appId)
	if err != nil {
		return &model.App{}, err
	}
	err = DeletePersistentVolumes(clientset, appId, *appDetails.Organization.Slug)
	if err != nil {
		fmt.Println(err)
	}

	deploymentUpdateErr := UpdateDeploymentsRecord("destroyed", appId, getDeployment.Deployment_id, time.Now())
	if deploymentUpdateErr != nil {
		return nil, deploymentUpdateErr
	}

	return &model.App{}, nil
}

func UpdateDeploymentsRecord(status, appId, deployment_id string, updatedAt time.Time) error {
	statement, err := database.Db.Prepare("UPDATE app_deployments set status = ?, updatedAt = ? where appId = ? and deployment_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(status, updatedAt, appId, deployment_id)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDeploymentELBRecord(recordName, recordId, id string) error {
	statement, err := database.Db.Prepare("UPDATE app_deployments set elb_record_name = ?, elb_record_id = ? where id = ?")
	if err != nil {
		return err
	}

	_, err = statement.Exec(recordName, recordId, id)
	if err != nil {

		return err
	}
	return nil
}

func CheckPortInUse(ports []int32, port int32) int32 {
	portInUse := false
	for _, v := range ports {
		if v == port {
			portInUse = true
			break
		}
	}
	if portInUse {
		portLower := int(port)
		portUpper := portLower + 50
		port = int32(portLower + rand.Intn(portUpper-portLower+1))
		CheckPortInUse(ports, port)
	}
	return port
}

func CreateOrDeleteDNSRecord(deploymentId, DNSRecordName, loadBalancerURL, regionCode, cloudType string, isDelete bool, userId string) error {

	regionCode = getCountryCode(regionCode)

	splitAppName := strings.Split(DNSRecordName, ".")[0]

	getAppDefinition, err := GetApp(splitAppName, userId)
	if err != nil {
		log.Println(err)
		return err
	}

	routingPolicy, err := _helper.GetRoutingPolicy(getAppDefinition.ParseConfig.Definition)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println(getAppDefinition.ParseConfig.Definition)
	recordName, _, recordId, err := awsService.CreateOrDeleteRecordSetRoute53(DNSRecordName, loadBalancerURL, regionCode, cloudType, isDelete, routingPolicy)

	UpdateDeploymentELBRecord(recordName, recordId, deploymentId)

	return err
}

func getHostName(fullHostname string) string {
	u, _ := url.Parse(fullHostname)
	return u.Hostname()
}

func getCountryCode(regionCode string) string {

	if regionCode == "IND" {
		return "AS"
	} else if regionCode == "EUR" || regionCode == "EUR-3" {
		return "EU"
	} else {
		return "NA"
	}
}

func GetElbUrlByAppName(appName string) (model.ElbURL, error) {
	statement, err := database.Db.Prepare("SELECT app_url FROM app_deployments where appId = ? and status = ?")
	if err != nil {
		log.Fatal(err)
	}
	row := statement.QueryRow(appName, "running")
	defer statement.Close()
	var Url model.ElbURL
	err = row.Scan(&Url.ElbURL)
	if err != nil {
		return model.ElbURL{}, err
	}

	return Url, nil
}

func DeletePersistentVolumes(clientset *kubernetes.Clientset, name, orgSlug string) error {

	persistentVolumeClient := clientset.CoreV1().PersistentVolumes()
	persistentVolumeClaimClient := clientset.CoreV1().PersistentVolumeClaims(orgSlug)
	err := persistentVolumeClient.Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = persistentVolumeClaimClient.Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

// func UpdateAppbySubOrgAndBusinessUnit(appName,subOrgId, businessUnitId string) error {
// 	statement, err := database.Db.Prepare("UPDATE app set sub_org_id = ?, business_unit_id = ? where name = ?;")
// 	if err != nil {
// 		return err
// 	}

// 	_, err = statement.Exec(subOrgId, businessUnitId, appName)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func CheckWorkloadForApp(appName string) (string, error) {

	query := `SELECT workload_management.environment_endpoint FROM workload_management 
	INNER JOIN app ON app.workload_management_id = workload_management.id 
	where app.name = ? `

	selDB, err := database.Db.Query(query, appName)
	if err != nil {
		return "", err
	}
	defer selDB.Close()
	var wlendPoint string

	for selDB.Next() {
		err = selDB.Scan(&wlendPoint)
		if err != nil {
			return "", err
		}
	}

	return wlendPoint, nil
}

func DeletePersistentVolume(clientset *kubernetes.Clientset, name, orgSlug string, PrivateRegistry model.GetUserSecret) error {
	var mySql string
	if PrivateRegistry.RegistryType == nil {
		mySql = ""
	} else {
		mySql = *PrivateRegistry.RegistryType
	}
	if mySql == "mysql" || mySql == "postgres" {

		persistentVolumeClient := clientset.CoreV1().PersistentVolumes()
		persistentVolumeClaimClient := clientset.CoreV1().PersistentVolumeClaims(orgSlug)
		err := persistentVolumeClient.Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}

		err = persistentVolumeClaimClient.Delete(context.TODO(), name, metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	err := DeleteSecrets(clientset, orgSlug, *PrivateRegistry.Name)
	if err != nil {
		return err
	}

	return nil
}

func DeleteIngress(clientset *kubernetes.Clientset, name, orgSlug string) error {

	err := clientset.NetworkingV1().Ingresses(orgSlug).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func DeleteVolume(appName string) error {
	statement, err := database.Db.Prepare("Delete from volumes where app_id = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(appName)
	if err != nil {
		return err
	}
	return nil
}

func CreateCLBRoute(appName, loadBalancerURL, externalPrt string, httpsEnForce bool) (string, error) {
	clbEndPoint := os.Getenv("HOST_NAME_CLB")

	source := appName + "." + clbEndPoint

	target := "http://" + loadBalancerURL + ":" + externalPrt
	clbUrl := os.Getenv("CLB_URL")
	// clbUrl := "http://clb2.nifetency.com:5555/api/routes/"

	postBody, _ := json.Marshal(map[string]interface{}{
		"source": source,
		"target": target,
		"settings": map[string]interface{}{
			"enforce_https": httpsEnForce,
		},
	})

	resp, err := http.NewRequest("POST", clbUrl, bytes.NewBuffer(postBody))
	if err != nil {
		return "", err
	}

	resp.Header.Add("Accept", "application/json")

	clt := http.Client{}

	res, err := clt.Do(resp)
	if err != nil {
		return "", err
	}
	var httpsMapping map[string]interface{}

	apiResponse, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(apiResponse, &httpsMapping)

	sourceUrl := httpsMapping["source"]

	sourceUrlString := fmt.Sprintf("%v", sourceUrl)

	return sourceUrlString, nil
}

func DeleteCLBRoute(appName string) (string, error) {

	clbUrl := os.Getenv("CLB_URL")
	clbEndPoint := os.Getenv("HOST_NAME_CLB")

	resp, err := http.NewRequest("DELETE", clbUrl+appName+"."+clbEndPoint+"/", &bytes.Buffer{})
	if err != nil {
		return "", err
	}

	resp.Header.Add("Accept", "application/json")

	clt1 := http.Client{}

	res, err := clt1.Do(resp)
	if err != nil {
		return "", err
	}
	return res.Status, nil
}

func UpdateDeploymentsTime(appName, time string) error {
	statement, err := database.Db.Prepare("UPDATE app set deployment_time = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(time, appName)
	if err != nil {
		return err
	}
	return nil
}

func UpdateBuildTime(appName, time string) error {
	statement, err := database.Db.Prepare("UPDATE app set build_time = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(time, appName)
	if err != nil {
		return err
	}
	return nil
}

func UpdateBuiLogsURL(appName, buildLogsUrl string) error {
	statement, err := database.Db.Prepare("UPDATE app set build_logs_url = ? where name = ?")
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.Exec(buildLogsUrl, appName)
	if err != nil {
		return err
	}
	return nil
}

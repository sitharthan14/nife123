package helper

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"strings"

	"github.com/nifetency/nife.io/internal/decode"
	secretregistry "github.com/nifetency/nife.io/internal/secret_registry"
	"golang.org/x/oauth2/google"

	// "github.com/nifetency/nife.io/service"

	//"unicode"

	"github.com/nifetency/nife.io/api/model"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

type envVariable struct {
	secret    string
	secetJson string
	env       string
	jsontype  string
	resource  string
	resJson   string
}

func ReadConfig() (*model.Config, error) {
	var config *model.Config
	dataBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(dataBytes, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func Int32Ptr(i int32) *int32 { return &i }

func getStringFlag(name string) string {
	return flag.Lookup(name).Value.(flag.Getter).Get().(string)
}

func LoadK8SConfig(configPath string) (*kubernetes.Clientset, error) {
	var kubeconfig *string
	kubeconfig = &configPath
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return new(kubernetes.Clientset), err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return new(kubernetes.Clientset), err
	}
	return clientset, err
}

func ReadK8SConfigPath(IP string) (string, string, error) {
	var config *model.Config
	dataBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return "", "", err
	}
	err = json.Unmarshal(dataBytes, &config)
	if err != nil {
		return "", "", err
	}

	for _, domain := range config.DomainMapping {
		if *domain.IPAddress == IP {
			return *domain.KubeConfigPath, *domain.NodeName, nil
		}
	}
	return "", "", fmt.Errorf("Config path not found for %s", IP)
}

func ReadDockerRegistry() (model.Registry, error) {
	var config model.Config
	dataBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		return *config.Registry, err
	}
	err = json.Unmarshal(dataBytes, &config)
	if err != nil {
		return *config.Registry, err
	}
	return *config.Registry, nil
}

func RespondwithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func AppNameCheckWithBlankSpace(appName string) error {
	string := strings.IndexByte(appName, ' ')
	if string != -1 {
		return fmt.Errorf("string contain empty space")
	}
	return nil
}

func FormatEnvArgsStrings(envMapArgs []*string) string {

	envMapString := make(map[string]string)
	for _, arg := range envMapArgs {
		parts := strings.Split(*arg, "=")
		if len(parts) != 2 {
			return ""
		}
		envMapString[parts[0]] = parts[1]

	}
	jsonString, err := json.Marshal(envMapString)
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	return string(jsonString)

}

func GlobalEnv(env []string) ([]string, error) {

	var envString []string

	for _, str := range env {
		checkstr := strings.Contains(str, "=")
		if checkstr {
			envString = append(envString, str)
		} else {
			globalEnv, err := secretregistry.GetSecretDetails("", str)
			if err != nil {
				return nil, err
			}

			if globalEnv.RegistryType == nil {
				return nil, fmt.Errorf("Invalid Gloabal Variable '%s': ", str)
			}
			password := decode.DePwdCode(*globalEnv.PassWord)
			envString = append(envString, *globalEnv.UserName+"="+password)
		}
	}
	return envString, nil
}

func EnvironmentArgument(envArgs string, interfaceType string, secret string, memoryResource model.Requirement) (string, error) {

	var environmentArgument string
	env := strings.Fields(envArgs)
	envMapString := make(map[string]string)
	envstring, err := GlobalEnv(env)
	if err != nil {
		return "", err
	}

	for _, i := range envstring {
		parts := strings.Split(i, "=")
		if len(parts) != 2 {
			return "", fmt.Errorf("Invalid env-arg '%s': must be in the format NAME=VALUE", i)
		}
		envMapString[parts[0]] = parts[1]
	}
	if interfaceType == "kube_config" {
		serializing, err := json.Marshal(envMapString)

		if err != nil {
			return "", err
		}
		environmentArgument = string(serializing)
	} else {

		envString, _ := json.Marshal(envMapString)
		jsonData := []byte(envString)

		var result map[string]string
		if err := json.Unmarshal(jsonData, &result); err != nil {
			return "", fmt.Errorf(err.Error())
		}

		var array []map[string]string
		for k := range result {
			var My_map = make(map[string]string)
			My_map["Name"] = k
			My_map["Value"] = result[k]
			array = append(array, My_map)
		}

		data, _ := json.Marshal(array)
		res := envVariable{
			env:      `"Env":`,
			jsontype: string(data),
		}

		if secret != "" {
			secretName := []map[string]interface{}{}
			secretElement := map[string]interface{}{
				"Name": secret,
			}

			secretName = append(secretName, secretElement)

			Data1, _ := json.Marshal(secretName)

			if envArgs != "" {
				res = envVariable{
					secret:    `"ImagePullSecrets":`,
					secetJson: string(Data1),
					env:       `,"Env":`,
					jsontype:  string(data),
				}
			} else {
				res = envVariable{
					secret:    `"ImagePullSecrets":`,
					secetJson: string(Data1),
				}
			}
		}

		if *memoryResource.LimitRequirement.CPU != "" {
			memstring := FormatMemoryResource(memoryResource)
			res.resource = `,"Resources":`
			res.resJson = memstring

		}

		jsonValue := fmt.Sprintf(`%s`, res)
		environmentArgument = jsonValue

	}

	return environmentArgument, nil
}

func CheckRequiredSecret(secName string, Definition map[string]interface{}) (bool, error) {
	secretName, _ := Definition["SecretName"].(interface{}).(string)
	if secName == secretName {
		return true, nil
	}
	return false, nil
}

func FormatMemoryResource(memoryResource model.Requirement) string {
	unit := os.Getenv("DEFAULT_MEMORY_UNIT")
	memoryres := map[string]interface{}{
		"Limits": map[string]interface{}{
			"cpu":    memoryResource.LimitRequirement.CPU,
			"memory": string(*memoryResource.LimitRequirement.Memory) + unit,
		},
		"Requests": map[string]interface{}{
			"cpu":    memoryResource.RequestRequirement.CPU,
			"memory": string(*memoryResource.RequestRequirement.Memory) + unit,
		},
	}
	m, _ := json.Marshal(memoryres)

	return string(m)
}

func WriteFileToTemp(path string) (os.File, error) {
	tempFile, err := os.Create(path)

	if err != nil {
		log.Println(err)
		return os.File{}, fmt.Errorf(err.Error())
	}

	return *tempFile, nil
}

func DeletedSourceFile(filePath string) error {
	err := os.RemoveAll(filePath)
	if err != nil {
		log.Println(err)
	}

	return err
}

func ReadFile() (google.Credentials, error) {
	data, err := ioutil.ReadFile("temp.json")
	if err != nil {
		fmt.Println("File reading error", err)
		return google.Credentials{}, fmt.Errorf(err.Error())
	}
	credentials, err := google.CredentialsFromJSON(context.Background(), data)
	if err != nil {
		fmt.Println(err)
		return google.Credentials{}, fmt.Errorf(err.Error())
	}

	return *credentials, nil
}

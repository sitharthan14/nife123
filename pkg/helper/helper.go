package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/nifetency/nife.io/api/model"
	database "github.com/nifetency/nife.io/internal/pkg/db/mysql"
)

// UnmarshallApp function for App List Mock Data
func UnmarshallToNode(filePath string) (*model.Nodes, error) {
	var node *model.Nodes

	dataBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return node, err
	}

	err = json.Unmarshal(dataBytes, &node)
	if err != nil {
		return node, err
	}
	return node, nil
}

// encrypt string to base64 crypto using AES
func Encrypt(key []byte, text string) string {
	// key := []byte(keyText)
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		log.Println(err)
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

// decrypt from base64 to decrypted string
func Decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		log.Println("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}

func GetInternalPort(definition map[string]interface{}) (int64, error) {
	services, ok := definition["services"].([]interface{})
	fmt.Println(services)
	fmt.Println(ok)
	if ok {
		service0 := services[0].(map[string]interface{})
		internalport, ok := service0["internal_port"]
		if ok {
			interfacePortToString := fmt.Sprintf("%v", internalport)
			port, err := strconv.Atoi(interfacePortToString)
			if err != nil {
				log.Println(err)
				return 0, err
			}
			return int64(port), nil

		}
	}
	return -1, errors.New("could not find internal port setting")
}

func GetExternalPort(definition map[string]interface{}) (int64, error) {
	services, ok := definition["services"].([]interface{})
	fmt.Println(services)
	fmt.Println(ok)
	if ok {
		service0 := services[0].(map[string]interface{})
		externalPort, ok := service0["external_port"]
		if ok {
			port, _ := externalPort.(float64)

			return int64(port), nil
		}
	}
	return -1, errors.New("could not find externalPort port setting")
}

func GetResourceRequirement(definition map[string]interface{}) model.Requirement {

	var res model.Requirement
	services, ok := definition["services"].([]interface{})
	if ok {
		service0 := services[0].(map[string]interface{})
		resRequest, ok := service0["requests"].(map[string]interface{})
		if ok {
			req, _ := json.Marshal(resRequest)
			json.Unmarshal(req, &res.RequestRequirement)

		}
		resLimit, ok := service0["limits"].(map[string]interface{})
		if ok {
			lim, _ := json.Marshal(resLimit)
			json.Unmarshal(lim, &res.LimitRequirement)
		}
	}

	return res
}

func SetInternalPort(definition map[string]interface{}, port int) map[string]interface{} {
	if services, ok := definition["services"].([]interface{}); ok {
		if len(services) == 0 {
			return nil
		}

		if service, ok := services[0].(map[string]interface{}); ok {
			service["internal_port"] = port
			return definition
		}
	}
	return nil
}

func SetExternalPort(definition map[string]interface{}, port int) map[string]interface{} {
	if services, ok := definition["services"].([]interface{}); ok {
		if len(services) == 0 {
			return nil
		}

		if service, ok := services[0].(map[string]interface{}); ok {
			service["external_port"] = port
			return definition
		}
	}
	return nil
}

func SetRoutingPolicy(definition map[string]interface{}, routingPolicy string) map[string]interface{} {
	if services, ok := definition["services"].([]interface{}); ok {
		if len(services) == 0 {
			return nil
		}

		if service, ok := services[0].(map[string]interface{}); ok {
			service["routing_policy"] = routingPolicy
			return definition
		}
	}
	return nil
}

func SetResourceRequirement(Definition map[string]interface{}, storagesize model.Requirement) map[string]interface{} {
	if services, ok := Definition["services"].([]interface{}); ok {
		if len(services) == 0 {
			return nil
		}

		if service, ok := services[0].(map[string]interface{}); ok {
			resRequest, ok := service["requests"].(map[string]interface{})
			if ok {
				resRequest["memory"] = storagesize.RequestRequirement.Memory
				resRequest["cpu"] = storagesize.RequestRequirement.CPU
			}
			resLimit, ok := service["limits"].(map[string]interface{})
			if ok {
				resLimit["memory"] = storagesize.LimitRequirement.Memory
				resLimit["cpu"] = storagesize.LimitRequirement.CPU
			}
		}
		return Definition
	}

	return nil
}

func GetRoutingPolicy(definition map[string]interface{}) (string, error) {
	services, ok := definition["services"].([]interface{})
	fmt.Println(services)
	fmt.Println(ok)
	if ok {
		service0 := services[0].(map[string]interface{})
		routing_policy, ok := service0["routing_policy"]
		if ok {
			policy := fmt.Sprintf("%v", routing_policy)

			return policy, nil
		}
	}
	return "", errors.New("could not find routingPolicy setting")
}

func UpdateEnvArgs(appName string, envArgs []*string) error {

	fmt.Println("working", envArgs)

	statement, err := database.Db.Prepare("Update app set envArgs = ? where name = ?")
	if err != nil {
		return err
	}

	var envstring []string

	for _, i := range envArgs {
		envstring = append(envstring, *i)
	}

	env := strings.Join(envstring, " ")

	_, err = statement.Exec(env, appName)
	if err != nil {
		return err
	}
	return nil
}


func GetAppsEnvArgs(appName, appVersion string) (string, string, error) {

	selDB, err := database.Db.Query("SELECT envArgs, builder_type FROM app_release where app_id = ? and version = ?;", appName, appVersion)
	if err != nil {
		return "", "", err
	}
	defer selDB.Close()

	var envArgs string
	var builderType string

	for selDB.Next() {
		err = selDB.Scan(&envArgs, &builderType)
		if err != nil {
			return "", "", err
		}
	}

	return envArgs, builderType, nil

}

func UpdateEnvArgsInRelease(appName string, envArgs []*string, version string) error {

	statement, err := database.Db.Prepare("Update app_release set envArgs = ? where app_id = ? and version = ?")
	if err != nil {
		return err
	}

	var envstring []string

	for _, i := range envArgs {
		envstring = append(envstring, *i)
	}

	env := strings.Join(envstring, " ")
	defer statement.Close()
	_, err = statement.Exec(env, appName, version)
	if err != nil {
		return err
	}
	return nil
}

func TemporaryPassword() (string, error) {
	lenOfString := 5
	b := make([]byte, lenOfString)
	if _, err := rand.Read(b); err != nil {
		return "", nil
	}
	tempPass := fmt.Sprintf("%X", b)
	return tempPass, nil
}

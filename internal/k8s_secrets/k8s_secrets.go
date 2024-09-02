package k8ssecrets

import (
	"context"
	"fmt"
	"log"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ListSecrets(clientset *kubernetes.Clientset, orgSlug string) (*apiv1.SecretList, error) {
	secretList, err := clientset.CoreV1().Secrets(orgSlug).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println(err.Error())
		return secretList, err
	}
	return secretList, nil
}


func CreateSecret(clientset *kubernetes.Clientset, orgSlug string) error {
	
	userName := "nife123"
	passWord := "Nife@2020"
	url := "https://index.docker.io/v1/"
	
	dockerCred := fmt.Sprintf(`{"auths":{"%s":{"username":"%s","password":"%s","email":"%s"}}}`, url, userName, passWord, "")
	fmt.Println(dockerCred)
	sec := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nife",
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


func Secret(clientset *kubernetes.Clientset, orgSlug string) error {

	secretList, err := ListSecrets(clientset, orgSlug)
	if err != nil {
		return err
	}

	for _, secret := range secretList.Items {
		if secret.GetObjectMeta().GetName() == "nife" {
			return nil
		}
	}
	
	err = CreateSecret(clientset, orgSlug)

	if err != nil {
		return err
	}
	return nil	
}
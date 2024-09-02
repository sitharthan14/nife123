package builtin

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nifetency/nife.io/pkg/helper"
	"github.com/nifetency/nife.io/service"
)

func GetBuiltIn() (map[string]Builtin, error) {
	builtins := make(map[string]Builtin)

	
		for _, rt := range Basicbuiltins {
			builtins[rt.Name] = rt
		}

	
	return builtins, fmt.Errorf("something went worng")
}

func GetAppDetails(appName, userId string) (int64, error) {
	appDetails, err := service.GetApp(appName, userId)
	if err != nil {
		log.Println(err)
		return 0, fmt.Errorf(err.Error())
	}

	internalPort, err := helper.GetInternalPort(appDetails.Config.Definition)
	if err != nil {
		log.Println(err)
		return 0, fmt.Errorf(err.Error())
	}

	return internalPort, err

}


func CreateDockerFile(internalPort int64, builtInFile Builtin, appName string)(string,error){

	if internalPort != 8080 {
		builtInFile.Template = strings.Replace(builtInFile.Template, "8080", fmt.Sprintf("%v", internalPort), -1)
	}

	getPath,err := FindPath("api/extracted_file/"+appName)

	if err != nil {
		log.Println(err)
		return "",err
	}

	filePath, _ := filepath.Abs("api/extracted_file/"+appName+"/"+getPath+"/Dockerfile")
	createfile, _ := os.Create(filePath)
	createfile.Close()
	err = ioutil.WriteFile(filePath, []byte(builtInFile.Template), 0644)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return "", err
}

func FindPath(filePath string)(string,error){
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	files, err := f.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
    var pathName string
	for _, v := range files {
		pathName = v.Name()
	}

	return pathName,err
}
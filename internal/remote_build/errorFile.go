package remoteBuild

import (	
	"log"
	"os"
	"path"
)

func DeletedSourceFile(filePath string)(error){
	err := os.RemoveAll(filePath)
	if err != nil {
		log.Println(err)
	}
	file := path.Base(filePath)
    err = os.Remove(file)
	if err != nil {
		log.Println(err)
	}
	return err
}
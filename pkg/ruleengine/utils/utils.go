package utils

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

func GetFileContent(filePath string) ([]byte, error) {
	var byteValue []byte
	var err error

	// Open the file
	fileHandler, err := os.Open(filePath)

	if err == nil {
		// Defer the closing of the file.
		defer closeFile(fileHandler)

		byteValue, err = ioutil.ReadAll(fileHandler)
	} else {
		log.Println("Unable to open the file.")
	}

	return byteValue, err
}

func closeFile(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Println("Unable to close the file.")
	}
}

func GetJsonMap(jsonStr string) (map[string]interface{}, error) {
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(jsonStr), &jsonMap)
	return jsonMap, err
}

func GetKeyValue(props map[string]interface{}, keyName string) interface{} {
	keyValue, ok := props[keyName]
	if !ok {
		return nil
	}
	return keyValue
}

func FileExists(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	} else {
		return false
	}
}

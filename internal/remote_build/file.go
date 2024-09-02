package remoteBuild

import (
	"fmt"
	"github.com/docker/docker/pkg/archive"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func GetFileFromS3(SourceURL string) []byte {
	url := SourceURL
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	reader, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return reader
}

func FindFile(path string) (string, error) {

	f, err := os.Open(path)
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

	filePath := path + "/" + pathName
	f, err = os.Open(filePath)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	files, err = f.Readdir(0)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	for _, v := range files {
		if string(v.Name()) == "Dockerfile" {
			filePath = path + "/" + pathName
			return filePath, err
		} else {
			filePath = ""
		}
	}

	if filePath == "" {
		filePath = "Docker file doesn't exists"
	}

	return filePath, err
}

type buildContext struct {
	workingDir string
}

func NewBuildContext() (*buildContext, error) {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	return &buildContext{workingDir: tempDir}, nil
}

func (b *buildContext) Close() error {
	return os.RemoveAll(b.workingDir)
}

func (b *buildContext) Archive() (*archive.TempArchive, error) {

	reader, err := archive.Tar(b.workingDir, archive.Gzip)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return archive.NewTempArchive(reader, "")
}

func Copy_folder(source string, dest string) (err error) {

	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			err = Copy_folder(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = copy_file(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return
}

func copy_file(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			os.Chmod(dest, sourceinfo.Mode())
		}
	}

	return
}

func (b *buildContext) AddSourceFolder(source string, destination string) string {
	Copy_folder(source, b.workingDir)
	return ""
}

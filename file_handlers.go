package servermanager

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func tryMoveFile(sourcePath string, destPath string) error {	
	sourceExists, err := pathExists(sourcePath)
	if err != nil {
		logrus.Errorf("tryMoveFile() 14: %s", err)
		return err
	}

	destExists, err := pathExists(destPath)
	if err != nil {
		logrus.Errorf("tryMoveFile() 20: %s", err)
		return err
	}

	if sourceExists && !destExists {
		isSourceDir, err := isDirectory(sourcePath)

		if err == nil {
			if !isSourceDir {
				err = os.MkdirAll(filepath.Dir(destPath), 0755)
			} 
		} else {
			logrus.Errorf("tryMoveFile() 32: %s", err)
			return err
		}

		if err == nil{
			if !isSourceDir {
			err = os.Rename(sourcePath, destPath)
			} else {
			err = rMoveFiles(sourcePath, destPath)
			}

			if err != nil{
				logrus.Errorf("tryMoveFile() 44: %s", err)
				return err
			} else {
				logrus.Infof("JSON migration : moved %s to %s", sourcePath, destPath)
			}
		} else {
			logrus.Errorf("tryMoveFile() 50: %s", err)
			return err
		}
	}
	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { 
		return true, nil 
	}

	if os.IsNotExist(err) { 
		return false, nil 
	} else {
		logrus.Errorf("pathExists: %s", err)
	}

	return true, err
}

func isDirectory(path string) (bool, error) {
	exists, err := pathExists(path)
	if err != nil{
		logrus.Errorf("isDirectory() 75: %s", err)
		return false, err
	}

	if exists {
		fileInfo, err := os.Stat(path)
		if err != nil{
			logrus.Errorf("isDirectory() 82: %s", err)
			return false, err
		}
		return fileInfo.IsDir(), err
	}
	return false, nil
}

func getFileList(dir string) ([]os.FileInfo, error) {
	isDir, err := isDirectory(dir)
	if err != nil || !isDir{
		logrus.Errorf("listFiles() 93: %s", err)
		return nil, err
	}

	files, err := ioutil.ReadDir(dir)
	var list []os.FileInfo

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		list = append(list, file)
	}

	return list, nil
}

func rMoveFiles(sourceDir string, destinationDir string) error {	
	isDir, err := isDirectory(sourceDir)
	if err != nil || !isDir{
		logrus.Errorf("rMoveFiles() 114: %s", err)
		return err
	}

	filesToMove, err  := getFileList(sourceDir)

	if err != nil{
		logrus.Errorf("rMoveFiles() 121: %s", err)
		return err
	}

	destExists, err := pathExists(destinationDir)

	if err != nil{
		logrus.Errorf("rMoveFiles() 128: %s", err)
		return err
	}

	if !destExists {
		err = os.MkdirAll(destinationDir, 0755)
	}

	if err != nil{
		logrus.Errorf("rMoveFiles() 137: %s", err)
		return err
	}

	for _, file := range filesToMove {
		if file.IsDir() {
			continue
		}
		err = os.Rename(filepath.Join(sourceDir,file.Name()), filepath.Join(destinationDir,file.Name()))

		if err != nil{
			logrus.Errorf("rMoveFiles() 148: %s", err)
		}
	}

	os.Remove(sourceDir)
	return nil
}
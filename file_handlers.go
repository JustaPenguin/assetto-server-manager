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
		logrus.Errorf("couldn't find path %s", sourcePath)
		return err
	}

	destExists, err := pathExists(destPath)
	if err != nil {
		logrus.Errorf("couldn't find path %s", destPath)
		return err
	}

	if sourceExists && !destExists {
		isSourceDir, err := isDirectory(sourcePath)

		if err == nil {
			if !isSourceDir {
				err = os.MkdirAll(filepath.Dir(destPath), 0755)
			}
		} else {
			logrus.Errorf("couldn't check path %s", sourcePath)
			return err
		}

		if err == nil {
			if !isSourceDir {
				err = os.Rename(sourcePath, destPath)
			} else {
				err = rMoveFiles(sourcePath, destPath)
			}

			if err != nil {
				logrus.Errorf("couldn't move files from %s to %s", sourcePath, destPath)
				return err
			} else {
				logrus.Infof("JSON migration : moved %s to %s", sourcePath, destPath)
			}
		} else {
			logrus.Errorf("couldn't create directory %s", destPath)
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
		logrus.Errorf("couldn't check for path %s", path)
	}

	return true, err
}

func isDirectory(path string) (bool, error) {
	exists, err := pathExists(path)
	if err != nil {
		logrus.Errorf("couldn't check for path %s", path)
		return false, err
	}

	if exists {
		fileInfo, err := os.Stat(path)
		if err != nil {
			logrus.Errorf("couldn't check for path %s", path)
			return false, err
		}
		return fileInfo.IsDir(), err
	}
	return false, nil
}

func getFileList(dir string) ([]os.FileInfo, error) {
	isDir, err := isDirectory(dir)

	if err != nil {
		logrus.Errorf("trying to get a file list from %s, but it is not a directory", dir)
		return nil, err
	}

	if !isDir {
		return nil, nil
	}

	files, err := ioutil.ReadDir(dir)
	var list []os.FileInfo

	if err != nil {
		logrus.Errorf("couldn't get file list from: %s", dir)
		return nil, err
	}

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
	if err != nil || !isDir {
		logrus.Errorf("trying to move files from %s, but it is not a directory", sourceDir)
		return err
	}

	filesToMove, err := getFileList(sourceDir)

	if err != nil {
		logrus.Errorf("couldn't get file list from: %s", sourceDir)
		return err
	}

	destExists, err := pathExists(destinationDir)

	if err != nil {
		logrus.Errorf("couldn't check for destination directory: %s", sourceDir)
		return err
	}

	if !destExists {
		err = os.MkdirAll(destinationDir, 0755)
	}

	if err != nil {
		logrus.Errorf("couldn't create destination directory: %s", destinationDir)
		return err
	}

	for _, file := range filesToMove {
		if file.IsDir() {
			continue
		}
		err = os.Rename(filepath.Join(sourceDir, file.Name()), filepath.Join(destinationDir, file.Name()))

		if err != nil {
			logrus.Errorf("couldn't move %s to %s", filepath.Join(sourceDir, file.Name()), filepath.Join(destinationDir, file.Name()))
		}
	}

	err = os.Remove(sourceDir)

	if err != nil {
		logrus.Errorf("couldn't delete %s", sourceDir)
	}

	return nil
}

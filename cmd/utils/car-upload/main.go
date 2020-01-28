package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	servermanager "github.com/JustaPenguin/assetto-server-manager"

	"golang.org/x/net/publicsuffix"
)

var (
	baseURL   string
	dir       string
	maxUpload int

	username, password string
)

func init() {
	flag.StringVar(&dir, "dir", "", "cars directory to upload")
	flag.StringVar(&baseURL, "url", "", "server manager base url")
	flag.StringVar(&username, "username", "", "")
	flag.StringVar(&password, "password", "", "")
	flag.IntVar(&maxUpload, "max", 30, "max number of cars to upload at once")
	flag.Parse()
}

func main() {
	var files []servermanager.ContentFile

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)

		if err != nil {
			return err
		}

		uploadFile := false

		switch info.Name() {
		case "data.acd", "tyres.ini", "ui_car.json", "ui_skin.json":
			uploadFile = true
		default:
			if strings.HasPrefix(info.Name(), "livery.") || strings.HasPrefix(info.Name(), "preview.") {
				uploadFile = true
			}
		}

		if !uploadFile {
			return nil
		}

		data, err := ioutil.ReadFile(path)

		if err != nil {
			return err
		}

		files = append(files, servermanager.ContentFile{
			Name:     info.Name(),
			FilePath: rel,
			Data:     base64.StdEncoding.EncodeToString(data),
		})

		if len(files) >= maxUpload {
			// do the upload request and clear out files
			err := uploadCars(files)

			if err != nil {
				return err
			}

			log.Printf("Successfully uploaded %d files", len(files))

			files = []servermanager.ContentFile{}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	if len(files) > 0 {
		// do the upload request and clear out files
		err := uploadCars(files)

		if err != nil {
			panic(err)
		}

		log.Printf("Successfully uploaded %d files", len(files))

		files = []servermanager.ContentFile{}
	}
}

func uploadCars(files []servermanager.ContentFile) error {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return err
	}

	client := &http.Client{
		Jar: jar,
	}

	form := url.Values{}
	form.Add("Username", username)
	form.Add("Password", password)

	// do a login
	resp, err := client.PostForm(baseURL+"/login", form)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body := new(bytes.Buffer)

	err = json.NewEncoder(body).Encode(files)

	if err != nil {
		return err
	}

	// now upload the files
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/car/upload", body)

	if err != nil {
		return err
	}

	uploadResp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer uploadResp.Body.Close()

	return nil
}

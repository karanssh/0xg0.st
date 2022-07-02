package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

// Upload a file, save and attribute a hash
func upload(w http.ResponseWriter, r *http.Request) {
	glog.Info("Request recieved")

	if !strings.Contains(r.Header.Get("Content-type"), "multipart/form-data") {
		glog.Error(`Bad request. Content-type should be "multipart/form-data."`)

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `Bad request. Content-type should be "multipart/form-data"`)
		return
	}

	var uuid string = GenerateUUID()
	var path string = fmt.Sprintf("./storage/%s/", uuid)

	// Prepare to get the file
	file, header, err := r.FormFile("file")
	defer func() {
		file.Close()
		glog.Infof(`File "%s" closed.`, header.Filename)
	}()
	if err != nil {
		glog.Errorf("Error retrieving file.")
		glog.Errorf("Error: %s", err.Error())

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request. Error retrieving file.")
		return
	}

	// Creates directory with UUID
	_, err = os.Stat(path)
	for !os.IsNotExist(err) {
		uuid = GenerateUUID()
		path := fmt.Sprintf("./storage/%s/", uuid)
		_, err = os.Stat(path)
	}

	if err := os.Mkdir(path, 0777); err != nil {
		glog.Error("Error saving file on server...")
		glog.Errorf("Error: %s", err.Error())

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "No storage available.")
		return
	}

	// Build and Write the file.
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		glog.Errorf("Content not readable.")
		glog.Errorf("Error: %s", err.Error())

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error. Content not readable.")
		return
	}
	err = os.WriteFile(path+header.Filename, bytes, 0777)
	if err != nil {
		glog.Errorf("Error writing file.")
		glog.Errorf("Error: %s", err.Error())

		w.WriteHeader(http.StatusInsufficientStorage)
		fmt.Fprintf(w, "Insufficient Storage. Error storing file.")
		return
	}

	// All good
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK, Successfully Uploaded\n http://%s/%s\n", r.Host, uuid)
}

// Gets the file using the provided UUID on the URL
func getFile(w http.ResponseWriter, r *http.Request) {
	//We will get under path and storage the only file that will be inside and return it to the client
	var uuid string

	// Maybe try a strings.Replace(r.URL.Path[1:], "/", "", -1)?
	var re = regexp.MustCompile(`(?m)[^\/]+$`)
	for _, match := range re.FindAllString(r.URL.Path, -1) {
		uuid = match
	}
	path := fmt.Sprintf("./storage/%s/", uuid)

	glog.Infof(`Route "%s"`, r.URL.Path)
	glog.Infof(`Retrieving UUID "%s"`, uuid)
	glog.Infof(`Retrieving Path "%s"`, path)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		glog.Errorf(`Error walking filepath "%s"`, path)
		glog.Errorf("Error: %s", err.Error())
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "File Not Found.")
		return
	}

	if len(files) <= 0 {
		glog.Errorf(`No files in directory "%s"`, path)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "File Not Found.")
		return
	}

	var filename = files[0].Name()
	glog.Infof(`Retrieving Filename "%s"`, fmt.Sprintf("./%s", filename))

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	http.ServeFile(w, r, fmt.Sprintf("./%s/%s", path, filename))
}

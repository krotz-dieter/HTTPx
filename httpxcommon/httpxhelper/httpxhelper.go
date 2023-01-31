package httpxhelper

import (
	"errors"
	"httpxcommon/multiparts"
	"httpxcommon/partscommon"
	"httpxcommon/singleparts"
	"mime"
	"net/http"
	"time"

	"k8s.io/klog"
)

func SendFiles(client *http.Client, url string, directory string, chunking string, mode string) (error, uint64, time.Duration) {
	// check chunking
	klog.V(partscommon.KlogDebug).Infof("Chunking mode for store:" + chunking)
	var size uint64
	var duration time.Duration
	if chunking == "single" {
		// upload files (for each file one POST)
		var sf singleparts.SinglepartFiles
		var errHandle error
		if mode == "async" {
			errHandle, size, duration = sf.AsyncPostFilesFromDirectory(client, url, directory)
		} else {
			errHandle, size, duration = sf.SyncPostFilesFromDirectory(client, url, directory)
		}
		if errHandle != nil {
			klog.Error(errHandle)
			return errHandle, size, duration
		}
	} else if chunking == "multi" {
		// upload files (one POST with all files in the body as multipart)
		var mf multiparts.MultipartFiles
		var errHandle error
		errHandle, size, duration = mf.PostFilesFromDirectory(client, url, directory)
		if errHandle != nil {
			klog.Error(errHandle)
			return errHandle, size, duration
		}
	} else {
		panic("Wrong chunking used! Please use single or multi")
	}
	return nil, size, duration
}

func RetrieveFiles(client *http.Client, url string, directory string) (error, uint64, time.Duration) {
	// keep start time
	var size uint64
	s := time.Now()
	// check chunking
	klog.V(partscommon.KlogDebug).Infof("Start the retrieve from url:" + url)

	// create a HTTP GET request
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		klog.Error("Error in creating GET request")
	}
	partscommon.LogRequest(r)
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	partscommon.LogResponse(res)

	// determine type and params
	contentType, params, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	switch contentType {

	case "multipart/related":
		{
			// save as multipart
			var mf multiparts.MultipartFiles
			var errHandle error
			errHandle, size = mf.SaveFilesFromResponse(res, directory, url)
			if errHandle != nil {
				klog.Error(errHandle)
				return errHandle, size, 0
			}
		}

	case "application/dicom":
		{
			// store singlepart message
			var sf singleparts.SinglepartFiles
			_, size = sf.StoreSinglePartMessage(&res.Header, &res.Body, directory, params)
		}
	default:
		{
			klog.Error("Content-Type is wrong")
			return errors.New("No correct content-type provided !"), 0, 0
		}
	}
	duration := time.Since(s)
	return nil, size, duration
}

// log requests
func LogRequest(r *http.Request) {
	partscommon.LogRequest(r)
}

// log requests
func LogResponse(r *http.Response) {
	partscommon.LogResponse(r)
}

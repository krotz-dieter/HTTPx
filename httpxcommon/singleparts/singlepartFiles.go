package singleparts

import (
	"bytes"
	"errors"
	"fmt"
	"httpxcommon/partscommon"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"k8s.io/klog"
)

// Type representing http3 handling
type SinglepartFiles struct{}

// get part file name
func (h *SinglepartFiles) GetSinglePartFileName(p *http.Header) string {
	_, params, _ := mime.ParseMediaType(p.Get("Content-Disposition"))
	filename := params["filename"]
	return filename
}

func (h *SinglepartFiles) StoreSinglePartMessage(header *http.Header, body *io.ReadCloser, dir string, params map[string]string) (int, uint64) {
	// keep start time
	s := time.Now()

	// process single part
	originalfilename := h.GetSinglePartFileName(header)
	klog.V(partscommon.KlogInfo).Info("Single Part file name from header: ", originalfilename)
	studyinstanceuid, seriesinstanceuid, sopinstanceuid := partscommon.GetDICOMInfo(originalfilename)
	if len(studyinstanceuid) == 0 || len(seriesinstanceuid) == 0 || len(sopinstanceuid) == 0 {
		klog.Error("No valid original file name in header")
		return http.StatusNoContent, 0
	}

	// create directories and calculate filename
	filename := partscommon.EnsureFilePath(dir, studyinstanceuid, seriesinstanceuid, sopinstanceuid)
	klog.V(partscommon.KlogInfo).Info("Target file name: ", filename)

	// create blank file
	file, errFile := os.Create(filename)
	if errFile != nil {
		klog.Error(errFile)
		return http.StatusNotFound, 0
	}
	defer file.Close()

	// copy part to file
	size, errCopy := io.Copy(file, *body)
	if errCopy != nil {
		klog.Error(errCopy)
		return http.StatusConflict, uint64(size)
	}
	klog.V(partscommon.KlogInfo).Infoln("Data copied into file:", filename, " with size:", size, " and time taken:", time.Since(s), " goroutine:", partscommon.GetGID())
	return http.StatusOK, uint64(size)
}

func ReadFileAndPost(client *http.Client, url string, path string) (error, uint64, time.Duration, time.Duration) {
	// read file
	sFile1 := time.Now()
	file, errOpen := os.Open(path)
	if errOpen != nil {
		klog.Error("error reading file", path)
		return errOpen, 0, 0, 0
	}
	fileContents, errRead := ioutil.ReadAll(file)
	if errRead != nil {
		klog.Error("error reading all from file:", path, " error:", errRead)
		return errRead, 0, 0, 0
	}
	defer file.Close()
	duration1 := time.Since(sFile1)

	// get DICOM info
	studyinstanceuid, seriesinstanceuid, sopinstanceuid := partscommon.GetDICOMInfo(path)
	if len(studyinstanceuid) == 0 || len(seriesinstanceuid) == 0 || len(sopinstanceuid) == 0 {
		klog.Error("No correct structure for file:", path)
		return errors.New("No correct structure for file"), 0, 0, 0
	}

	// Create a HTTP post request
	sFile2 := time.Now()
	r, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(fileContents))
	if err != nil {
		klog.Error("Error in creating POST request")
		return err, 0, 0, 0
	}
	r.Header.Add("Content-Type", "application/dicom")
	lenBody := len(fileContents)
	r.Header.Add("Content-Length", strconv.Itoa(lenBody))
	joinedpath := filepath.Join(studyinstanceuid, seriesinstanceuid, sopinstanceuid)
	klog.V(partscommon.KlogInfo).Info("Joined filename path:", joinedpath)
	r.Header.Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", joinedpath))
	partscommon.LogRequest(r)
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	partscommon.LogResponse(res)
	duration2 := time.Since(sFile2)

	klog.V(partscommon.KlogInfo).Info("GID:", partscommon.GetGID(), " file:", path, " length:", lenBody, " fileI/O:", duration1, " POST:", duration2, " result:", res.StatusCode)
	return nil, uint64(lenBody), duration1, duration2
}

func (h *SinglepartFiles) SyncPostFilesFromDirectory(client *http.Client, url string, directory string) (error, uint64, time.Duration) {
	// build path
	path := directory
	klog.V(partscommon.KlogDebug).Info("Processing directory: ", path)
	s := time.Now()
	var dur1, dur2 time.Duration
	var size uint64

	// walk through all files
	errWalk := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			klog.V(partscommon.KlogDebug).Info("No files in directory: ", path, err)
			return err
		}

		// process every file
		if !info.IsDir() {
			// read file and post content
			err, nrBytes, duration1, duration2 := ReadFileAndPost(client, url, path)
			if err != nil {
				klog.Error("Error in reading file: ", err)
				return err
			}
			dur1 += duration1
			dur2 += duration2
			size += nrBytes
		}
		return nil
	})
	if errWalk != nil {
		klog.V(partscommon.KlogDebug).Info("No files in directory: ", path)
		return errWalk, 0, 0
	}
	total := time.Since(s)
	klog.V(partscommon.KlogDebug).Info("Time total taken: ", total, " GID:", partscommon.GetGID(), " fileI/O:", dur1, " POST:", dur2)
	return nil, size, total
}

type FileResult struct {
	mu        sync.Mutex
	size      uint64
	nr        uint64
	duration1 time.Duration
	duration2 time.Duration
}

func FileWorker(client *http.Client, url string, id int, files <-chan string, done chan<- bool, result *FileResult) {
	klog.V(2).Info(" WORKER:", id, " goroutine:", partscommon.GetGID())
	for {
		file, more := <-files
		if more {
			klog.V(2).Info(" WORKER:", id, " start file:", file, " goroutine:", partscommon.GetGID())
			// read file and post content
			err, nrBytes, duration1, duration2 := ReadFileAndPost(client, url, file)
			if err != nil {
				klog.Error("Error in reading file: ", err)
			}
			result.mu.Lock()
			result.size += nrBytes
			result.nr += 1
			result.duration1 += duration1
			result.duration2 += duration2
			result.mu.Unlock()
			klog.V(2).Info(" WORKER:", id, " end   file:", file, " goroutine:", partscommon.GetGID(), " size:", nrBytes)
		} else {
			done <- true
			return
		}
	}
}

func (h *SinglepartFiles) AsyncPostFilesFromDirectory(client *http.Client, url string, directory string) (error, uint64, time.Duration) {
	// build path
	path := directory
	klog.V(partscommon.KlogDebug).Info("Processing directory: ", path)
	s := time.Now()

	// create channels and a number of workers
	numWorkers := runtime.NumCPU()
	files := make(chan string)
	done := make(chan bool, numWorkers)
	result := FileResult{
		size: 0, nr: 0, duration1: 0, duration2: 0,
	}

	// create workers
	klog.V(2).Info("Create ", numWorkers, " workers")
	for w := 1; w <= numWorkers; w++ {
		go FileWorker(client, url, w, files, done, &result)
	}

	// walk through all files
	errWalk := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			klog.V(2).Info("No files in directory: ", path, err)
			return err
		}
		// process every file
		if !info.IsDir() {
			files <- path
		}
		return nil
	})
	if errWalk != nil {
		klog.V(2).Info("No files in directory: ", path)
		return errWalk, 0, 0
	}

	// close the channel
	close(files)
	// wait for done of each worker
	for w := 1; w <= numWorkers; w++ {
		<-done
	}
	// close(results)
	total := time.Since(s)
	klog.V(partscommon.KlogDebug).Info("ASYNC Time total taken: ", total, " GID:", partscommon.GetGID(), " total size:", partscommon.ByteCountSI(result.size), " files:", result.nr, " I/O:", result.duration1, " POST:", result.duration2)
	close(done)
	return nil, result.size, total
}

func (h *SinglepartFiles) UploadFile(w http.ResponseWriter, filename string) (error, uint64) {
	// read file
	file, err := os.Open(filename)
	if err != nil {
		klog.Error("error reading file", filename)
		return err, 0
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return err, 0
	}
	defer file.Close()

	// write to body
	var n int
	if n, err = w.Write(fileContents); err != nil {
		return err, uint64(n)
	}
	return nil, uint64(n)
}

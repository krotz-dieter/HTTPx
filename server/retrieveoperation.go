package main

import (
	"fmt"
	"httpxcommon/multiparts"
	"httpxcommon/partscommon"
	"httpxcommon/singleparts"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog"
)

// Type representing retrieve of an study
type RetrieveOperation struct {
	dirIn string
}

// retrieve transaction on study level
func (h *RetrieveOperation) RetrieveStudy(w http.ResponseWriter, r *http.Request) {
	// read body part
	s := time.Now()
	body, err1 := io.ReadAll(r.Body)
	if err1 != nil {
		klog.V(partscommon.KlogDebug).Info("Error reading body while handling /studies: %s\n", err1.Error())
	}

	// dump message
	partscommon.LogRequest(r)

	// get the variables
	vars := mux.Vars(r)
	studyinstanceuid := vars["study"]
	if len(studyinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for study:", studyinstanceuid)
	} else {
		klog.Error("No study instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// load files from directory
	err, size := h.ProcessStudy(w, studyinstanceuid)
	if err != nil {
		klog.Error("Error processing study:", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Write(body)
	duration := time.Since(s)
	partscommon.LogTotalTimeInfo("RETRIEVE STUDY "+studyinstanceuid, duration, size, duration, false)
}

// retrieve transaction on series level
func (h *RetrieveOperation) RetrieveSeries(w http.ResponseWriter, r *http.Request) {
	// read body part
	s := time.Now()
	body, err1 := io.ReadAll(r.Body)
	if err1 != nil {
		klog.V(partscommon.KlogDebug).Info("Error reading body while handling /studies: %s\n", err1.Error())
	}

	// dump message
	partscommon.LogRequest(r)

	// get the variables
	vars := mux.Vars(r)
	studyinstanceuid := vars["study"]
	if len(studyinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for study:", studyinstanceuid)
	} else {
		klog.Error("No study instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	seriesinstanceuid := vars["series"]
	if len(seriesinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for series:", seriesinstanceuid)
	} else {
		klog.Error("No series instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// load files from directory
	err, size := h.ProcessSeries(w, studyinstanceuid, seriesinstanceuid)
	if err != nil {
		klog.Error("Error processing study:", err)
	}
	w.Write(body)
	duration := time.Since(s)
	partscommon.LogTotalTimeInfo("RETRIEVE SERIES "+studyinstanceuid+"/"+seriesinstanceuid, duration, size, duration, false)
}

// retrieve transaction on series level
func (h *RetrieveOperation) RetrieveInstance(w http.ResponseWriter, r *http.Request) {
	// dump message
	s := time.Now()
	partscommon.LogRequest(r)

	// get the variables
	vars := mux.Vars(r)
	studyinstanceuid := vars["study"]
	if len(studyinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for study:", studyinstanceuid)
	} else {
		klog.Error("No study instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	seriesinstanceuid := vars["series"]
	if len(seriesinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for series:", seriesinstanceuid)
	} else {
		klog.Error("No series instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sopinstanceuid := vars["instance"]
	if len(sopinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Retrieve requested for instance:", sopinstanceuid)
	} else {
		klog.Error("No sop instance uid provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// load files from directory
	err, size := h.ProcessInstance(w, studyinstanceuid, seriesinstanceuid, sopinstanceuid)
	if err != nil {
		klog.Error("Error reading instance: %s\n", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	duration := time.Since(s)
	partscommon.LogTotalTimeInfo("RETRIEVE INSTANCE "+studyinstanceuid+"/"+seriesinstanceuid+"/"+sopinstanceuid, duration, size, duration, false)
}

func (h *RetrieveOperation) ProcessStudy(w http.ResponseWriter, study string) (error, uint64) {
	//global header
	var mf multiparts.MultipartFiles
	ct := fmt.Sprintf("multipart/related; boundary=%q; type=\"application/dicom\"", mf.GetBoundary())
	klog.V(partscommon.KlogDebug).Info("Setting Content-Type to ", ct)
	w.Header().Set("Content-Type", ct)

	// build path
	path := filepath.Join(h.dirIn, study)

	// upload files
	err, size := mf.UploadFilesFromDirectory(w, path)
	if err != nil {
		klog.Error("Error uploading files:", err)
		return err, size
	}
	return nil, size
}

func (h *RetrieveOperation) ProcessSeries(w http.ResponseWriter, study string, series string) (error, uint64) {
	//global header
	var mf multiparts.MultipartFiles
	ct := fmt.Sprintf("multipart/related; boundary=%q; type=\"application/dicom\"", mf.GetBoundary())
	klog.V(partscommon.KlogDebug).Info("Setting Content-Type to ", ct)
	w.Header().Set("Content-Type", ct)

	// build path
	path := filepath.Join(h.dirIn, study, series)

	// upload files
	err, size := mf.UploadFilesFromDirectory(w, path)
	if err != nil {
		klog.Error("Error uploading files:", err)
		return err, size
	}
	return nil, size
}

func (h *RetrieveOperation) ProcessInstance(w http.ResponseWriter, study string, series string, instance string) (error, uint64) {
	// global header for single part message
	var sf singleparts.SinglepartFiles
	w.Header().Set("Content-Type", "application/dicom")
	joinedpath := filepath.Join(study, series, instance)
	klog.V(partscommon.KlogInfo).Info("Joined filename path:", joinedpath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", joinedpath))

	// build path
	path := filepath.Join(h.dirIn, study, series, instance+".dcm")

	// upload files
	err, size := sf.UploadFile(w, path)
	if err != nil {
		klog.Error("Error uploading file:", err)
		return err, size
	}
	return nil, size
}

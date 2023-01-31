package main

import (
	"httpxcommon/httpxhelper"
	"httpxcommon/multiparts"
	"httpxcommon/partscommon"
	"httpxcommon/singleparts"
	"mime"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog"
)

// Type representing store of an study
type StoreOperation struct {
	dirOut string
}

// store transaction on study level
func (h *StoreOperation) StoreStudy(w http.ResponseWriter, r *http.Request) {
	// dump message
	s := time.Now()
	httpxhelper.LogRequest(r)

	// get the variables
	vars := mux.Vars(r)
	studyinstanceuid := vars["study"]
	if len(studyinstanceuid) > 0 {
		klog.V(partscommon.KlogDebug).Info("Store requested for study with instance uid: ", studyinstanceuid)
	}

	// determine type and params
	var code int
	var size uint64
	contentType, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch contentType {

	case "multipart/related":
		{
			// store multipart message
			var mf multiparts.MultipartFiles
			code, size = mf.StoreMultipartMessage(&r.Header, &r.Body, h.dirOut, params)
		}

	case "application/dicom":
		{
			// store singlepart message
			var sf singleparts.SinglepartFiles
			code, size = sf.StoreSinglePartMessage(&r.Header, &r.Body, h.dirOut, params)
		}
	default:
		w.WriteHeader(http.StatusPreconditionRequired)
	}
	w.WriteHeader(code)
	duration := time.Since(s)
	partscommon.LogTotalTimeInfo("STORE "+studyinstanceuid, duration, size, duration, false)
}

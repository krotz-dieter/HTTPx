package multiparts

import (
	"bytes"
	"errors"
	"fmt"
	"httpxcommon/partscommon"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"
)

// Type representing http3 handling
type MultipartFiles struct{}

// boundary to be used
func (h *MultipartFiles) GetBoundary() string {
	return "DICOMDATABOUNDARY"
}

// get part file name
func (h *MultipartFiles) GetMultiPartFileName(p *multipart.Part) string {
	_, params, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
	filename := params["filename"]
	return filename
}

func (h *MultipartFiles) StoreMultipartMessage(header *http.Header, body *io.ReadCloser, dir string, params map[string]string) (int, uint64) {
	// keep start time
	s := time.Now()
	var size uint64

	// determine type and params
	size = 0
	boundary := params["boundary"]
	mr := multipart.NewReader(*body, boundary)
	klog.V(partscommon.KlogInfo).Info("Body boundary found: ", boundary, " in multipart/related messsage")
out:
	for {
		sPart := time.Now()
		part, err := mr.NextPart()
		switch {
		case err == io.EOF:
			{
				klog.V(partscommon.KlogInfo).Info("EOF encountered in multipart, quit")
				break out
			}
		case err != nil:
			{
				klog.Error(err)
				return http.StatusRequestedRangeNotSatisfiable, 0
			}
		}

		// process multi part
		originalfilename := h.GetMultiPartFileName(part)
		klog.V(partscommon.KlogInfo).Info("Part file name from header: ", originalfilename)
		studyinstanceuid, seriesinstanceuid, sopinstanceuid := partscommon.GetDICOMInfo(originalfilename)
		if len(studyinstanceuid) == 0 || len(seriesinstanceuid) == 0 || len(sopinstanceuid) == 0 {
			klog.Error("No valid original file name in header")
			return http.StatusNoContent, 0
		}

		// create directories and calculate filename
		filename := partscommon.EnsureFilePath(dir, studyinstanceuid, seriesinstanceuid, sopinstanceuid)

		// create blank file
		file, errFile := os.Create(filename)
		if errFile != nil {
			klog.Error(errFile)
			return http.StatusNotFound, 0
		}
		defer file.Close()
		// copy part to file
		length, errCopy := io.Copy(file, part)
		if errCopy != nil {
			klog.Error(errCopy)
			return http.StatusConflict, 0
		}
		size += uint64(length)
		klog.V(partscommon.KlogInfo).Infoln("Data copied into file:", filename, " with size:", length, " and time taken:", time.Since(sPart))
	}
	klog.V(partscommon.KlogDebug).Info("Time total taken: ", time.Since(s), " size:", size)
	return http.StatusOK, uint64(size)
}

func (h *MultipartFiles) SaveFilesFromResponse(rsp *http.Response, directory string, urlIn string) (error, uint64) {
	//measure duration
	s := time.Now()
	var errRet error = nil

	// get the study instance uid from url if available
	u, err := url.Parse(urlIn)
	if err != nil {
		panic(err)
	}
	var studyinstanceuid string
	urlPart := strings.Split(u.Path, "/")
	if len(urlPart) > 2 {
		if urlPart[1] == "studies" {
			studyinstanceuid = urlPart[2]
			klog.V(partscommon.KlogInfo).Info("Study instance uid:", studyinstanceuid)
		}
	}

	// checkout params
	_, params, _ := mime.ParseMediaType(rsp.Header.Get("Content-Type"))

	// store multipart message
	code, size := h.StoreMultipartMessage(&rsp.Header, &rsp.Body, directory, params)
	klog.V(partscommon.KlogInfo).Info("Code returned from storing multipart body:", code)

	klog.V(partscommon.KlogDebug).Info("Time total taken: ", time.Since(s))
	return errRet, size
}

func (h *MultipartFiles) ProcessFileSync(path string, writer *multipart.Writer) (uint64, error) {
	// path has to follow a certain structure
	studyinstanceuid, seriesinstanceuid, sopinstanceuid := partscommon.GetDICOMInfo(path)
	if len(studyinstanceuid) == 0 || len(seriesinstanceuid) == 0 || len(sopinstanceuid) == 0 {
		klog.Error("No valid original file name in header")
		return 0, errors.New("No correct file structure using DICOM tags !")
	}

	// read file
	file, err := os.Open(path)
	if err != nil {
		klog.V(partscommon.KlogDebug).Info("error reading file", path)
		return 0, err
	}
	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// create single part
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", "application/dicom")
	// header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name()))
	joinedpath := filepath.Join(studyinstanceuid, seriesinstanceuid, sopinstanceuid)
	klog.V(partscommon.KlogInfo).Info("Joined filename path:", joinedpath)
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", joinedpath))

	// create the part
	wPart, err := writer.CreatePart(header)
	if err != nil {
		return 0, err
	}
	// write the file content
	var n int
	if n, err = wPart.Write(fileContents); err != nil {
		return 0, err
	}

	// TEST DATA
	// if _, err = wPart.Write([]byte{'A', 'B', 'C', 'D'}); err != nil {
	// 	return 0, err
	// }
	return uint64(n), nil
}

func (h *MultipartFiles) UploadFilesFromDirectory(body io.Writer, directory string) (error, uint64) {
	// create a writer
	writer := multipart.NewWriter(body)
	// set the boundary
	err := writer.SetBoundary(h.GetBoundary())
	if err != nil {
		klog.Error(err)
		return err, 0
	}

	// build path
	path := directory
	klog.V(partscommon.KlogDebug).Info("Processing directory: ", path)
	s := time.Now()

	// walk through all files
	var len uint64 = 0
	errWalk := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			klog.V(partscommon.KlogDebug).Info("No files in directory: ", path, err)
			return err
		}

		// process every file
		if !info.IsDir() {
			sFile := time.Now()
			l, err := h.ProcessFileSync(path, writer)
			if err != nil {
				klog.V(partscommon.KlogDebug).Info("Error processing file: ", path, err)
			}
			klog.V(partscommon.KlogDebug).Info("Reading file: ", path, " in: ", time.Since(sFile), " size:", l)
			len += l
		}
		return nil
	})
	if errWalk != nil {
		klog.V(partscommon.KlogDebug).Info("No files in directory: ", path)
		return errWalk, len
	}
	klog.V(partscommon.KlogDebug).Info("Time total taken: ", time.Since(s), " size:", len)
	// this will lead to communication when closing part writer
	writer.Close()
	return nil, len
}

func (h *MultipartFiles) PostFilesFromDirectory(client *http.Client, url string, directory string) (error, uint64, time.Duration) {
	// create a new multipart writer and send all in one POST
	var size uint64
	s := time.Now()
	body := &bytes.Buffer{}

	// upload files
	errHandle, size := h.UploadFilesFromDirectory(body, directory)
	if errHandle != nil {
		klog.Error(errHandle)
		return errHandle, 0, 0
	}

	// Create a HTTP post request
	r, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		klog.Error("Error in creating POST request")
	}
	ct := fmt.Sprintf("multipart/related; boundary=%q; type=\"application/dicom\"", h.GetBoundary())
	r.Header.Add("Content-Type", ct)
	r.Header.Add("Content-Length", strconv.Itoa(body.Len()))
	partscommon.LogRequest(r)
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	partscommon.LogResponse(res)
	return nil, size, time.Since(s)
}

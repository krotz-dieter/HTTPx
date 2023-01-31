package partscommon

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"
)

const (
	KlogStatistics = 1
	KlogHttp       = 2
	KlogDebug      = 3
	KlogInfo       = 4
)

func GetDICOMInfo(originalfilename string) (string, string, string) {
	// path has to follow a certain structure
	sopinstanceuid, seriesinstanceuid, studyinstanceuid := "", "", ""
	parts := strings.Split(originalfilename, string(os.PathSeparator))
	if len(parts) < 3 {
		klog.V(KlogInfo).Info("No correct attachment with filename, quit")
		return studyinstanceuid, seriesinstanceuid, sopinstanceuid
	}
	sopinstanceuid = parts[len(parts)-1]
	sopinstanceuid = strings.ReplaceAll(sopinstanceuid, ".dcm", "")
	seriesinstanceuid = parts[len(parts)-2]
	studyinstanceuid = parts[len(parts)-3]
	klog.V(KlogInfo).Info("Extracted studyinstanceuid:", studyinstanceuid, " seriesinstanceuid:", seriesinstanceuid, " sopinstanceuid:", sopinstanceuid)
	return studyinstanceuid, seriesinstanceuid, sopinstanceuid
}

func EnsureFilePath(dir string, studyinstanceuid string, seriesinstanceuid string, sopinstanceuid string) string {
	// create directories for study and series level
	directoryname := filepath.Join(dir, studyinstanceuid)
	_ = os.Mkdir(directoryname, os.ModePerm)
	directoryname = filepath.Join(directoryname, seriesinstanceuid)
	_ = os.Mkdir(directoryname, os.ModePerm)
	// build file name
	filename := filepath.Join(dir, studyinstanceuid, seriesinstanceuid, sopinstanceuid+".dcm")
	klog.V(KlogInfo).Info("Target file name: ", filename)
	return filename
}

// log requests
func LogRequest(r *http.Request) {
	x, err := httputil.DumpRequest(r, false)
	if err != nil {
		return
	}
	klog.V(KlogHttp).Info(fmt.Sprintf("%q", x), " goroutine:", GetGID())
}

// log requests
func LogResponse(r *http.Response) {
	x, err := httputil.DumpResponse(r, false)
	if err != nil {
		return
	}
	klog.V(KlogHttp).Info(fmt.Sprintf("-> RESPONSE:%q", x), " goroutine:", GetGID())
}

// getGID gets the current goroutine ID (copied from https://blog.sgmansfield.com/2015/12/goroutine-ids/)
func GetGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func CheckDirectory(dir string) string {
	// get current path
	if dir == "" {
		path, err := os.Getwd()
		if err != nil {
			panic("Failed to get current path")
		}
		klog.V(KlogDebug).Info("Using current directory as path: ", path)
		return path
	} else {
		_, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				// File or directory does not exist
				panic("Directory " + dir + " does not exist. Please provide an existing directory !")
			}
		}
	}
	return dir
}

func ByteCountSI(b uint64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func LogTotalTimeInfo(s string, total time.Duration, size uint64, duration time.Duration, force bool) {
	speed := float64(size) / float64(duration.Seconds())
	klog.V(KlogDebug).Info(s, "Total time: ", total, " duration processing:", duration)
	if force {
		fmt.Println(s, " Time total taken: ", duration, " size:", ByteCountSI(size), " speed:", ByteCountSI(uint64(speed))+"/s")
	} else {
		klog.V(KlogStatistics).Info(s, " Time total taken: ", duration, " size:", ByteCountSI(size), " speed:", ByteCountSI(uint64(speed))+"/s")
	}
}

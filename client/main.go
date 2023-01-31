package main

import (
	"crypto/x509"
	"flag"
	"fmt"
	"httpxcommon/partscommon"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"k8s.io/klog"
)

func (h bufferedWriteCloser) Close() error {
	if err := h.Writer.Flush(); err != nil {
		return err
	}
	return h.Closer.Close()
}

func AddRootCA(certPath string, certPool *x509.CertPool) {
	caCertPath := path.Join(certPath, "cert", "cert-public.pem")
	klog.V(partscommon.KlogDebug).Info("Trying to access public certificate: ", caCertPath)
	caCertRaw, err := os.ReadFile(caCertPath)
	if err != nil {
		panic(err)
	}
	if ok := certPool.AppendCertsFromPEM(caCertRaw); !ok {
		panic("Could not add root ceritificate to pool.")
	}
}

func HandleCertificate(filename string, certDir string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	AddRootCA(certDir, pool)
	return pool, nil
}

func main() {
	// klog default
	klog.InitFlags(nil)
	defer klog.Flush()
	s := time.Now()

	// determine root path
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}
	certPath := path.Dir(filename)
	certPath = strings.TrimRight(certPath, filepath.Base(certPath))

	// additional parameters
	insecure := flag.Bool("insecure", false, "skip certificate verification")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	httpVersion := flag.String("http", "1.1", "http version to be used: 1.1 | 2.0 | 3.0")
	operation := flag.String("operation", "retrieve", "operation to be executed: retrieve | send")
	directory := flag.String("dir", "", "directory to be used")
	chunking := flag.String("chunking", "single", "chunking in parts to be used: single | multi")
	mode := flag.String("mode", "sync", "mode to be used: sync | async")
	dirCert := flag.String("cert", certPath, "directory with public and private certificate: cert-priv.perm, cert-public.pem")
	flag.Parse()
	// urls to be called
	urls := flag.Args()
	if len(urls) < 1 {
		fmt.Println("Usage: httpx-client flags url")
		fmt.Println("flags:")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("Retrieve with HTTPS/2: httpx-client -http 2.0 -dir . https://127.0.0.1:8082/studies/1.3.12.2.1107.5.99.3.30000009040610340869700000002")
		fmt.Println("Retrieve with HTTPS/3 and use detailed logs: httpx-client -v 8 -http 3.0  -dir . https://127.0.0.1:8083/studies/1.3.12.2.1107.5.99.3.30000009040610340869700000002")
		fmt.Println("Send with HTTPS/3 in async mode: httpx-client -http 3.0 -operation send -chunking single -mode async -dir . https://127.0.0.1:8083/studies/1.3.12.2.1107.5.99.3.30000009052811420737800000003")
		return
	}

	// check parameters
	if !((*mode == "sync") || (*mode == "async")) {
		panic("Please provide for mode: sync | async")
	}
	// check parameters
	if !((*chunking == "single") || (*chunking == "multi")) {
		panic("Please provide for mode: sync | async")
	}

	// check input directory
	*directory = partscommon.CheckDirectory(*directory)

	// handle the certificates
	var pool *x509.CertPool = nil
	if !*insecure {
		var errPool error = nil
		pool, errPool = HandleCertificate(filename, *dirCert)
		if errPool != nil {
			panic("Failed to handle sets of certificates")
		}
	}

	// handle operations
	var size uint64
	var duration time.Duration
	if *operation == "retrieve" {
		// handle request based on http version
		var errHttpGet error = nil
		if *httpVersion == "1.1" {
			var http1 http1Handler
			errHttpGet, size, duration = http1.HandleHttpGet(urls[0], pool, insecure, *directory)
		}
		if *httpVersion == "2.0" {
			var http2 http2Handler
			errHttpGet, size, duration = http2.HandleHttpGet(urls[0], pool, insecure, *directory)
		}
		if *httpVersion == "3.0" {
			var http3 http3Handler
			errHttpGet, size, duration = http3.HandleHttpGet(urls[0], enableQlog, pool, insecure, *directory)
		}
		if errHttpGet != nil {
			klog.Errorf("HTTP call returned error: %v", errHttpGet)
		}
		partscommon.LogTotalTimeInfo(" RETRIEVE", time.Since(s), size, duration, true)
	} else if *operation == "send" {
		// handle request based on http version
		var errHttpPost error = nil
		if *httpVersion == "1.1" {
			var http1 http1Handler
			http1.chunking = *chunking
			http1.mode = *mode
			errHttpPost, size, duration = http1.HandleHttpPost(urls[0], pool, insecure, *directory)
		}
		if *httpVersion == "2.0" {
			var http2 http2Handler
			http2.chunking = *chunking
			http2.mode = *mode
			errHttpPost, size, duration = http2.HandleHttpPost(urls[0], pool, insecure, *directory)
		}
		if *httpVersion == "3.0" {
			var http3 http3Handler
			http3.chunking = *chunking
			http3.mode = *mode
			errHttpPost, size, duration = http3.HandleHttpPost(urls[0], enableQlog, pool, insecure, *directory)
		}
		if errHttpPost != nil {
			klog.Errorf("POST call returned error: %v", errHttpPost)
		}
		partscommon.LogTotalTimeInfo(" SEND", time.Since(s), size, duration, true)
	}
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"httpxcommon/partscommon"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"crypto/tls"
	_ "net/http/pprof"

	"golang.org/x/net/http2"

	"k8s.io/klog"

	"github.com/gorilla/mux"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
)

type binds []string

func (b binds) String() string {
	return strings.Join(b, ",")
}

func (b *binds) Set(v string) error {
	*b = strings.Split(v, ",")
	return nil
}

// Size is needed by the /demo/upload handler to determine the size of the uploaded file
type Size interface {
	Size() int64
}

// main handler function
func setupHandler(www string, dirIn string) http.Handler {
	// route := http.NewServeMux()
	route := mux.NewRouter()

	// use a default file server per default if there is some domain defined
	if len(www) > 0 {
		klog.V(partscommon.KlogDebug).Info("Hosting file server under ", www)
		route.Handle("/", http.FileServer(http.Dir(www)))
	}

	// echo call for tests
	route.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading body while handling /echo: %s\n", err.Error())
		}

		// read body part
		body, err1 := io.ReadAll(r.Body)
		if err1 != nil {
			klog.V(partscommon.KlogDebug).Info("Error reading body while handling /echo: %s\n", err1.Error())
		}

		// dump message
		partscommon.LogRequest(r)
		w.Write(body)
	})

	// DICOM handlers
	var ss StoreOperation
	ss.dirOut = dirIn
	route.HandleFunc("/studies/{study}", ss.StoreStudy).Methods("POST")
	var rs RetrieveOperation
	rs.dirIn = dirIn
	route.HandleFunc("/studies/{study}", rs.RetrieveStudy).Methods("GET")
	route.HandleFunc("/studies/{study}/series/{series}", rs.RetrieveSeries).Methods("GET")
	route.HandleFunc("/studies/{study}/series/{series}/instances/{instance}", rs.RetrieveInstance).Methods("GET")
	return route
}

type bufferedWriteCloser struct {
	*bufio.Writer
	io.Closer
}

// NewBufferedWriteCloser creates an io.WriteCloser from a bufio.Writer and an io.Closer
func NewBufferedWriteCloser(writer *bufio.Writer, closer io.Closer) io.WriteCloser {
	return &bufferedWriteCloser{
		Writer: writer,
		Closer: closer,
	}
}

func (h bufferedWriteCloser) Close() error {
	if err := h.Writer.Flush(); err != nil {
		return err
	}
	return h.Closer.Close()
}

// GetCertificatePaths returns the paths to certificate and key
func GetCertificatePaths(certPath string) (string, string) {
	pubCert := path.Join(certPath, "cert", "cert-public.pem")
	privCert := path.Join(certPath, "cert", "cert-priv.pem")
	klog.V(partscommon.KlogDebug).Info("Public Certificate:", pubCert, " Private Certificate:", privCert)
	return pubCert, privCert
}

func main() {
	// logging setup
	klog.InitFlags(nil)
	defer klog.Flush()

	// determine root path
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current frame")
	}
	certPath := path.Dir(filename)
	certPath = strings.TrimRight(certPath, filepath.Base(certPath))

	// check parameters
	// flag.Var(&bs, "bind", "bind to")
	www := flag.String("www", "", "www data")
	tcp := flag.Bool("tcp", false, "also listen on TCP")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	dirIn := flag.String("dir", "", "directory to be used as main directory")
	dirCert := flag.String("cert", certPath, "directory with public and private certificate: cert-priv.perm, cert-public.pem")
	flag.Parse()
	klog.V(partscommon.KlogDebug).Info("Parameters www:", *www, " tcp:", *tcp, " QLog:", *enableQlog)

	// calculate cert path
	partscommon.CheckDirectory(*dirCert)
	certFile, keyFile := GetCertificatePaths(*dirCert)

	// setup handler
	handler := setupHandler(*www, partscommon.CheckDirectory(*dirIn))
	var quicConf *quic.Config = nil
	if *enableQlog {
		quicConf := &quic.Config{}
		quicConf.Tracer = qlog.NewTracer(func(_ logging.Perspective, connID []byte) io.WriteCloser {
			filename := fmt.Sprintf("server_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				klog.Fatal(err)
			}
			klog.V(partscommon.KlogDebug).Info("Creating qlog file %s.\n", filename)
			return NewBufferedWriteCloser(bufio.NewWriter(f), f)
		})
	}

	// use waitgroup to wait for all threads to be finished
	var wg sync.WaitGroup

	// start http listener on HTTP/1.1
	// defer profile.Start().Stop()
	wg.Add(1)
	go func() {
		fmt.Println("Running http server (HTTP /1.1) on port 8080 using TCP in goroutine:", partscommon.GetGID())
		defer wg.Done()
		klog.V(partscommon.KlogDebug).Info(http.ListenAndServe(":8080", handler))
	}()

	// start http listener on HTTPS/1.1
	wg.Add(1)
	go func() {
		fmt.Println("Running http server (HTTPS/1.1) on port 8081 using TCP in goroutine:", partscommon.GetGID())
		defer wg.Done()
		httpServer := &http.Server{
			Handler:      handler,
			Addr:         ":8081",
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		klog.V(partscommon.KlogDebug).Info(httpServer.ListenAndServeTLS(certFile, keyFile))
	}()

	// start http listener on HTTPS/2
	wg.Add(1)
	go func() {
		fmt.Println("Running http server (HTTPS/2.0) on port 8082 using TCP in goroutine:", partscommon.GetGID())
		defer wg.Done()
		var httpServer = http.Server{
			Addr: ":8082", Handler: handler,
		}
		var http2Server = http2.Server{}
		_ = http2.ConfigureServer(&httpServer, &http2Server)
		klog.V(partscommon.KlogDebug).Info(httpServer.ListenAndServeTLS(certFile, keyFile))
	}()

	// start http listener on HTTPS/3
	bs := binds{}
	bs = binds{":8083"}
	wg.Add(1)
	for _, b := range bs {
		bCap := b
		go func() {
			var err error
			defer wg.Done()
			if *tcp {
				fmt.Println("Running http server (HTTPS/3.0) on port 8083 using TCP in goroutine:", partscommon.GetGID())
				err = http3.ListenAndServe(bCap, certFile, keyFile, handler)
			} else {
				fmt.Println("Running http server (HTTPS/3.0) on port 8083 using UDP in goroutine:", partscommon.GetGID())
				server := http3.Server{
					Handler:    handler,
					Addr:       bCap,
					QuicConfig: quicConf,
				}
				err = server.ListenAndServeTLS(certFile, keyFile)
			}
			if err != nil {
				fmt.Println(err)
			}
		}()
	}
	wg.Wait()
}

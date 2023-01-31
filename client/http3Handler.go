package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
	"k8s.io/klog"

	"httpxcommon/httpxhelper"
	"httpxcommon/partscommon"
	"httpxcommon/terminal"
)

// Type representing http3 handling
type http3Handler struct {
	chunking string
	mode     string
}

type bufferedWriteCloser struct {
	*bufio.Writer
	io.Closer
}

// NewBufferedWriteCloser creates an io.WriteCloser from a bufio.Writer and an io.Closer
func (h *http3Handler) NewBufferedWriteCloser(writer *bufio.Writer, closer io.Closer) io.WriteCloser {
	return &bufferedWriteCloser{
		Writer: writer,
		Closer: closer,
	}
}

func (h *http3Handler) InitializeClient(enableQlog *bool, pool *x509.CertPool, insecure *bool) *http.Client {
	// log file for quic protocol
	var quicConf *quic.Config = nil
	if *enableQlog {
		quicConf := &quic.Config{}
		quicConf.Tracer = qlog.NewTracer(func(_ logging.Perspective, connID []byte) io.WriteCloser {
			filename := fmt.Sprintf("client_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			klog.V(partscommon.KlogDebug).Infof("Creating qlog file %s.\n", filename)
			return h.NewBufferedWriteCloser(bufio.NewWriter(f), f)
		})
	}
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: *insecure,
		},
		QuicConfig: quicConf,
	}
	defer roundTripper.Close()

	hclient := &http.Client{
		Transport: roundTripper,
	}
	return hclient
}

func (h *http3Handler) HandleHttpGet(url string, enableQlog *bool, pool *x509.CertPool, insecure *bool, directory string) (error, uint64, time.Duration) {
	// start spinner
	info := "Retrieve using HTTP GET with HTTPS/3.0 on:" + url
	terminal.Println()
	_, err := terminal.StartSpinner(info)
	if err != nil {
		klog.Error(err)
		terminal.Println()
		return err, 0, 0
	}

	// initialize client
	client := h.InitializeClient(enableQlog, pool, insecure)

	// retrieve the files
	errHandle, size, duration := httpxhelper.RetrieveFiles(client, url, directory)
	if errHandle != nil {
		klog.Error(errHandle)
		return errHandle, size, duration
	}
	return nil, size, duration
}

func (h *http3Handler) HandleHttpPost(url string, enableQlog *bool, pool *x509.CertPool, insecure *bool, directory string) (error, uint64, time.Duration) {
	// start spinner
	info := "Send using HTTP POST with HTTPS/3.0 on:" + url
	_, err := terminal.StartSpinner(info)
	if err != nil {
		klog.Error(err)
		terminal.Println()
		return err, 0, 0
	}

	// initialize client
	client := h.InitializeClient(enableQlog, pool, insecure)

	// send files
	errHandle, size, duration := httpxhelper.SendFiles(client, url, directory, h.chunking, h.mode)
	if errHandle != nil {
		klog.Error(errHandle)
		return errHandle, size, duration
	}
	return nil, size, duration
}

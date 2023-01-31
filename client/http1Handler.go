package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"httpxcommon/httpxhelper"
	"httpxcommon/terminal"

	"k8s.io/klog"
)

// Type representing http1 handling
type http1Handler struct {
	chunking string
	mode     string
}

func (h *http1Handler) InitializeClient(pool *x509.CertPool, insecure *bool) *http.Client {
	// use certificate
	client := &http.Client{}
	tlsConfig := &tls.Config{
		RootCAs: pool,
	}
	client.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	return client
}

func (h *http1Handler) HandleHttpGet(url string, pool *x509.CertPool, insecure *bool, directory string) (error, uint64, time.Duration) {
	// start spinner
	info := "Retrieve using HTTP GET with HTTPS/1.1 on:" + url
	terminal.Println()
	_, err := terminal.StartSpinner(info)
	if err != nil {
		klog.Error(err)
		terminal.Println()
		return err, 0, 0
	}

	// initialize client
	client := h.InitializeClient(pool, insecure)

	// retrieve the files
	errHandle, size, duration := httpxhelper.RetrieveFiles(client, url, directory)
	if errHandle != nil {
		klog.Error(errHandle)
		return errHandle, size, duration
	}
	return nil, size, duration
}

func (h *http1Handler) HandleHttpPost(url string, pool *x509.CertPool, insecure *bool, directory string) (error, uint64, time.Duration) {
	// start spinner
	info := "Send using HTTP POST with HTTPS/1.1 on:" + url
	_, err := terminal.StartSpinner(info)
	if err != nil {
		klog.Error(err)
		terminal.Println()
		return err, 0, 0
	}

	// initialize client
	client := h.InitializeClient(pool, insecure)

	// send files
	errHandle, size, duration := httpxhelper.SendFiles(client, url, directory, h.chunking, h.mode)
	if errHandle != nil {
		klog.Error(errHandle)
		return errHandle, size, duration
	}
	return nil, size, duration
}

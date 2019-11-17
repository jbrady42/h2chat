package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

const url = "https://localhost:8000"

var httpVersion = flag.Int("version", 2, "HTTP version")

var httpTrans *http2.Transport

func init() {
	httpTrans = &http2.Transport{
		TLSClientConfig: tlsConfig(),
	}
}

func tlsConfig() *tls.Config {
	// Create a pool with the server certificate since it is not signed
	// by a known CA
	caCert, err := ioutil.ReadFile("certs/server.crt")
	if err != nil {
		log.Fatalf("Reading server certificate: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration with the certificate of the server
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return tlsConfig
}

func GetTopics() {

}

func SendMessage() {

}

func getClient() *http.Client {
	client := &http.Client{}
	client.Transport = httpTrans
	return client
}

func main() {
	flag.Parse()
	client := getClient()

	// Perform the request
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalf("Failed get: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed reading response body: %s", err)
	}
	fmt.Printf(
		"Got response %d: %s %s\n",
		resp.StatusCode, resp.Proto, string(body))
}

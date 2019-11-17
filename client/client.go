package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/jbrady42/h2chat"
	"github.com/r3labs/sse"
	"golang.org/x/net/http2"
)

const baseUrl = "https://localhost:8000"

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

func SendMessage(msg string) {
	postUrl := baseUrl + "/messages"

	reqBody, err := json.Marshal(h2chat.Message{
		Name:    "Test name",
		Message: msg,
	})
	if err != nil {
		log.Fatalf("Error encoding message %s", err)
	}

	client := getClient()
	resp, err := client.Post(postUrl, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error posting message %s", err)
	}

	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed reading response body: %s", err)
	}
	// log.Printf("Response is %s", string(body))
}

func getClient() *http.Client {
	client := &http.Client{}
	client.Transport = httpTrans
	return client
}

func main() {
	flag.Parse()

	go func() {
		time.Sleep(time.Second)

		var c = 0
		for {
			time.Sleep(time.Second)
			SendMessage(fmt.Sprintf("Testing %d", c))
			c += 1
		}
	}()

	eventClient := sse.NewClient(baseUrl + "/events")
	eventClient.Connection.Transport = httpTrans

	eventClient.Subscribe("messages", func(msg *sse.Event) {
		// Got some data!
		fmt.Println(string(msg.Data))
	})
}

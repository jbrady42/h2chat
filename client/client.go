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
	"github.com/marcusolsson/tui-go"
	"github.com/r3labs/sse"
	"golang.org/x/net/http2"
)

const baseUrl = "https://localhost:8000"
const postUrl = baseUrl + "/messages"
const topicUrl = baseUrl + "/topics"

var httpVersion = flag.Int("version", 2, "HTTP version")

var httpTrans *http2.Transport

type client struct {
	eventClient *sse.Client
	events      chan *sse.Event
	ui          tui.UI
	history     *tui.Box
}

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

func SendMessage(msg string) {
	reqBody, err := json.Marshal(h2chat.Message{
		Name:    "Test name",
		Message: msg,
		Time:    time.Now(),
		Topic:   "default",
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

func getTopics() []string {
	client := getClient()
	resp, err := client.Get(topicUrl)
	if err != nil {
		log.Fatalf("Error fetching channels %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed reading response body: %s", err)
	}
	log.Printf("Response is %s", string(body))

	var topics []string
	err = json.Unmarshal(body, &topics)
	if err != nil {
		log.Fatalf("Failed parsing topic response body: %s", err)
	}
	return topics
}

func getClient() *http.Client {
	client := &http.Client{}
	client.Transport = httpTrans
	return client
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func setSelected(inc int, list *tui.List) {
	current := list.Selected()
	if inc < 0 {
		current = Max(current+inc, 0)
	} else {
		current = Min(current+inc, list.Length()-1)
	}
	list.Select(current)
}

func (t *client) subscribeTopic(topic string) {
	events := make(chan *sse.Event)
	err := t.eventClient.SubscribeChan(topic, events)
	if err != nil {
		log.Fatalf("Cant create listener %s", err)
	}
	t.events = events

	go t.handleMessages(events)
}

func (t *client) handleMessages(events chan *sse.Event) {
	for msg := range events {
		var post h2chat.Message
		err := json.Unmarshal(msg.Data, &post)
		if err != nil {
			log.Fatalf("Failed decoding incoming message %v", err)
		}

		t.ui.Update(func() {
			t.history.Append(tui.NewHBox(
				tui.NewLabel(post.Time.Format(time.Kitchen)),
				tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", post.Name))),
				tui.NewLabel(post.Message),
				tui.NewSpacer(),
			))
		})
	}
}

func main() {
	flag.Parse()

	eventClient := sse.NewClient(baseUrl + "/events")
	eventClient.Connection.Transport = httpTrans

	topics := getTopics()

	// GUI

	tList := tui.NewList()

	for _, a := range topics {
		tList.AddItems(a)
	}

	tList.Select(0)

	sidebar := tui.NewVBox(
		tList,
		tui.NewSpacer(),
	)
	sidebar.SetBorder(true)

	history := tui.NewVBox()

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	root := tui.NewHBox(sidebar, chat)

	input.OnSubmit(func(e *tui.Entry) {
		if e.Text() == "" {
			return // Skip empty messages
		}
		SendMessage(e.Text())
		input.SetText("")
	})

	ui, err := tui.New(root)
	if err != nil {
		log.Fatalf("Cant create UI %s", err)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })
	ui.SetKeybinding("Up", func() { historyScroll.Scroll(0, -1) })
	ui.SetKeybinding("Down", func() { historyScroll.Scroll(0, 1) })

	ui.SetKeybinding("PgUp", func() { setSelected(-1, tList) })
	ui.SetKeybinding("PgDn", func() { setSelected(1, tList) })

	client := client{eventClient, nil, ui, history}

	// Not blocking
	client.subscribeTopic("default")

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}

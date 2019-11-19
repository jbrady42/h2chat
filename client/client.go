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
	"os"
	"time"

	"github.com/cznic/mathutil"
	"github.com/jbrady42/h2chat"
	"github.com/marcusolsson/tui-go"
	"github.com/r3labs/sse"
	"golang.org/x/net/http2"
)

var httpVersion = flag.Int("version", 2, "HTTP version")

var logger *log.Logger

func init() {
	f, err := os.Create("debug.log")
	if err != nil {
		// ...
	}
	// defer f.Close()

	logger = log.New(f, "", log.LstdFlags)
}

type client struct {
	baseUrl      string
	httpTrans    *http2.Transport
	eventClient  *sse.Client
	Topics       []string
	currentTopic string
	eventChan    chan *sse.Event
	ui           *chatUI
	messages     []h2chat.Message
}

type chatUI struct {
	ui      tui.UI
	history *tui.Box
}

func NewClient(baseUrl string) client {
	httpTrans := &http2.Transport{
		TLSClientConfig: tlsConfig(),
	}

	eventClient := sse.NewClient(baseUrl + "/events")
	eventClient.Connection.Transport = httpTrans

	var msgs []h2chat.Message
	var topics []string
	return client{
		baseUrl, httpTrans, eventClient,
		topics, "", nil, nil, msgs,
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

func (t *client) resetMessages() {
	var msg []h2chat.Message
	t.messages = msg
}

func (t *client) SendMessage(msg string) {
	postUrl := t.baseUrl + "/messages"
	reqBody, err := json.Marshal(h2chat.Message{
		Name:    "Test name",
		Message: msg,
		Time:    time.Now(),
		Topic:   t.currentTopic,
	})
	if err != nil {
		log.Fatalf("Error encoding message %s", err)
	}

	client := t.getClient()
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

func (t *client) loadTopics() {
	t.Topics = t.getTopics()
}

func (t *client) getTopics() []string {
	topicUrl := t.baseUrl + "/topics"
	client := t.getClient()
	resp, err := client.Get(topicUrl)
	if err != nil {
		log.Fatalf("Error fetching channels %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed reading response body: %s", err)
	}
	logger.Printf("Response is %s", string(body))

	var topics []string
	err = json.Unmarshal(body, &topics)
	if err != nil {
		log.Fatalf("Failed parsing topic response body: %s", err)
	}
	return topics
}

func (t *client) getClient() *http.Client {
	client := &http.Client{}
	client.Transport = t.httpTrans
	return client
}

func setSelected(inc int, list *tui.List) {
	current := list.Selected()
	if inc < 0 {
		current = mathutil.Max(current+inc, 0)
	} else {
		current = mathutil.Min(current+inc, list.Length()-1)
	}
	list.Select(current)
}

func (t *client) subscribeTopic(topic string) {

	logger.Println("Setting topic to " + topic)

	// Unsubscribe first to prevent further UI updates
	if t.eventChan != nil {
		t.eventClient.Unsubscribe(t.eventChan)
		close(t.eventChan)
		t.eventChan = nil
	}

	// Clear history
	t.resetMessages()

	t.currentTopic = topic
	events := make(chan *sse.Event)
	err := t.eventClient.SubscribeChan(topic, events)
	if err != nil {
		log.Fatalf("Cant create listener %s", err)
	}
	t.eventChan = events
	logger.Println("Subscribed to new topic " + topic)
	go t.handleMessages(events)
}

func (t *client) handleMessages(events chan *sse.Event) {
	for msg := range events {
		logger.Print(".")
		var post h2chat.Message
		err := json.Unmarshal(msg.Data, &post)
		if err != nil {
			log.Fatalf("Failed decoding incoming message %v", err)
		}
		if post.Topic == t.currentTopic {
			t.messages = append(t.messages, post)
			t.updateUI()
		}
	}
}

func (t *client) updateUI() {
	t.ui.ui.Update(func() {
		clearBox(t.ui.history)

		//Repaint
		for _, msg := range t.messages {
			t.ui.history.Append(tui.NewHBox(
				tui.NewLabel(msg.Time.Format(time.Kitchen)),
				tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", msg.Name))),
				tui.NewLabel(msg.Message),
				tui.NewSpacer(),
			))
		}
	})
}

func clearBox(box *tui.Box) {
	len := box.Length()
	for a := 0; a < len; a++ {
		box.Remove(a)
	}
}

func main() {
	flag.Parse()

	baseUrl := "https://localhost:8000"

	client := NewClient(baseUrl)
	client.loadTopics()

	// GUI

	tList := tui.NewList()

	for _, a := range client.Topics {
		tList.AddItems(a)
	}

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
		client.SendMessage(e.Text())
		input.SetText("")
	})

	tList.OnSelectionChanged(func(l *tui.List) {
		indx := l.Selected()
		topic := client.Topics[indx]
		go client.subscribeTopic(topic)
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

	cUI := &chatUI{ui, history}

	client.ui = cUI

	// Not blocking
	// client.subscribeTopic("default")
	go func() {
		time.Sleep(1 * time.Second)
		ui.Update(func() {
			tList.Select(1)
		})
	}()

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}

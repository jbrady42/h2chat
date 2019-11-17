package h2chat

import "time"

type Message struct {
	Name    string
	Message string
	Time    time.Time
	Topic   string
}

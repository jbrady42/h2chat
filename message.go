package h2chat

import "time"

type Message struct {
	Name      string
	Message   string
	Timestamp time.Time
}

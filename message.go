package h2_chat

import "time"

type Message struct {
	name      String
	message   String
	Timestamp time.Time
}

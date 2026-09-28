package ws

import (
	"sync"
	"testing"
)

func TestTicketIsSingleUseAndBoundToIdentity(t *testing.T) {
	hub := NewHub()
	ticket := hub.IssueTicket(42, 7)
	if ticket == "" {
		t.Fatal("IssueTicket() returned empty ticket")
	}
	first, ok := hub.ConsumeTicket(ticket)
	if !ok || first.UserID != 42 || first.OrgID != 7 {
		t.Fatalf("ConsumeTicket() = %#v, %v", first, ok)
	}
	if _, ok := hub.ConsumeTicket(ticket); ok {
		t.Fatal("ConsumeTicket() accepted reused ticket")
	}
}

func TestBroadcastCopiesRoomBeforeSending(t *testing.T) {
	hub := NewHub()
	client := &Client{Hub: hub, UserID: 1, OrgID: 9, Send: make(chan []byte, 8)}
	hub.Register(client)

	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			hub.Broadcast(9, Message{Type: "task:updated", Data: map[string]int{"id": 1}})
		}()
	}
	group.Add(1)
	go func() {
		defer group.Done()
		hub.Unregister(client)
	}()
	group.Wait()

	if count := hub.Count(9); count != 0 {
		t.Fatalf("Count() = %d, want 0", count)
	}
}

func TestMessagesRemainSeparateFrames(t *testing.T) {
	hub := NewHub()
	client := &Client{Hub: hub, UserID: 1, OrgID: 2, Send: make(chan []byte, 2)}
	hub.Register(client)
	hub.Broadcast(2, Message{Type: "first", Data: 1})
	hub.Broadcast(2, Message{Type: "second", Data: 2})

	first := <-client.Send
	second := <-client.Send
	if string(first) == string(second) {
		t.Fatalf("broadcast messages were merged: %s", first)
	}
}

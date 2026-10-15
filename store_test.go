package whatsapp

import (
	"fmt"
	"sync"
	"testing"
)

func TestStoreAddChat(t *testing.T) {
	store := newStore()
	chat := Chat{Jid: "123@c.us", Name: "test"}
	if err := store.AddChat(chat); err != nil {
		t.Fatalf("AddChat returned error: %v", err)
	}
	got, ok := store.GetChat("123@s.whatsapp.net")
	if !ok || got.Name != chat.Name {
		t.Fatalf("GetChat() = (%+v, %t), want chat %+v", got, ok, chat)
	}
}

func TestStoreConcurrentAddChat(t *testing.T) {
	store := newStore()
	const workers = 32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			jid := fmt.Sprintf("%d@s.whatsapp.net", i)
			if err := store.AddChat(Chat{Jid: jid}); err != nil {
				t.Errorf("AddChat(%q) returned error: %v", jid, err)
			}
		}(i)
	}
	wg.Wait()
	if got := len(store.GetChats()); got != workers {
		t.Fatalf("GetChats() returned %d chats, want %d", got, workers)
	}
}

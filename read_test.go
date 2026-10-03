package whatsapp

import (
	"testing"

	"github.com/gorilla/websocket"
)

func TestProcessReadDataRejectsEmptyFrame(t *testing.T) {
	wac := &Conn{}
	if err := wac.processReadData(websocket.TextMessage, nil); err != ErrInvalidWsData {
		t.Fatalf("processReadData(empty) error = %v, want %v", err, ErrInvalidWsData)
	}
}

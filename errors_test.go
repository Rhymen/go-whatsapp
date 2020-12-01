package whatsapp

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrConnectionFailedError(t *testing.T) {
	err := ErrConnectionFailed{
		Err: errors.New("test"),
	}

	assert.Equal(t, "connection to WhatsApp servers failed: test", err.Error())
}

func TestErrConnectionClosedError(t *testing.T) {
	err := ErrConnectionClosed{
		Text: "test",
		Code: 1,
	}

	assert.Equal(t, "server closed connection,code: 1,text: test", err.Error())
}

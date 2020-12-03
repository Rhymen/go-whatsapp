package whatsapp

import (
	"strconv"
	"testing"

	"github.com/Rhymen/go-whatsapp/binary"
	"github.com/stretchr/testify/assert"
)

var writeBinaryMock func(n binary.Node, m metric, f flag, messageTag string) (<-chan string, error)
var createUploadProfilePicTagMock func(msgCount int) string

type ProfileMockConn struct{}

func (c *ProfileMockConn) writeBinary(n binary.Node, m metric, f flag, messageTag string) (<-chan string, error) {
	return writeBinaryMock(n, m, f, messageTag)
}

func (c *ProfileMockConn) createUploadProfilePicNode(msgCount int, tag, wid string, image, preview []byte) binary.Node {
	return new(Conn).createUploadProfilePicNode(msgCount, tag, wid, image, preview)
}

func (c *ProfileMockConn) createUploadProfilePicTag(msgCount int) string {
	return createUploadProfilePicTagMock(msgCount)
}

func TestUploadProfilePic(t *testing.T) {
	writeBinaryCalled := false
	profilePicTag := "create-profile-pic-tag"
	messageCount := 1
	wid := "wid"
	image := []byte("image")
	preview := []byte("preview")

	mock := &ProfileMockConn{}
	createUploadProfilePicTagMock = func(msgCount int) string {
		return profilePicTag
	}
	writeBinaryMock = func(n binary.Node, m metric, f flag, messageTag string) (<-chan string, error) {
		writeBinaryCalled = true
		contentNode := n.Content.([]interface{})[0].(binary.Node)
		contentImageNode := contentNode.Content.([]binary.Node)[0]
		contentPreviewNode := contentNode.Content.([]binary.Node)[1]

		assert.Equal(t, profile, m)
		assert.Equal(t, flag(136), f)
		assert.Equal(t, profilePicTag, messageTag)

		assert.Equal(t, n.Description, "action")
		assert.Equal(t, n.Attributes["type"], "set")
		assert.Equal(t, n.Attributes["epoch"], strconv.Itoa(messageCount))

		assert.Equal(t, contentNode.Description, "picture")
		assert.Equal(t, contentNode.Attributes["id"], profilePicTag)
		assert.Equal(t, contentNode.Attributes["jid"], wid)
		assert.Equal(t, contentNode.Attributes["type"], "set")

		assert.Equal(t, contentImageNode.Description, "image")
		assert.Equal(t, contentImageNode.Content, image)
		assert.Nil(t, contentImageNode.Attributes)

		assert.Equal(t, contentPreviewNode.Description, "preview")
		assert.Equal(t, contentPreviewNode.Content, preview)
		assert.Nil(t, contentPreviewNode.Attributes)

		return make(<-chan string), nil
	}

	_, err := UploadProfilePic(mock, messageCount, wid, image, preview)

	assert.Nil(t, err)
	assert.True(t, writeBinaryCalled)
}

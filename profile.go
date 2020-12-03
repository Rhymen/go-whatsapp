package whatsapp

import (
	"fmt"
	"strconv"
	"time"

	"github.com/Rhymen/go-whatsapp/binary"
)

type ProfilePicUploader interface {
	writeBinary(binary.Node, metric, flag, string) (<-chan string, error)
	createUploadProfilePicNode(msgCount int, tag, wid string, image, preview []byte) binary.Node
	createUploadProfilePicTag(msgCount int) string
}

func (wac *Conn) createUploadProfilePicTag(msgCount int) string {
	return fmt.Sprintf("%d.--%d", time.Now().Unix(), msgCount*19)
}

func (wac *Conn) createUploadProfilePicNode(msgCount int, tag, wid string, image, preview []byte) binary.Node {
	return binary.Node{
		Description: "action",
		Attributes: map[string]string{
			"type":  "set",
			"epoch": strconv.Itoa(msgCount),
		},
		Content: []interface{}{
			binary.Node{
				Description: "picture",
				Attributes: map[string]string{
					"id":   tag,
					"jid":  wid,
					"type": "set",
				},
				Content: []binary.Node{
					{
						Description: "image",
						Attributes:  nil,
						Content:     image,
					},
					{
						Description: "preview",
						Attributes:  nil,
						Content:     preview,
					},
				},
			},
		},
	}
}

// Pictures must be JPG 640x640 and 96x96, respectively
func (wac *Conn) UploadProfilePic(image, preview []byte) (<-chan string, error) {
	return UploadProfilePic(wac, wac.msgCount, wac.Info.Wid, image, preview)
}

func UploadProfilePic(wac ProfilePicUploader, msgCount int, wid string, image, preview []byte) (<-chan string, error) {
	t := wac.createUploadProfilePicTag(msgCount)
	n := wac.createUploadProfilePicNode(msgCount, t, wid, image, preview)

	return wac.writeBinary(n, profile, 136, t)
}

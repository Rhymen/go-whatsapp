package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"github.com/Rhymen/go-whatsapp/binary/proto"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
)

func main() {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating connection: %v\n", err)
		return
	}

	err = login(wac)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error logging in: %v\n", err)
		return
	}

	<-time.After(6 * time.Second)

	previousMessage := "How about u"
	quotedMessage := proto.Message{
		Conversation: &previousMessage,
	}

	ContextInfo := whatsapp.ContextInfo{
		QuotedMessage:   &quotedMessage,
		QuotedMessageID: "F229B6BCCE2A0D6BBB0AA56110887D20",
		Participant:     "919315346424@s.whatsapp.net", //Who sent the original message
	}

	title := "List 1"
	description := "description 1"
	rowId := "rowId1"

	var Section = []*proto.Section{ ///
		{
			Title: &title,
			Rows: []*proto.Row{{
				Title:       &title,
				Description: &description,
				RowId:       &rowId,
			}},
		},
	}
	msg := whatsapp.ListMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: "966593276726-1614673783@g.us",
		},
		ContextInfo: ContextInfo,
		Title:       "This is List title",
		Description: "This is List description",
		ButtonText:  "This is List buttonText",
		FooterText:  "This is List footerText",
		ListType:    proto.ListMessage_SINGLE_SELECT,
		Sections:    Section,
	}

	msgId, err := wac.Send(msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error sending message: %v", err)
		os.Exit(1)
	} else {
		fmt.Println("Message Sent -> ID : " + msgId)
		<-time.After(3 * time.Second)
	}
}

func login(wac *whatsapp.Conn) error {
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			return fmt.Errorf("restoring failed: %v\n", err)
		}
	} else {
		//no saved session -> regular login
		qr := make(chan string)
		go func() {
			terminal := qrcodeTerminal.New()
			terminal.Get(<-qr).Print()
		}()
		session, err = wac.Login(qr)
		if err != nil {
			return fmt.Errorf("error during login: %v\n", err)
		}
	}

	//save session
	err = writeSession(session)
	if err != nil {
		return fmt.Errorf("error saving session: %v\n", err)
	}
	return nil
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	file, err := os.Open(os.TempDir() + "/whatsappSession2.gob")
	if err != nil {
		return session, err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&session)
	if err != nil {
		return session, err
	}
	return session, nil
}

func writeSession(session whatsapp.Session) error {
	file, err := os.Create(os.TempDir() + "/whatsappSession.gob")
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(session)
	if err != nil {
		return err
	}
	return nil
}

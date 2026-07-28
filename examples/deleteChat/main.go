package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
)

type waHandler struct {
	c            *whatsapp.Conn
	ReceivedChat chan struct{}
}

func (h *waHandler) ShouldCallSynchronously() bool {
	return true
}

func (w *waHandler) HandleChatList(chats []whatsapp.Chat) {
	fmt.Println("Chat list received")
	chatMap := make(map[string]whatsapp.Chat)
	for _, chat := range w.c.Store.Chats {
		chatMap[chat.Jid] = chat
	}
	for _, chat := range chats {
		chatMap[chat.Jid] = chat
	}
	w.ReceivedChat <- struct{}{}
}

//HandleError needs to be implemented to be a valid WhatsApp handler
func (h *waHandler) HandleError(err error) {
	if e, ok := err.(*whatsapp.ErrConnectionFailed); ok {
		log.Printf("Connection failed, underlying error: %v", e.Err)
		log.Println("Waiting 30sec...")
		<-time.After(30 * time.Second)
		log.Println("Reconnecting...")
		err := h.c.Restore()
		if err != nil {
			log.Fatalf("Restore failed: %v", err)
		}
	} else {
		log.Printf("error occoured: %v\n", err)
	}
}

func main() {
	//create new WhatsApp connection
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating connection: %v\n", err)
		return
	}
	wac.SetClientVersion(2, 2121, 6)
	wac.SetClientName("Ubuntu", "Linux", "0.1.0")
	//load saved session
	session, err := readSession()
	if err == nil {
		//restore session
		session, err = wac.RestoreWithSession(session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "restoring failed: %v\n", err)
			return
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
			fmt.Fprintf(os.Stderr, "error during login: %v\n", err)
		}
	}
	err = writeSession(session)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error saving session: %v\n", err)
	}
	fmt.Printf("login successful, session: %v\n", session)

	//Add handler
	handler := &waHandler{wac, make(chan struct{}, 1)}
	wac.AddHandler(handler)

	// Get Chat list
	<-handler.ReceivedChat
	fmt.Printf("Listing chats\n")
	chatMap := make(map[int]string)
	i := 0
	for _, contact := range wac.Store.Chats {
		chatMap[i] = contact.Jid
		fmt.Printf("%d - %+v\n", i, contact)
		i++
	}
	fmt.Println("Please choose a chat number:")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read chat number due to %+v", err)
	}
	number, err := strconv.Atoi(input[:len(input)-1])
	if err != nil {
		fmt.Printf("Failed to convert chat number to Int %+v", err)
	}
	fmt.Println(chatMap[number], number, chatMap)
	ch, err := wac.DeleteChat(chatMap[number])
	if err != nil {
		fmt.Println("err:", err)
	}
	output := <-ch
	fmt.Println(output)
}

func readSession() (whatsapp.Session, error) {
	session := whatsapp.Session{}
	file, err := os.Open(os.TempDir() + "/whatsappSession.gob")
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

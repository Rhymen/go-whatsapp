package main

import (
	"fmt"
	"os"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Rhymen/go-whatsapp"
)

func main() {
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		panic(err)
	}

	// Check the actual whatsapp web version via: wac.CheckCurrentServerVersion()
	wac.SetClientVersion(2, 2134, 10)

	qr := make(chan string)
	go func() {
		terminal := qrcodeTerminal.New()
		terminal.Get(<-qr).Print()
	}()

	session, err := wac.Login(qr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error during login: %v\n", err)
		return
	}
	fmt.Printf("login successful, session: %v\n", session)
}

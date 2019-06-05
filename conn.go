//Package whatsapp provides a developer API to interact with the WhatsAppWeb-Servers.
package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gleandroj/go-whatsapp/binary"
	"github.com/gleandroj/go-whatsapp/crypto/cbc"

	"github.com/gorilla/websocket"
)

type metric byte

const (
	debugLog metric = iota + 1
	queryResume
	queryReceipt
	queryMedia
	queryChat
	queryContacts
	queryMessages
	presence
	presenceSubscribe
	group
	read
	chat
	received
	pic
	status
	message
	queryActions
	block
	queryGroup
	queryPreview
	queryEmoji
	queryMessageInfo
	spam
	querySearch
	queryIdentity
	queryUrl
	profile
	contact
	queryVcard
	queryStatus
	queryStatusUpdate
	privacyStatus
	queryLiveLocations
	liveLocation
	queryVname
	queryLabels
	call
	queryCall
	queryQuickReplies
)

type flag byte

const (
	ignore flag = 1 << (7 - iota)
	ackRequest
	available
	notAvailable
	expires
	skipOffline
)

// connection status. TODO: wac has isConnected method, consider integration
type ConnectionStatus struct {
	sync.RWMutex
	connected bool
}

func (s *ConnectionStatus) Connected() bool {
	s.RLock()
	defer s.RUnlock()
	return s.connected
}

func (s *ConnectionStatus) SetConnected(value bool) {
	s.Lock()
	s.connected = value
	s.Unlock()
}

func newConnectionStatus(initialStatus bool) *ConnectionStatus {
	return &ConnectionStatus{connected: initialStatus}
}

/*
Conn is created by NewConn. Interacting with the initialized Conn is the main way of interacting with our package.
It holds all necessary information to make the package work internally.
*/
type Conn struct {
	wsConn           *websocket.Conn
	wsConnOK         bool
	wsConnMutex      sync.RWMutex
	session          *Session
	listener         map[string]chan string
	listenerMutex    sync.RWMutex
	writeChan        chan wsMsg
	handler          []Handler
	msgCount         int
	msgTimeout       time.Duration
	Info             *Info
	Store            *Store
	ServerLastSeen   time.Time
	connectionStatus *ConnectionStatus

	timeTag string // last 3 digits obtained after a successful login takeover

	longClientName  string
	shortClientName string
	clientVersion   string

	loginSessionLock sync.RWMutex
	Proxy            func(*http.Request) (*url.URL, error)

	writerLock sync.RWMutex
}

type wsMsg struct {
	messageType int
	data        []byte
}

/*
Creates a new connection with a given timeout. The websocket connection to the WhatsAppWeb servers get´s established.
The goroutine for handling incoming messages is started
*/
func NewConn(timeout time.Duration) (*Conn, error) {
	return NewConnWithOptions(&Options{
		Timeout: timeout,
	})
}

// NewConnWithProxy Create a new connect with a given timeout and a http proxy.
func NewConnWithProxy(timeout time.Duration, proxy func(*http.Request) (*url.URL, error)) (*Conn, error) {
	return NewConnWithOptions(&Options{
		Timeout: timeout,
		Proxy:   proxy,
	})
}

// NewConnWithOptions Create a new connect with a given options.
type Options struct {
	Proxy           func(*http.Request) (*url.URL, error)
	Timeout         time.Duration
	Handler         []Handler
	ShortClientName string
	LongClientName  string
	ClientVersion   string
	Store           *Store
}

func NewConnWithOptions(opt *Options) (*Conn, error) {
	if opt == nil {
		return nil, ErrOptionsNotProvided
	}
	wac := &Conn{
		handler:         make([]Handler, 0),
		msgCount:        0,
		msgTimeout:      opt.Timeout,
		Store:           newStore(),
		longClientName:  "github.com/Rhymen/go-whatsapp",
		shortClientName: "go-whatsapp",
		clientVersion:   "0.1.0",
	}
	if opt.Handler != nil {
		wac.handler = opt.Handler
	}
	if opt.Store != nil {
		wac.Store = opt.Store
	}
	if opt.Proxy != nil {
		wac.Proxy = opt.Proxy
	}
	if len(opt.ShortClientName) != 0 {
		wac.shortClientName = opt.ShortClientName
	}
	if len(opt.LongClientName) != 0 {
		wac.longClientName = opt.LongClientName
	}
	if len(opt.ClientVersion) != 0 {
		wac.clientVersion = opt.ClientVersion
	}

	if err := wac.connect(); err != nil {
		return nil, err
	}

	wac.connectionStatus.SetConnected(true)

	go wac.readPump()
	go wac.writePump()
	go wac.keepAlive(20000, 90000)

	return wac, nil

}

func (wac *Conn) isConnected() bool {
	wac.wsConnMutex.RLock()
	defer wac.wsConnMutex.RUnlock()
	if wac.wsConn == nil {
		return false
	}
	if wac.wsConnOK {
		return true
	}

	// just send a keepalive to test the connection
	wac.sendKeepAlive()

	// this method is expected to be called by loops. So we can just return false
	return false
}

// connect should be guarded with wsConnMutex
func (wac *Conn) connect() error {
	dialer := &websocket.Dialer{
		ReadBufferSize:   0,
		WriteBufferSize:  0,
		HandshakeTimeout: wac.msgTimeout,
		Proxy:            wac.Proxy,
	}

	headers := http.Header{"Origin": []string{"https://web.whatsapp.com"}}

	server := strconv.Itoa(rand.Intn(8) + 1)
	wsConn, _, err := dialer.Dial("wss://w"+server+".web.whatsapp.com/ws", headers)

	if err != nil {
		return fmt.Errorf("couldn't dial whatsapp web websocket: %v", err)
	}

	wsConn.SetCloseHandler(func(code int, text string) error {
		_, _ = fmt.Fprintf(os.Stderr, "websocket connection closed(%d, %s)\n", code, text)

		// from default CloseHandler
		message := websocket.FormatCloseMessage(code, "")
		_ = wsConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))

		// our close handling
		if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			fmt.Println("Trigger reconnect")
			go wac.reconnect()
		} else {
			wac.connectionStatus.SetConnected(false)
		}
		return nil
	})

	wac.wsConn = wsConn
	wac.wsConnOK = true
	return nil
}

// reconnect should be run as go routine
func (wac *Conn) reconnect() {
	wac.wsConnMutex.Lock()
	_ = wac.wsConn.Close()
	wac.wsConn = nil
	wac.wsConnOK = false
	wac.wsConnMutex.Unlock()

	// wait up to 60 seconds and then reconnect. As writePump should send immediately, it might
	// reconnect as well. So we check its existance before reconnecting
	for !wac.isConnected() {
		time.Sleep(time.Duration(rand.Intn(60)) * time.Second)

		wac.wsConnMutex.Lock()
		if wac.wsConn == nil {
			if err := wac.connect(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "could not reconnect to websocket: %v\n", err)
			}
		}
		wac.wsConnMutex.Unlock()
	}
}

func (wac *Conn) write(data []interface{}) (<-chan string, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ts := time.Now().Unix()
	messageTag := fmt.Sprintf("%d.--%d", ts, wac.msgCount)
	msg := fmt.Sprintf("%s,%s", messageTag, d)

	ch := make(chan string, 1)

	wac.listenerMutex.Lock()
	wac.listener[messageTag] = ch
	wac.listenerMutex.Unlock()

	wac.writeChan <- wsMsg{websocket.TextMessage, []byte(msg)}

	wac.msgCount++
	return ch, nil
}

func (wac *Conn) writeBinary(node binary.Node, metric metric, flag flag, tag string) (<-chan string, error) {
	if len(tag) < 2 {
		return nil, fmt.Errorf("no tag specified or to short")
	}
	b, err := binary.Marshal(node)
	if err != nil {
		return nil, err
	}

	cipher, err := cbc.Encrypt(wac.session.EncKey, nil, b)
	if err != nil {
		return nil, err
	}

	h := hmac.New(sha256.New, wac.session.MacKey)
	h.Write(cipher)
	hash := h.Sum(nil)

	data := []byte(tag + ",")
	data = append(data, byte(metric), byte(flag))
	data = append(data, hash[:32]...)
	data = append(data, cipher...)

	ch := make(chan string, 1)

	wac.listenerMutex.Lock()
	wac.listener[tag] = ch
	wac.listenerMutex.Unlock()

	msg := wsMsg{websocket.BinaryMessage, data}
	wac.writeChan <- msg

	wac.msgCount++
	return ch, nil
}

func (wac *Conn) readPump() {
	defer wac.wsConn.Close()

	for {
		if !wac.connectionStatus.Connected() {
			break
		}

		msgType, msg, err := wac.wsConn.ReadMessage()
		if err != nil {
			wac.wsConnOK = false
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				wac.handle(fmt.Errorf("unexpected websocket close: %v", err))
			} else {
				wac.connectionStatus.SetConnected(false)
				break
			}
			// sleep for a second and retry reading the next message
			time.Sleep(time.Second)
			continue
		}
		wac.wsConnOK = true

		data := strings.SplitN(string(msg), ",", 2)

		//Kepp-Alive Timestmap
		if data[0][0] == '!' {
			msecs, err := strconv.ParseInt(data[0][1:], 10, 64)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error converting time string to uint: %v\n", err)
				continue
			}
			wac.ServerLastSeen = time.Unix(msecs/1000, (msecs%1000)*int64(time.Millisecond))
			continue
		}

		wac.listenerMutex.RLock()
		listener, hasListener := wac.listener[data[0]]
		wac.listenerMutex.RUnlock()

		if hasListener && len(data[1]) > 0 {
			listener <- data[1]

			wac.listenerMutex.Lock()
			delete(wac.listener, data[0])
			wac.listenerMutex.Unlock()
		} else if msgType == 2 && wac.session != nil && wac.session.EncKey != nil {
			message, err := wac.decryptBinaryMessage([]byte(data[1]))
			if err != nil {
				wac.handle(fmt.Errorf("error decoding binary: %v", err))
				continue
			}

			wac.dispatch(message)
		} else {
			if len(data[1]) > 0 {
				wac.handle(string(data[1]))
			}
		}

	}
}

func (wac *Conn) writePump() {
	for msg := range wac.writeChan {
		if !wac.connectionStatus.Connected() {
			break
		}

		for !wac.isConnected() {
			// reconnect to send the message ASAP
			wac.wsConnMutex.Lock()
			if wac.wsConn == nil {
				if err := wac.connect(); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "could not reconnect to websocket: %v\n", err)
				}
			}
			wac.wsConnMutex.Unlock()
			if !wac.isConnected() {
				// reconnecting failed. Sleep for a while and try again afterwards
				time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
			}
		}
		if err := wac.wsConn.WriteMessage(msg.messageType, msg.data); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error writing to socket: %v\n", err)
			wac.wsConnOK = false
			// add message to channel again to no loose it
			go func() {
				wac.writeChan <- msg
			}()
		}
	}
}

func (wac *Conn) sendKeepAlive() {
	// whatever issues might be there allow sending this message
	wac.wsConnOK = true
	wac.writeChan <- wsMsg{
		messageType: websocket.TextMessage,
		data:        []byte("?,,"),
	}
}

func (wac *Conn) AdminTest() (bool, error) {
	if !wac.connected {
		return false, ErrNotConnected
	}

	if !wac.loggedIn {
		return false, ErrInvalidSession
	}

	result, err := wac.sendAdminTest()
	return result, err
}

func (wac *Conn) keepAlive(minIntervalMs int, maxIntervalMs int) {
	for {
		if !wac.connectionStatus.Connected() {
			break
		}
		wac.sendKeepAlive()
		interval := rand.Intn(maxIntervalMs-minIntervalMs) + minIntervalMs
		<-time.After(time.Duration(interval) * time.Millisecond)
	}
}

// IsConnected returns whether the server connection is established or not
func (wac *Conn) IsConnected() bool {
	return wac.connected
}

// GetConnected returns whether the server connection is established or not
//
// Deprecated: function name is not go idiomatic, use IsConnected instead
func (wac *Conn) GetConnected() bool {
	return wac.connected
}

//IsLoggedIn returns whether the you are logged in or not
func (wac *Conn) IsLoggedIn() bool {
	return wac.loggedIn
}

// GetLoggedIn returns whether the you are logged in or not
//
// Deprecated: function name is not go idiomatic, use IsLoggedIn instead.
func (wac *Conn) GetLoggedIn() bool {
	return wac.loggedIn
}

package whatsapp

import (
	"github.com/Rhymen/go-whatsapp/binary"
	"strings"
	"sync"
	"errors"
)

type Store struct {
	Contacts map[string]Contact
	Chats    map[string]Chat
	sync.RWMutex
}

type Contact struct {
	Jid    string
	Notify string
	Name   string
	Short  string
}

type Chat struct {
	Jid             string
	Name            string
	Unread          string
	LastMessageTime string
	IsMuted         string
	IsMarkedSpam    string
}

func newStore() *Store {
	return &Store{
		make(map[string]Contact),
		make(map[string]Chat),
		sync.RWMutex{},
	}
}

func (wac *Conn) updateContacts(contacts interface{}) {
	c, ok := contacts.([]interface{})
	if !ok {
		return
	}
	defer wac.Store.Unlock()
	wac.Store.Lock()
	for _, contact := range c {
		contactNode, ok := contact.(binary.Node)
		if !ok {
			continue
		}

		jid := strings.Replace(contactNode.Attributes["jid"], "@c.us", "@s.whatsapp.net", 1)
		wac.Store.Contacts[jid] = Contact{
			jid,
			contactNode.Attributes["notify"],
			contactNode.Attributes["name"],
			contactNode.Attributes["short"],
		}
	}
}

func (wac *Conn) GetStoreContactList() map[string]Contact {

	defer wac.Store.RUnlock()
	wac.Store.RLock()

	return wac.Store.Contacts
}

func (wac *Conn) GetStoreContact(jid string) (Contact, bool) {

	defer wac.Store.RUnlock()
	wac.Store.RLock()

	if contact, ok := wac.Store.Contacts[jid]; ok {
		return contact, ok
	}

	return Contact{}, false
}

func (wac *Conn) AddStoreContact(contact Contact) error {

	if contact.Jid == "" {
		return errors.New("jit cannot be empty ")
	}

	defer wac.Store.Unlock()
	wac.Store.Lock()

	jid := strings.Replace(contact.Jid, "@c.us", "@s.whatsapp.net", 1)
	wac.Store.Contacts[jid] = contact

	return nil
}

func (wac *Conn) updateChats(chats interface{}) {
	c, ok := chats.([]interface{})
	if !ok {
		return
	}

	defer wac.Store.Unlock()
	wac.Store.Lock()

	for _, chat := range c {
		chatNode, ok := chat.(binary.Node)
		if !ok {
			continue
		}

		jid := strings.Replace(chatNode.Attributes["jid"], "@c.us", "@s.whatsapp.net", 1)
		wac.Store.Chats[jid] = Chat{
			jid,
			chatNode.Attributes["name"],
			chatNode.Attributes["count"],
			chatNode.Attributes["t"],
			chatNode.Attributes["mute"],
			chatNode.Attributes["spam"],
		}
	}
}

func (wac *Conn) GetStoreChatList() map[string]Chat {

	defer wac.Store.RUnlock()
	wac.Store.RLock()

	return wac.Store.Chats
}

func (wac *Conn) GetStoreChat(jid string) (Chat, bool) {

	defer wac.Store.RUnlock()
	wac.Store.RLock()

	if chat, ok := wac.Store.Chats[jid]; ok {
		return chat, ok
	}

	return Chat{}, false
}

func (wac *Conn) AddStoreChat(chat Chat) error {

	if chat.Jid == "" {
		return errors.New("jit cannot be empty ")
	}

	defer wac.Store.RUnlock()
	wac.Store.RLock()

	jid := strings.Replace(chat.Jid, "@c.us", "@s.whatsapp.net", 1)
	wac.Store.Chats[jid] = chat

	return nil
}

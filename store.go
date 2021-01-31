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

func (sr *Store) updateContacts(contacts interface{}) {
	c, ok := contacts.([]interface{})
	if !ok {
		return
	}
	defer sr.Unlock()
	sr.Lock()
	for _, contact := range c {
		contactNode, ok := contact.(binary.Node)
		if !ok {
			continue
		}

		jid := strings.Replace(contactNode.Attributes["jid"], "@c.us", "@s.whatsapp.net", 1)
		sr.Contacts[jid] = Contact{
			jid,
			contactNode.Attributes["notify"],
			contactNode.Attributes["name"],
			contactNode.Attributes["short"],
		}
	}
}

func (sr *Store) GetContacts() map[string]Contact {

	defer sr.RUnlock()
	sr.RLock()

	return sr.Contacts
}

func (sr *Store) GetContact(jid string) (Contact, bool) {

	defer sr.RUnlock()
	sr.RLock()

	if contact, ok := sr.Contacts[jid]; ok {
		return contact, ok
	}

	return Contact{}, false
}

func (sr *Store) AddContact(contact Contact) error {

	if contact.Jid == "" {
		return errors.New("jit cannot be empty ")
	}

	defer sr.Unlock()
	sr.Lock()

	jid := strings.Replace(contact.Jid, "@c.us", "@s.whatsapp.net", 1)
	sr.Contacts[jid] = contact

	return nil
}

func (sr *Store) updateChats(chats interface{}) {
	c, ok := chats.([]interface{})
	if !ok {
		return
	}

	defer sr.Unlock()
	sr.Lock()

	for _, chat := range c {
		chatNode, ok := chat.(binary.Node)
		if !ok {
			continue
		}

		jid := strings.Replace(chatNode.Attributes["jid"], "@c.us", "@s.whatsapp.net", 1)
		sr.Chats[jid] = Chat{
			jid,
			chatNode.Attributes["name"],
			chatNode.Attributes["count"],
			chatNode.Attributes["t"],
			chatNode.Attributes["mute"],
			chatNode.Attributes["spam"],
		}
	}
}

func (sr *Store) GetChats() map[string]Chat {

	defer sr.RUnlock()
	sr.RLock()

	return sr.Chats
}

func (sr *Store) GetChat(jid string) (Chat, bool) {

	defer sr.RUnlock()
	sr.RLock()

	if chat, ok := sr.Chats[jid]; ok {
		return chat, ok
	}

	return Chat{}, false
}

func (sr *Store) AddChat(chat Chat) error {

	if chat.Jid == "" {
		return errors.New("jit cannot be empty ")
	}

	defer sr.RUnlock()
	sr.RLock()

	jid := strings.Replace(chat.Jid, "@c.us", "@s.whatsapp.net", 1)
	sr.Chats[jid] = chat

	return nil
}

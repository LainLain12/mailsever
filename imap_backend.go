package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend"
	"golang.org/x/crypto/bcrypt"
)

type IMAPBackend struct {
	db *sql.DB
}

func NewIMAPBackend(db *sql.DB) *IMAPBackend {
	return &IMAPBackend{db: db}
}

func (b *IMAPBackend) Login(connInfo *imap.ConnInfo, username, password string) (backend.User, error) {
	var hashedPassword string
	var userID int
	err := b.db.QueryRow("SELECT id, password FROM users WHERE email = ?", username).
		Scan(&userID, &hashedPassword)
	if err != nil {
		return nil, errors.New("authentication failed")
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
		return nil, errors.New("authentication failed")
	}

	return &IMAPUser{
		username: username,
		userID:   userID,
		db:       b.db,
	}, nil
}

type IMAPUser struct {
	username string
	userID   int
	db       *sql.DB
}

func (u *IMAPUser) Username() string {
	return u.username
}

func (u *IMAPUser) ListMailboxes(subscribed bool) ([]backend.Mailbox, error) {
	return []backend.Mailbox{
		&IMAPMailbox{
			name:     "INBOX",
			username: u.username,
			db:       u.db,
		},
	}, nil
}

func (u *IMAPUser) GetMailbox(name string) (backend.Mailbox, error) {
	if name != "INBOX" {
		return nil, errors.New("mailbox not found")
	}
	return &IMAPMailbox{
		name:     name,
		username: u.username,
		db:       u.db,
	}, nil
}

func (u *IMAPUser) CreateMailbox(name string) error {
	return errors.New("operation not supported")
}

func (u *IMAPUser) DeleteMailbox(name string) error {
	return errors.New("operation not supported")
}

func (u *IMAPUser) RenameMailbox(existingName, newName string) error {
	return errors.New("operation not supported")
}

func (u *IMAPUser) Logout() error {
	return nil
}

type IMAPMailbox struct {
	name     string
	username string
	db       *sql.DB
}

func (m *IMAPMailbox) Name() string {
	return m.name
}

func (m *IMAPMailbox) Info() (*imap.MailboxInfo, error) {
	return &imap.MailboxInfo{
		Attributes: []string{},
		Delimiter:  "/",
		Name:       m.name,
	}, nil
}

func (m *IMAPMailbox) Status(items []imap.StatusItem) (*imap.MailboxStatus, error) {
	status := &imap.MailboxStatus{
		Name:     m.name,
		ReadOnly: false,
		Items:    make(map[imap.StatusItem]interface{}),
	}

	for _, item := range items {
		switch item {
		case imap.StatusMessages:
			var count int
			m.db.QueryRow("SELECT COUNT(*) FROM emails WHERE to_email = ?", m.username).Scan(&count)
			status.Items[imap.StatusMessages] = uint32(count)
		case imap.StatusUidNext:
			status.Items[imap.StatusUidNext] = uint32(1000)
		case imap.StatusUidValidity:
			status.Items[imap.StatusUidValidity] = uint32(1)
		case imap.StatusRecent:
			status.Items[imap.StatusRecent] = uint32(0)
		case imap.StatusUnseen:
			var count int
			m.db.QueryRow("SELECT COUNT(*) FROM emails WHERE to_email = ? AND read = FALSE", m.username).Scan(&count)
			status.Items[imap.StatusUnseen] = uint32(count)
		}
	}

	return status, nil
}

func (m *IMAPMailbox) SetSubscribed(subscribed bool) error {
	return nil
}

func (m *IMAPMailbox) Check() error {
	return nil
}

func (m *IMAPMailbox) ListMessages(uid bool, seqSet *imap.SeqSet, items []imap.FetchItem, ch chan<- *imap.Message) error {
	defer close(ch)

	rows, err := m.db.Query("SELECT id, from_email, subject, body, date FROM emails WHERE to_email = ? ORDER BY date DESC", m.username)
	if err != nil {
		return err
	}
	defer rows.Close()

	seqNum := uint32(1)
	for rows.Next() {
		var id int
		var from, subject, body, date string
		rows.Scan(&id, &from, &subject, &body, &date)

		if seqSet != nil && !seqSet.Contains(seqNum) {
			seqNum++
			continue
		}

		msg := &imap.Message{
			SeqNum: seqNum,
			Uid:    uint32(id),
		}

		for _, item := range items {
			switch item {
			case imap.FetchEnvelope:
				msg.Envelope = &imap.Envelope{
					Date:    time.Now(),
					Subject: subject,
					From:    []*imap.Address{{PersonalName: "", MailboxName: strings.Split(from, "@")[0], HostName: strings.Split(from, "@")[1]}},
					To:      []*imap.Address{{PersonalName: "", MailboxName: strings.Split(m.username, "@")[0], HostName: strings.Split(m.username, "@")[1]}},
				}
			case imap.FetchBodyStructure:
				msg.BodyStructure = &imap.BodyStructure{
					MIMEType:    "text",
					MIMESubType: "plain",
				}
			case imap.FetchFlags:
				msg.Flags = []string{}
			}
		}

		ch <- msg
		seqNum++
	}

	return nil
}

func (m *IMAPMailbox) SearchMessages(uid bool, criteria *imap.SearchCriteria) ([]uint32, error) {
	return []uint32{}, nil
}

func (m *IMAPMailbox) CreateMessage(flags []string, date time.Time, body imap.Literal) error {
	return errors.New("operation not supported")
}

func (m *IMAPMailbox) UpdateMessagesFlags(uid bool, seqset *imap.SeqSet, operation imap.FlagsOp, flags []string) error {
	return nil
}

func (m *IMAPMailbox) CopyMessages(uid bool, seqset *imap.SeqSet, destName string) error {
	return errors.New("operation not supported")
}

func (m *IMAPMailbox) Expunge() error {
	return nil
}

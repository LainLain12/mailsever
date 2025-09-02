package main

import (
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/emersion/go-smtp"
	"golang.org/x/crypto/bcrypt"
)

type SMTPBackend struct {
	db *sql.DB
}

func NewSMTPBackend(db *sql.DB) *SMTPBackend {
	return &SMTPBackend{db: db}
}

func (b *SMTPBackend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &SMTPSession{db: b.db}, nil
}

type SMTPSession struct {
	db   *sql.DB
	from string
	to   []string
}

func (s *SMTPSession) AuthPlain(username, password string) error {
	var hashedPassword string
	err := s.db.QueryRow("SELECT password FROM users WHERE email = ?", username).Scan(&hashedPassword)
	if err != nil {
		return errors.New("authentication failed")
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
		return errors.New("authentication failed")
	}

	return nil
}

func (s *SMTPSession) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *SMTPSession) Rcpt(to string) error {
	s.to = append(s.to, to)
	return nil
}

func (s *SMTPSession) Data(r io.Reader) error {
	// Read the email content
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	content := string(data)

	// Extract subject from headers
	subject := ""
	lines := strings.Split(content, "\n")
	bodyStart := 0

	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "subject:") {
			subject = strings.TrimSpace(line[8:])
		}
		if line == "" || line == "\r" {
			bodyStart = i + 1
			break
		}
	}

	// Extract body
	body := ""
	if bodyStart < len(lines) {
		body = strings.Join(lines[bodyStart:], "\n")
	}

	// Store email for each recipient
	for _, to := range s.to {
		_, err := s.db.Exec("INSERT INTO emails (from_email, to_email, subject, body) VALUES (?, ?, ?, ?)",
			s.from, to, subject, body)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SMTPSession) Reset() {
	s.from = ""
	s.to = nil
}

func (s *SMTPSession) Logout() error {
	return nil
}

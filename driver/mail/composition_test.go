package mail

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goforj/mail"
	"github.com/goforj/mail/mailmailgun"
	"github.com/goforj/mail/mailpostmark"
	"github.com/goforj/mail/mailresend"
	"github.com/goforj/mail/mailsendgrid"
	"github.com/goforj/mail/mailsmtp"
	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
)

// TestProviderComposition exercises real Mail drivers through the borrowed Mailer boundary.
func TestProviderComposition(t *testing.T) {
	cases := []struct {
		name, acknowledgement string
		status                int
		new                   func(string, *http.Client) (mail.Driver, error)
	}{
		{"resend", `{"id":"email-42"}`, 200, func(endpoint string, client *http.Client) (mail.Driver, error) {
			return mailresend.New(mailresend.Config{APIKey: "fixture", Endpoint: endpoint, HTTPClient: client})
		}},
		{"sendgrid", "", 202, func(endpoint string, client *http.Client) (mail.Driver, error) {
			return mailsendgrid.New(mailsendgrid.Config{APIKey: "fixture", Endpoint: endpoint, HTTPClient: client})
		}},
		{"mailgun", `{"id":"email-42","message":"Queued"}`, 200, func(endpoint string, client *http.Client) (mail.Driver, error) {
			return mailmailgun.New(mailmailgun.Config{Domain: "example.com", APIKey: "fixture", Endpoint: endpoint, HTTPClient: client})
		}},
		{"postmark", `{"MessageID":"email-42","ErrorCode":0,"Message":"OK"}`, 200, func(endpoint string, client *http.Client) (mail.Driver, error) {
			return mailpostmark.New(mailpostmark.Config{ServerToken: "fixture", Endpoint: endpoint, HTTPClient: client})
		}},
	}
	for _, entry := range cases {
		t.Run(entry.name, func(t *testing.T) {
			var reject atomic.Bool
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
				if err != nil || r.Method != "POST" || !strings.Contains(string(body), "recipient@example.com") || !strings.Contains(string(body), "sender@example.com") {
					t.Error("Mail configuration did not reach provider")
				}
				calls.Add(1)
				if reject.Load() {
					w.WriteHeader(503)
					_, _ = io.WriteString(w, "private provider response")
					return
				}
				w.WriteHeader(entry.status)
				_, _ = io.WriteString(w, entry.acknowledgement)
			}))
			defer server.Close()
			backend, err := entry.new(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			mailer := mail.New(backend, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
			driver := New(mailer, "recipient@example.com")
			notificationtest.Sender(t, func() notification.Sender { return driver })
			reject.Store(true)
			err = driver.Send(context.Background(), notification.Message{Title: "Ticket", Body: "Changed"})
			if !errors.Is(err, notification.ErrOutcomeUnknown) || strings.Contains(err.Error(), "private") {
				t.Fatal("unsafe email failure evidence", err)
			}
			reject.Store(false)
			if err := mailer.Send(context.Background(), mail.Message{To: []mail.Recipient{{Email: "recipient@example.com"}}, Subject: "Direct email", Text: "Mailer remains caller-owned"}); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 11 {
				t.Fatal("unexpected provider request count", calls.Load())
			}
		})
	}
}

// TestSMTPComposition verifies real SMTP envelope and MIME handling through the adapter.
func TestSMTPComposition(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	captures := make(chan string, 16)
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go receiveSMTP(connection, captures)
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := mailsmtp.New(mailsmtp.Config{Host: host, Port: port})
	if err != nil {
		t.Fatal(err)
	}
	mailer := mail.New(backend, mail.WithDefaultFrom("sender@example.com", "Helpdesk"))
	driver := New(mailer, "recipient@example.com")
	notificationtest.Sender(t, func() notification.Sender { return driver })
	for i := 0; i < 9; i++ {
		data := <-captures
		if !strings.Contains(data, "sender@example.com") || !strings.Contains(data, "recipient@example.com") || !strings.Contains(data, "T-42 changed") {
			t.Fatal("SMTP envelope or message changed")
		}
	}
}

// receiveSMTP implements the local protocol fixture without replacing Mail's SMTP transport.
func receiveSMTP(connection net.Conn, captures chan<- string) {
	defer connection.Close()
	protocol := textproto.NewConn(connection)
	_ = protocol.PrintfLine("220 localhost ESMTP fixture")
	envelope := ""
	for {
		command, err := protocol.ReadLine()
		if err != nil {
			return
		}
		switch {
		case strings.HasPrefix(command, "EHLO"), strings.HasPrefix(command, "HELO"):
			_ = protocol.PrintfLine("250 localhost")
		case strings.HasPrefix(command, "MAIL FROM:"), strings.HasPrefix(command, "RCPT TO:"):
			envelope += command + "\n"
			_ = protocol.PrintfLine("250 OK")
		case command == "DATA":
			_ = protocol.PrintfLine("354 send message")
			body, err := io.ReadAll(protocol.DotReader())
			if err != nil {
				return
			}
			captures <- envelope + string(body)
			_ = protocol.PrintfLine("250 queued")
		case command == "QUIT":
			_ = protocol.PrintfLine("221 bye")
			return
		default:
			_ = protocol.PrintfLine("250 OK")
		}
	}
}

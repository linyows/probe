package main

import (
	"context"
	"log"
	"net"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
)

func main() {
	s := imapserver.New()

	s.HandleLogin(func(conn *imapserver.Conn, cmd *imap.Command) *imap.Response {
		username, _ := cmd.Arguments[0].AsString()
		password, _ := cmd.Arguments[1].AsString()

		if username != "hello@example.local" || password != "secret" {
			return &imap.Response{
				Tag:   cmd.Tag,
				State: imap.RespNo,
				Info:  imap.String("Invalid credentials"),
			}
		}

		log.Printf("User %s logged in", username)
		conn.SetUser(&MockUser{Username: username})

		return &imap.Response{
			Tag:   cmd.Tag,
			State: imap.RespOK,
			Info:  imap.String("Login successful"),
		}
	})

	s.HandleLogout(func(conn *imapserver.Conn, cmd *imap.Command) *imap.Response {
		log.Printf("Client logged out: %v", conn)
		return &imap.Response{
			Tag:   cmd.Tag,
			State: imap.RespOK,
			Info:  imap.String("Bye"),
		}
	})

	listener, err := net.Listen("tcp", ":993")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Println("IMAP server started on :1143")

	if err := s.Serve(listener); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

type MockUser struct {
	Username string
}

func (u *MockUser) Name() string {
	return u.Username
}

func (u *MockUser) Context() context.Context {
	return context.Background()
}

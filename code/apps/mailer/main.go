package main

import (
	"log/slog"
	"os"
	"time"

	email "github.com/goblimey/go-simple-mail" // Package email
)

func main() {
	const fromAddr = "johnsmith@gmail.com"
	const toAddr = "janesmith@gmail.com"
	const subject = "the email subject"

	// Create an optional slog logger.  The Config type supplies methods such as LogError
	// which check for a nil logger before trying to use it.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Configure the email connection.  This one connects to John Smith's
	// gmail account via a TLS connection, authenticating with his Google password.
	cfg := email.NewConfigWithLogger(
		"smtp.gmail.com", // mail server.
		465,              // Port.
		fromAddr,
		"googlepassword", // This should be supplied as a secret.
		true,             // Secure connection.
		true,             // Secured via TLS.
		5*time.Second,    // Connection timeout.
		logger,
	)

	connection, connError := email.Connect(cfg)
	if connError != nil {
		cfg.LogError(connError.Error())
		return
	}
	defer connection.Close()

	attachment, attError := email.NewAttachment("./attachment.txt", "attachment.txt")
	if attError != nil {
		cfg.LogError(attError.Error())
		return
	}

	attachments := []email.Attachment{*attachment}

	const body = "<html><h1>this is a heading</h1>this is the running text</html>"

	em := email.NewEmail(cfg, fromAddr, toAddr, "the subject", body, attachments)

	if err := em.Send(connection); err != nil {
		cfg.LogError(err.Error())
		return
	}

	connection.Close()
}

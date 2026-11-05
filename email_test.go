package email

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gabriel-vasile/mimetype" // Package mimetype.
	"github.com/wneessen/go-mail"        // Package mail.
)

// TestConnectAndSend is an integration test which checks Connect and Send by sending
// several emails to the configured address.  To run it, you need to edit the config and
// the constants "fromAddr" and "toAddr" to suit your environment and uncomment the
// commented out calls.  Left as it is, the test runs to completion, so doesn't break
// the test suite, but does nothing useful.
func TestConnectAndSend(t *testing.T) {

	const fromAddr = "somebody@example.com"        // fix this.
	const toAddr = "somebody.else@some.domain.com" // Fix this.

	const body1 = "<html><h1>this is test 1</h1>this is the rest</html>"
	const body2 = "<html><h1>this is test 2</h1>this is the rest</html>"
	const body3 = "<html><h1>this is test 3</h1>this is the rest</html>"

	const wantContentType = "text/html; charset=utf-8"

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// This config assumes that your email service uses a TLS connection, not SSL or
	// plain text.  The connection timeout should be sufficient but not too long, as that
	// could slow down a failing test.  It typically takes about two seconds to connect to
	// an email server, authenticate and send an email, so a timeout of 5 seconds should
	// be enough.  If not, you can set a larger value.
	cfg := NewConfigWithLogger(
		"mail.example.com",
		465,
		fromAddr,
		"MyEmailPassword", // Fix this.
		true,
		true,
		5*time.Second,
		logger,
	)

	// Attach the file "atch.txt" from the current directory and tell the receiving server
	// to use the same name when it downloads it.
	attachment, attError := NewAttachment("./atch.txt", "atch.txt")
	if attError != nil {
		t.Error(attError)
		return
	}

	attachments := []Attachment{*attachment}

	em1 := NewEmail(cfg, fromAddr, "", toAddr, "test email 1", body1, attachments)

	// This allows the previous statements to run when the calls that use it
	// are commented out.
	_ = em1

	/*
	 * These calls should be commented out until you set the configuration to match
	 * for your environment.
	 */

	// connection, connError := Connect(cfg)
	// if connError != nil {
	// 	cfg.LogError(connError.Error())
	// 	return
	// }
	// defer connection.Close()

	// if err := em1.Send(connection); err != nil {
	// 	cfg.LogError(err.Error())
	// 	return
	// }

	// if em1.BodyContentType != wantContentType {
	// 	t.Errorf("want %s got %s", wantContentType, em1.BodyContentType)
	// }

	// Work out and supply the body type of this email.
	mimeTypeOfBody2 := mimetype.Detect([]byte(body2))
	contentTypeOfBody2 := mail.ContentType(mimeTypeOfBody2.String())

	em2 := NewEmailWithBodyContentType(cfg, fromAddr, "", toAddr, "test email 2", body2,
		attachments, contentTypeOfBody2)

	_ = em2

	// if err := em2.Send(connection); err != nil {
	// 	cfg.LogError(err.Error())
	// 	return
	// }
	// connection.Close()
}

// TestNewAttachment checks NewAttachment using the file atch.txt, which has a known
// content type.
func TestNewAttachment(t *testing.T) {

	const want = "text/plain; charset=utf-8"

	at, err := NewAttachment("./atch.txt", "atch.txt")
	if err != nil {
		t.Error(err)
		return
	}

	if at.Filename != "./atch.txt" {
		t.Errorf("want ./atch.txt, got %s", at.Filename)
	}

	if at.OriginalFilename != "atch.txt" {
		t.Errorf("want atch.txt, got %s", at.OriginalFilename)
	}

	if at.ContentType != want {
		t.Errorf("want %s got %s", want, at.ContentType)
	}
}

// TestNewAttachmentWithJunkFile checks that NewAttachment returns the correct error when the file
// does not exist.
func TestNewAttachmentWithJunkFile(t *testing.T) {

	// It appears that fs.Open() returns whatever message it gets from the underlying operating system,
	// so the result may not be consistent across different systems.  We assume that it will contain
	// this text.
	const want = "cannot find the file"

	_, err := NewAttachment("junk", "junk")
	if err == nil {
		t.Error("expected an error")
		return
	}

	if !strings.Contains(err.Error(), want) {
		t.Errorf("got unexpected error %v", err)
	}
}

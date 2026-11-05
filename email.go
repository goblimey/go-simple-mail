// email provides a simple interface to W Neeson's go-mail package.  Unlike many Go
// email packages, go-mail is well tested, well maintained and supports the use of
// attachments.  However, it offers little documentation other than the unit tests
// and I had tomspen quite a lot of time figuring out how to use it.
//
// Email servers offer various styles of connection and if you are using a
// commercial service you may not have any choice which one is used.  This package
// supports a plain text connection, a TLS connection or an SSL connection, all
// secured by user name and password.  Other styles of connection are not supported.
// If you need a different style, you need to call go-mail directly, but this package
// may be a useful guide as to how to do that.
package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/wneessen/go-mail" // "mail."
)

// SendErrorLeader is the start of a error message which can be watched for
// by the calling package.  It indicates that the mail failed at the
// sending stage, which would not necessarily stop other emails being sent,
// rather than an earlier when the connection to the server was set up.
const SendErrorLeader = "email.Send: failed to send mail to "

type Config struct {
	Host              string        // The email server.
	Port              int           // The email port on the email server.
	User              string        // The account used on the email server.
	Password          string        // The password of the sending account.
	SecureConnection  bool          // The connection is secure (TLS or SSL).
	TLS               bool          //The connection is secured using TLS - if false, then by SSL.
	ConnectionTimeout time.Duration // The email timeout as a duration.
	Logger            *slog.Logger  // The optional logger.
}

func NewConfigWithLogger(host string, port int, user, password string, secure, tls bool, timeout time.Duration, logger *slog.Logger) *Config {

	cfg := NewConfig(host, port, user, password, secure, tls, timeout)
	cfg.Logger = logger

	return cfg
}

func NewConfig(host string, port int, user, password string, secure, tls bool, timeout time.Duration) *Config {

	cfg := Config{
		Host:              host,
		Port:              port,
		User:              user,
		Password:          password,
		ConnectionTimeout: timeout,
		SecureConnection:  secure,
		TLS:               tls,
	}

	return &cfg
}

// LogInfo logs the string at Info level.
func (cfg *Config) LogInfo(s string) {
	if cfg.Logger != nil {
		cfg.Logger.Info(s)
	}
}

// LogDebug logs the string at Debug level.
func (cfg *Config) LogDebug(s string) {
	if cfg.Logger != nil {
		cfg.Logger.Debug(s)
	}
}

// LogError logs the string at Error level.
func (cfg *Config) LogError(s string) {
	if cfg.Logger != nil {
		cfg.Logger.Error(s)
	}
}

type Connection struct {
	Cfg    *Config      // The configuration.
	Client *mail.Client // The running SMTP client.
}

func Connect(cfg *Config) (*Connection, error) {

	// tlsConfig is used if the connection is secure.
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         cfg.Host,
	}

	var client *mail.Client
	var createClientError error

	switch {
	// It appears that SSLPort should only be set true if the connection is
	// SSL, not when it's TLS.
	case cfg.SecureConnection && cfg.TLS:
		// The connection is secured by TLS (which is the modern convention).
		// I have tested this configuration with a live server.
		client, createClientError = mail.NewClient(
			cfg.Host,
			mail.WithPort(cfg.Port),
			mail.WithSSLPort(false),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.User),
			mail.WithPassword(cfg.Password),
			mail.WithTimeout(cfg.ConnectionTimeout),
			mail.WithTLSConfig(tlsConfig),
			mail.WithTLSPolicy(mail.TLSMandatory),
		)

	case cfg.SecureConnection && !cfg.TLS:
		// The connection is secured by SSL.  I don't have access to a mail server
		// that works this way so I have not tested this configuration with a
		// live server.
		client, createClientError = mail.NewClient(
			cfg.Host,
			mail.WithPort(cfg.Port),
			mail.WithSSLPort(true),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.User),
			mail.WithPassword(cfg.Password),
			mail.WithTimeout(cfg.ConnectionTimeout),
			mail.WithTLSConfig(tlsConfig),
			mail.WithTLSPolicy(mail.TLSMandatory),
		)

	default:
		// The connection is not encrypted, which is sometimes done for
		// testing, but rarely in real applications these days, because a
		// plain text connection is not secure.
		client, createClientError = mail.NewClient(
			cfg.Host,
			mail.WithPort(cfg.Port),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(cfg.User),
			mail.WithPassword(cfg.Password),
			mail.WithTimeout(cfg.ConnectionTimeout),
		)
	}
	if createClientError != nil {
		return nil, createClientError
	}

	connection := Connection{Cfg: cfg, Client: client}
	return &connection, nil
}

// Close the client.
func (c *Connection) Close() error {
	defer c.completeClose()
	if c.Client != nil {
		if err := c.Client.Close(); err != nil {
			c.Cfg.LogError(fmt.Sprintf("connection.Close: %v", err))
			return err
		}
	}
	return nil
}

// completeClose sets the client in the given connection to nil.
func (c *Connection) completeClose() {
	c.Client = nil
}

// Attachment contains an attachment and its content type.  Files to be attached are stored
// in the directory given by the config.  The name here is relative to that directory.
type Attachment struct {
	Filename         string           // The path name of the stored file on this server.
	OriginalFilename string           // The suggested file name to be used when downloaded by the receiving mail client.
	ContentType      mail.ContentType // The content type of the file.
}

// NewAttachment creates and returns a new Attachment.  The filename is the pathname of
// the attached file on the local machine.  The original filename is the suggested name
// that the receiving email client might use to store the file when downloaded.  The
// function opens and reads the file to figure out the content type.  That overhead is
// avoided if you use NewAttachmentWithContentType instead.  Finding the content type
// can produce an error, for example if the file doesn't exist.  Any such error is
// returned.
func NewAttachment(filename, originalFilename string) (*Attachment, error) {

	// Find and set the content type.
	contentType, conError := ContentTypeOfFile(filename)
	if conError != nil {
		return nil, conError
	}

	at := NewAttachmentWithContentType(filename, originalFilename, contentType)

	return at, nil
}

// NewAttachmentWithContentType creates and returns a new Attachment.  This is an
// alternative to NewAttachment, useful if the content type of the file is known, as it
// avoid the overhead of figuring it out.  The given filename is the pathname of the
// file to be attached on the local machine.  The given originalFilename is the suggested
// name that the receiving email client might use to store the file when downloaded.  The given
// contentType is the content type of the file (plain text, HTML or whatever) as defined
// by the package github.com/gabriel-vasile/mimetype.  It's the responsibility of the
// caller to get the content type right.
func NewAttachmentWithContentType(filename, originalFilename string, contentType mail.ContentType) *Attachment {

	at := Attachment{
		Filename:         filename,
		OriginalFilename: originalFilename,
		ContentType:      contentType,
	}

	return &at
}

// Email represents a single email - From address, To address, subject etc.
type Email struct {
	Cfg             *Config
	Subject         string
	From            string
	To              string
	ReplyTo         string
	Body            string
	BodyContentType mail.ContentType
	Attachments     []Attachment
}

// // NewMail creates an Email and sets the content type of the body.
func NewEmail(cfg *Config, from, replyTo, to, subject, body string, attachments []Attachment) *Email {

	em := Email{
		Cfg:             cfg,
		Subject:         subject,
		From:            from,
		ReplyTo:         replyTo,
		To:              to,
		Body:            body,
		BodyContentType: "",
		Attachments:     attachments,
	}
	return &em
}

// NewMailWithBodyContentType creates an Email and sets the content type of the body.
func NewEmailWithBodyContentType(cfg *Config, from, replyTo, to, subject, body string,
	attachments []Attachment, bodyContentType mail.ContentType) *Email {

	em := NewEmail(cfg, from, replyTo, to, subject, body, attachments)

	em.BodyContentType = bodyContentType

	return em
}

// Send sends the email.
func (em *Email) Send(connection *Connection) error {

	// Create a wneesson mail.
	m := mail.NewMsg()

	m.Subject(em.Subject)

	if err := m.From(em.From); err != nil {
		return err
	}

	if err := m.ReplyTo(em.ReplyTo); err != nil {
		return err
	}

	if err := m.To(em.To); err != nil {
		return err
	}

	// In some cases, but not all,  it makes sense to have the caller set the
	// content type of the body.  If that wasn't done, set it now.
	if len(em.BodyContentType) <= 0 {
		mimeTypeOfBody := mimetype.Detect([]byte(em.Body))
		em.BodyContentType = mail.ContentType(mimeTypeOfBody.String())
	}

	m.SetBodyString(em.BodyContentType, em.Body)

	for i, at := range em.Attachments {
		if len(em.Attachments[i].ContentType) <= 0 {
			var conErr error
			em.Attachments[i].ContentType, conErr = ContentTypeOfFile(em.Attachments[i].Filename)
			if conErr != nil {
				return conErr
			}
		}
		m.AttachFile(
			at.Filename,
			mail.WithFileName(at.OriginalFilename),
			mail.WithFileContentType(em.Attachments[i].ContentType),
		)
	}

	if err := connection.Client.DialAndSend(m); err != nil {
		return fmt.Errorf(SendErrorLeader+"%s: %v\n", em.To, err)
	}

	return nil
}

// ContentTypeOfFile returns the content type of the file with the given name.
func ContentTypeOfFile(filename string) (mail.ContentType, error) {
	var openError error
	var f *os.File
	f, openError = os.Open(filename)
	if openError != nil {
		return "", openError
	}
	defer f.Close()
	var detectError error
	var mimeTypeOfAttachment *mimetype.MIME
	mimeTypeOfAttachment, detectError = mimetype.DetectReader(f)
	if detectError != nil {
		return "", detectError
	}
	result := mail.ContentType(mimeTypeOfAttachment.String())

	return result, nil
}

go-simple-mail
======

A simple email client package written in Go

There are lots of email client packages available 
for Go but many of them are no longer supported (with issues open for a long time) 
and/or badly tested.
Some don't support attachments.

The package github.com/wneessen/go-mail is well supported, well tested and supports attachments.  However, it's not very well documented.  I had to trawl through the unit tests to figure out how to use it.

This package is a wrapper around the wneesson package which I claim is easier to understand and use. 

The wneesson package has very few dependencies and I've avoided adding unnecessary ones in this package.  It needs to figure out the content type of the body text and the attachment files - plain text, HTML or whatever.  To do that  use github.com/gabriel-vasile/mimetype.  

Email servers offer various styles of connection and if you are using a
commercial service you may not have any choice which one is used.  This package
supports a plain text connection, a TLS connection or an SSL connection, all
secured by user name and password.  I only have mail servers available that offer TLS connections,
which is the standard these days,
so that's the only one that I can test.

Other styles of connection are not supported here.
If you need a different style you need
to call wneesson's go-mail directly.  This package may be a useful guide as to how to do that.

Installation
---

```
    git clone https://github.com/goblimey/go-simple-mail.git
    cd go-simple-mail
    go get github.com/wneessen/go-mail
    go get github.com/gabriel-vasile/mimetype
```

Handling Secrets
-----

The example below is nice and simple 
but it doesn't handle its secrets well.

Secrets are values that need to be kept secret - 
for example, user names and passwords.
There is lots of advice out there about handling them securely

It's worth pointing out that if the source code of a program 
contains bits of text that contain secrets 
(as in the example)
it's possible to find them in the compiled binary and extract them.
If somebody breaks into your system and they can read the binary file,
they can extract any secrets embedded in it.

The general advice is that the secrets should be fed into the program 
at run time via environment variables.
For extra security, these can be set from an external secrets store.

For more detailed advice see [this Reddit thread](https://www.reddit.com/r/golang/comments/szhapw/passwords_secrets_keys_best_practice/)

You don't want to leave secret text lying around in the program's
memory when it's no longer needed.  For example, 
if your program opens a connection to the mail server just once at the start
and uses it for the rest of the run, 
you don't need the password once the connection is open.
Clearing it out is not as easy as it looks in Go.
See [this package](https://pkg.go.dev/runtime/secret)
for how to do it properly.


Sending an email with an attachment
----------------

```
    import (
        "os"
        "log/slog"
        "time"
        "github.com/goblimey/go-simple-mail" // Package email
    )
```

```go
    const fromAddr = "johnsmith@gmail.com"
    const toAddr = "janesmith@gmail.com"
    const subject = "the email subject"

    // Create an optional slog logger.  The Config type supplies methods such as LogError 
    // which check for a nil logger before trying to use it.
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    // Configure the email connection.  This one connects to John Smith's
    // gmail account via a TLS connection, authenticating with his Google password.
    cfg := email.NewConfigWithLogger(
        "smtp.gmail.com",   // mail server.
        465,                // Port.
        fromAddr, 
        "googlepassword",   // This should be supplied as a secret.
        true,               // Secure connection.
        true,               // Secured via TLS.
        5 * time.Second,    // Connection timeout.
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
```

Content Type
------------

The underlying mailer github.com/wneessen/go-mail needs to be given the content
type of the body of the email and any attached files.

This package uses github.com/gabriel-vasile/mimetype to figure out content types, but if the content is supplied as a file, that involves opening the file and reading it.

The calling program can create an email object and use it over and over, for example to send the same email to more than one address.
The first call of Send figures out the content type and stores it in the email object.
Subsequent calls pick up the content type and avoid the overhead of finding it again. 
Similarly, NewAttachment figures out the content type of the attached file and stores it in the Attachment object for reuse.  

If you know the content type you can supply it.
Create the email object with NewEmailWithBodyContentType instead of NewEmail.

The NewAttachmentWithContentType creates an Attachment object and allows you to set the content type to your chosen value.

To figure out the content type of a string, you need some extra imports:

```go
    import (
		"os"
        "log/slog"
        "time"
        "github.com/goblimey/go-simple-mail" // Package email
        "github.com/wneessen/go-mail"        // Package mail.
        "github.com/gabriel-vasile/mimetype" // Package mimetype.
    )
```

Then get the content type like so:

```go
    const body = "<html><h1>this is a heading</h1>this is the running text</html>"
    mimeTypeOfBody := mimetype.Detect([]byte(body))
    contentTypeOfBody := mail.ContentType(mimeTypeOfBody.String())
```

   To figure out the content type of a file, given the name in filename,
   open the file and use DetectReader:

```go
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
```

Whichever account you connect with, many email providers restrict the sending
addresses you can use.
Maybe you have to use the same address 
or maybe you have to use an address in the same domain.
If you want replies to go to some other email address, use ReplyTo.

When you send an email with an attachment, 
you specify the pathname of the file on your machine
and the name that you suggest that the receiving email client might use when
it downloads the file.
In this example, the local file is "./atch.txt" and 
the suggested name is "atch.txt".
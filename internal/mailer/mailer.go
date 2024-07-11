package mailer

import (
	"bytes"
	"embed"
	"text/template"
	"time"

	"github.com/go-mail/mail/v2"
)

// Declare a variable with type embed.FS to hold the email templates.
// This has comment directive in the format `//go:embed <path>` above it, to indicate
// that we want to store the contents of the ./templates directory in the templateFS embedded file system variable.

//go:embed "templates"
var templateFS embed.FS

// Mailer struct definition which contains a mail.Dialer instance (used to connect to the SMTP server),
// and the sender information for the email.
type Mailer struct {
	dialer *mail.Dialer
	sender string
}

func New(host string, port int, username, password, sender string) Mailer {
	dialer := mail.NewDialer(host, port, username, password)
	dialer.Timeout = 5 * time.Second

	return Mailer{
		dialer: dialer,
		sender: sender,
	}
}

// Send() method on the Mailer type. This takes the recipient email address, name of the file containing the templates,
// and any dynamic data for the templates as an interface{} parameter.
func (m Mailer) Send(recipient, templateFile string, data interface{}) error {
	// Use the ParseFS() method to parse the required template file from the embedded file system.
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	// Execute the named template/s "subject/plainBody/htmlBody", passing in the dynamic data and storing the result in a
	// bytes.Buffer variable.
	subject := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(subject, "subject", data)
	if err != nil {
		return err
	}

	plainBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(plainBody, "plainBody", data)
	if err != nil {
		return err
	}

	htmlBody := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(htmlBody, "htmlBody", data)
	if err != nil {
		return err
	}

	// Use the mail.NewMessage() function to initialize a new mail.
	// Note: AddAlternative should always be called after SetBody.
	msg := mail.NewMessage()
	msg.SetHeader("To", recipient)
	msg.SetHeader("From", m.sender)
	msg.SetHeader("Subject", subject.String())
	msg.SetBody("text/plain", plainBody.String())
	msg.AddAlternative("text/html", htmlBody.String())

	// Call the DialAndSend() method on the dialer to connect to the SMTP server and send the email.
	// This opens a connection to the SMTP server, sends the message, then closes the connection.
	// If there is a timeout, it will return an error.
	err = m.dialer.DialAndSend(msg)
	if err != nil {
		return err
	}

	return nil
}

package mail

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/email_verification.txt.tmpl
var emailVerificationTextTemplate string

//go:embed templates/email_verification.html.tmpl
var emailVerificationHTMLTemplate string

type EmailVerification struct {
	Username string
	Email    string
	Token    string
}

func (m EmailVerification) Envelope() Envelope {
	return Envelope{
		To:      []Address{{Name: m.Username, Address: m.Email}},
		Subject: "Verify your email",
	}
}

func (m EmailVerification) Content() (Content, error) {
	data := struct {
		Username string
		Token    string
	}{Username: m.Username, Token: m.Token}
	text, err := renderVerificationTemplate("email-verification.txt", emailVerificationTextTemplate, data)
	if err != nil {
		return Content{}, err
	}
	html, err := renderVerificationTemplate("email-verification.html", emailVerificationHTMLTemplate, data)
	if err != nil {
		return Content{}, err
	}
	return Content{Text: text, HTML: html}, nil
}

func (EmailVerification) Attachments() []Attachment { return nil }

func renderVerificationTemplate(name, source string, data any) (string, error) {
	tmpl, err := template.New(name).Parse(source)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return "", err
	}
	return output.String(), nil
}

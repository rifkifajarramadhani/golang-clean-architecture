package mail

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed templates/welcome.txt.tmpl
var welcomeTextTemplate string

//go:embed templates/welcome.html.tmpl
var welcomeHTMLTemplate string

type Welcome struct {
	Username string
	Email    string
}

func (m Welcome) Envelope() Envelope {
	return Envelope{
		To:      []Address{{Name: m.Username, Address: m.Email}},
		Subject: "Welcome!",
	}
}

func (m Welcome) Content() (Content, error) {
	data := struct {
		Username string
	}{Username: m.Username}

	text, err := renderTemplate("welcome.txt", welcomeTextTemplate, data)
	if err != nil {
		return Content{}, err
	}
	html, err := renderTemplate("welcome.html", welcomeHTMLTemplate, data)
	if err != nil {
		return Content{}, err
	}
	return Content{Text: text, HTML: html}, nil
}

func (Welcome) Attachments() []Attachment { return nil }

func renderTemplate(name, source string, data any) (string, error) {
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

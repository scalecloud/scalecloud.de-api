package emailmanager

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

const (
	TemplateNewsletterConfirmation = "newsletter-confirmation.html"
	TemplateSeatInvitation         = "seat-invitation.html"
)

type NewsletterConfirmationTemplateData struct {
	Email            string
	ConfirmationLink string
}

type SeatInvitationTemplateData struct {
	ProductName  string
	ProductType  string
	InviterName  string
	InviterEmail string
	StorageQuota string
	ClaimURL     string
	SupportEmail string
}

//go:embed templates/*.html
var templateFiles embed.FS

var emailTemplates = template.Must(template.ParseFS(templateFiles, "templates/*.html"))

func RenderTemplate(name string, data any) (string, error) {
	var rendered bytes.Buffer
	if err := emailTemplates.ExecuteTemplate(&rendered, name, data); err != nil {
		return "", fmt.Errorf("render email template %q: %w", name, err)
	}
	return rendered.String(), nil
}

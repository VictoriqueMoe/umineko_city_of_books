package notification

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.tmpl"))

type NotifEmailData struct {
	Subject   string
	ActorName string
	Action    string
	Title     string
	LinkURL   string
	SiteName  string
}

type ReportEmailData struct {
	ReporterName string
	TargetType   string
	Reason       string
	LinkURL      string
	SiteName     string
}

type ReportResolvedEmailData struct {
	ResolverName string
	TargetType   string
	Comment      string
	LinkURL      string
	SiteName     string
}

type PasswordResetEmailData struct {
	SiteName string
	LinkURL  string
}

type VerificationEmailData struct {
	SiteName string
	LinkURL  string
}

type EmailChangedEmailData struct {
	SiteName string
	NewEmail string
	LinkURL  string
}

func render(name, subject string, data any) (string, string) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return subject, fmt.Sprintf("<p>%s</p>", subject)
	}

	return subject, buf.String()
}

func notifEmail(actorName, action, title, linkURL, siteName string) (subject string, body string) {
	subject = fmt.Sprintf("%s %s", actorName, action)

	data := NotifEmailData{
		Subject:   subject,
		ActorName: actorName,
		Action:    action,
		Title:     title,
		LinkURL:   linkURL,
		SiteName:  siteName,
	}

	return render("notification.tmpl", subject, data)
}

func reportEmail(reporterName, targetType, reason, linkURL, siteName string) (subject string, body string) {
	subject = fmt.Sprintf("New report from %s", reporterName)

	data := ReportEmailData{
		ReporterName: reporterName,
		TargetType:   targetType,
		Reason:       reason,
		LinkURL:      linkURL,
		SiteName:     siteName,
	}

	return render("report.tmpl", subject, data)
}

func reportResolvedEmail(resolverName, targetType, comment, linkURL, siteName string) (subject string, body string) {
	subject = "Your report has been resolved"

	data := ReportResolvedEmailData{
		ResolverName: resolverName,
		TargetType:   targetType,
		Comment:      comment,
		LinkURL:      linkURL,
		SiteName:     siteName,
	}

	return render("report_resolved.tmpl", subject, data)
}

func PasswordResetEmail(siteName, linkURL string) (subject string, body string) {
	subject = fmt.Sprintf("Reset your %s password", siteName)

	data := PasswordResetEmailData{
		SiteName: siteName,
		LinkURL:  linkURL,
	}

	return render("password_reset.tmpl", subject, data)
}

func EmailChangedEmail(siteName, newEmail, linkURL string) (subject string, body string) {
	subject = fmt.Sprintf("Your %s email address was changed", siteName)

	data := EmailChangedEmailData{
		SiteName: siteName,
		NewEmail: newEmail,
		LinkURL:  linkURL,
	}

	return render("email_changed.tmpl", subject, data)
}

func VerificationEmail(siteName, linkURL string) (subject string, body string) {
	subject = fmt.Sprintf("Confirm your %s email", siteName)

	data := VerificationEmailData{
		SiteName: siteName,
		LinkURL:  linkURL,
	}

	return render("verify_email.tmpl", subject, data)
}

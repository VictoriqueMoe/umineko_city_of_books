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
}

type ReportEmailData struct {
	ReporterName string
	TargetType   string
	Reason       string
	LinkURL      string
}

type ReportResolvedEmailData struct {
	ResolverName string
	TargetType   string
	Comment      string
	LinkURL      string
}

func NotifEmail(actorName, action, title, linkURL string) (subject string, body string) {
	subject = fmt.Sprintf("%s %s", actorName, action)

	data := NotifEmailData{
		Subject:   subject,
		ActorName: actorName,
		Action:    action,
		Title:     title,
		LinkURL:   linkURL,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "notification.tmpl", data); err != nil {
		return subject, fmt.Sprintf("<p>%s</p>", subject)
	}

	return subject, buf.String()
}

func ReportEmail(reporterName, targetType, reason, linkURL string) (subject string, body string) {
	subject = fmt.Sprintf("New report from %s", reporterName)

	data := ReportEmailData{
		ReporterName: reporterName,
		TargetType:   targetType,
		Reason:       reason,
		LinkURL:      linkURL,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "report.tmpl", data); err != nil {
		return subject, fmt.Sprintf("<p>%s</p>", subject)
	}

	return subject, buf.String()
}

func ReportResolvedEmail(resolverName, targetType, comment, linkURL string) (subject string, body string) {
	subject = "Your report has been resolved"

	data := ReportResolvedEmailData{
		ResolverName: resolverName,
		TargetType:   targetType,
		Comment:      comment,
		LinkURL:      linkURL,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "report_resolved.tmpl", data); err != nil {
		return subject, fmt.Sprintf("<p>%s</p>", subject)
	}

	return subject, buf.String()
}

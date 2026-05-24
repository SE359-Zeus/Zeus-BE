package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"log"
)

type EmailTemplateRequest struct {
	To       string
	Subject  string
	Template string
	Data     any
}

type EmailService interface {
	SendTemplate(ctx context.Context, req EmailTemplateRequest) error
}

type SMTPEmailService struct {
	host      string
	port      string
	user      string
	password  string
	from      string
	templates map[string]*template.Template
}

type CreateAccountEmailData struct {
	To        string
	FullName  string
	Username  string
	Password  string
	Role      string
	CreatedAt time.Time
}

func NewSMTPEmailService(host, port, user, password, fromAddress, templateDir string) (*SMTPEmailService, error) {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	user = strings.TrimSpace(user)
	password = strings.TrimSpace(password)
	fromAddress = strings.TrimSpace(fromAddress)
	templateDir = strings.TrimSpace(templateDir)

	if host == "" {
		return nil, errors.New("smtp host is required")
	}
	if port == "" {
		return nil, errors.New("smtp port is required")
	}
	if user == "" {
		return nil, errors.New("smtp user is required")
	}
	if password == "" {
		return nil, errors.New("smtp password is required")
	}
	if fromAddress == "" {
		fromAddress = user
	}
	if templateDir == "" {
		templateDir = "templates"
	}

	templates, err := loadEmailTemplates(templateDir)
	if err != nil {
		return nil, err
	}

	return &SMTPEmailService{
		host:      host,
		port:      port,
		user:      user,
		password:  password,
		from:      fromAddress,
		templates: templates,
	}, nil
}

func loadEmailTemplates(templateDir string) (map[string]*template.Template, error) {
	info, err := os.Stat(templateDir)
	if err != nil {
		return nil, fmt.Errorf("stat email template path %s: %w", templateDir, err)
	}

	templateFiles := make([]string, 0)
	if info.IsDir() {
		entries, err := os.ReadDir(templateDir)
		if err != nil {
			return nil, fmt.Errorf("read email template dir: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
				templateFiles = append(templateFiles, filepath.Join(templateDir, entry.Name()))
			}
		}
	} else {
		if !strings.HasSuffix(strings.ToLower(templateDir), ".html") {
			return nil, fmt.Errorf("email template path %s must be a directory or .html file", templateDir)
		}
		templateFiles = append(templateFiles, templateDir)
	}

	if len(templateFiles) == 0 {
		return nil, fmt.Errorf("no html templates found in %s", templateDir)
	}

	sort.Strings(templateFiles)
	templates := make(map[string]*template.Template, len(templateFiles))
	for _, path := range templateFiles {
		name := filepath.Base(path)
		tpl, err := template.ParseFiles(path)
		if err != nil {
			return nil, fmt.Errorf("parse email template %s: %w", name, err)
		}
		templates[name] = tpl
	}

	return templates, nil
}

func (s *SMTPEmailService) SendTemplate(ctx context.Context, req EmailTemplateRequest) error {
	if s == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.To == "" {
		return errors.New("recipient address is required")
	}
	if req.Subject == "" {
		return errors.New("email subject is required")
	}
	if req.Template == "" {
		return errors.New("email template is required")
	}

	tpl, ok := s.templates[req.Template]
	if !ok {
		return fmt.Errorf("email template %q not loaded", req.Template)
	}

	var body bytes.Buffer
	if err := tpl.Execute(&body, req.Data); err != nil {
		return fmt.Errorf("render email template %s: %w", req.Template, err)
	}

	mime := "MIME-Version: 1.0\r\n" +
		fmt.Sprintf("From: %s\r\n", s.from) +
		fmt.Sprintf("To: %s\r\n", req.To) +
		fmt.Sprintf("Subject: %s\r\n", req.Subject) +
		"Content-Type: text/html; charset=UTF-8\r\n\r\n" +
		body.String()

	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	log.Printf("Sending SMTP email template=%s to=%s subject=%s", req.Template, req.To, req.Subject)
	if err := smtp.SendMail(addr, auth, s.from, []string{req.To}, []byte(mime)); err != nil {
		log.Printf("Error sending SMTP email: %v", err)
		return fmt.Errorf("smtp send: %w", err)
	}

	log.Printf("SMTP Email sent successfully to %s", req.To)
	return nil
}

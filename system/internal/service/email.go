package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"log"

	"github.com/resend/resend-go/v3"
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

type ResendEmailService struct {
	from      string
	templates map[string]*template.Template
	client    *resend.Client
}

type CreateAccountEmailData struct {
	To        string
	FullName  string
	Username  string
	Password  string
	Role      string
	CreatedAt time.Time
}

func NewResendEmailService(apiKey, fromAddress, fromName, templateDir string) (*ResendEmailService, error) {
	apiKey = strings.TrimSpace(apiKey)
	fromAddress = strings.TrimSpace(fromAddress)
	fromName = strings.TrimSpace(fromName)
	templateDir = strings.TrimSpace(templateDir)

	if apiKey == "" {
		return nil, errors.New("resend api key is required")
	}
	if fromAddress == "" {
		return nil, errors.New("resend from address is required")
	}
	if templateDir == "" {
		templateDir = "templates"
	}

	templates, err := loadEmailTemplates(templateDir)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpClient.Transport = &debugTransport{base: http.DefaultTransport}
	client := resend.NewCustomClient(httpClient, apiKey)

	return &ResendEmailService{
		from:      formatResendFrom(fromAddress, fromName),
		templates: templates,
		client:    client,
	}, nil
}

func formatResendFrom(address, name string) string {
	address = strings.TrimSpace(address)
	name = strings.TrimSpace(name)
	if address == "" {
		return ""
	}
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

// debugTransport logs outbound requests and responses for troubleshooting.
// It redacts the Authorization header when logging.
type debugTransport struct {
	base http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}

	var reqBody []byte
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		reqBody = b
		req.Body = io.NopCloser(bytes.NewReader(b))
	}

	// prepare redacted headers for logging
	loggedHeaders := make(map[string][]string)
	for k, v := range req.Header {
		if strings.ToLower(k) == "authorization" {
			loggedHeaders[k] = []string{"<redacted>"}
		} else {
			loggedHeaders[k] = v
		}
	}

	log.Printf("Resend HTTP Request: %s %s headers=%v body_len=%d", req.Method, req.URL.String(), loggedHeaders, len(reqBody))

	resp, err := base.RoundTrip(req)
	if err != nil {
		log.Printf("Resend HTTP roundtrip error: %v", err)
		return resp, err
	}

	if resp.Body != nil {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(b))
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyStr := strings.TrimSpace(string(b))
			if len(bodyStr) > 0 {
				log.Printf("Resend HTTP Response: status=%s body=%s", resp.Status, bodyStr)
			} else {
				log.Printf("Resend HTTP Response: status=%s (empty body)", resp.Status)
			}
		} else {
			log.Printf("Resend HTTP Response: status=%s body_len=%d", resp.Status, len(b))
		}
	} else {
		log.Printf("Resend HTTP Response: status=%s (no body)", resp.Status)
	}

	return resp, err
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

func (s *ResendEmailService) SendTemplate(ctx context.Context, req EmailTemplateRequest) error {
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

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{req.To},
		Subject: req.Subject,
		Html:    body.String(),
	}

	log.Printf("Sending email template=%s to=%s subject=%s", req.Template, req.To, req.Subject)
	sent, err := s.client.Emails.Send(params)
	if err != nil {
		log.Printf("Error sending email template=%s to=%s: %v", req.Template, req.To, err)
		log.Printf("Resend error type %T", err)
		return fmt.Errorf("send email: %w", err)
	}

	if sent.Id != "" {
		log.Printf("Email sent template=%s to=%s id=%s", req.Template, req.To, sent.Id)
	} else {
		log.Printf("Email sent template=%s to=%s (no id returned)", req.Template, req.To)
	}

	return nil
}

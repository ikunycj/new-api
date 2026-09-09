package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"slices"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
)

func sanitizeEmailHeader(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}

func splitEmailRecipients(receiver string) (envelope []string, header []string, err error) {
	for _, raw := range strings.Split(receiver, ";") {
		raw = strings.TrimSpace(sanitizeEmailHeader(raw))
		if raw == "" {
			continue
		}
		parsed, parseErr := mail.ParseAddress(raw)
		if parseErr != nil {
			return nil, nil, fmt.Errorf("invalid SMTP recipient")
		}
		envelope = append(envelope, parsed.Address)
		header = append(header, parsed.String())
	}
	return envelope, header, nil
}

func smtpFromAddress() (string, error) {
	from := strings.TrimSpace(sanitizeEmailHeader(SMTPFrom))
	parsed, err := mail.ParseAddress(from)
	if err != nil || !strings.Contains(parsed.Address, "@") {
		return "", fmt.Errorf("invalid SMTP account")
	}
	return parsed.Address, nil
}

func normalizePlainEmailText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func isEmailBlockElement(name string) bool {
	switch name {
	case "article", "blockquote", "div", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "li", "p", "pre", "section", "table", "tr":
		return true
	default:
		return false
	}
}

func htmlEmailToPlainText(content string) string {
	document, err := xhtml.Parse(strings.NewReader(content))
	if err != nil {
		return normalizePlainEmailText(content)
	}

	var plainText strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			plainText.WriteString(node.Data)
			return
		}
		if node.Type != xhtml.ElementNode {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			return
		}

		switch strings.ToLower(node.Data) {
		case "head", "script", "style":
			return
		case "br":
			plainText.WriteByte('\n')
			return
		case "a":
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			for _, attribute := range node.Attr {
				if strings.EqualFold(attribute.Key, "href") && strings.TrimSpace(attribute.Val) != "" {
					plainText.WriteString(" [")
					plainText.WriteString(attribute.Val)
					plainText.WriteByte(']')
					break
				}
			}
			return
		}

		block := isEmailBlockElement(strings.ToLower(node.Data))
		if block {
			plainText.WriteByte('\n')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if block {
			plainText.WriteByte('\n')
		}
	}
	walk(document)
	return normalizePlainEmailText(plainText.String())
}

func writeEmailMIMEPart(writer *multipart.Writer, contentType string, content string) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType+"; charset=UTF-8")
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	encoder := quotedprintable.NewWriter(part)
	if _, err = encoder.Write([]byte(content)); err != nil {
		return err
	}
	return encoder.Close()
}

func buildEmailMessage(receiver string, subject string, content string) ([]byte, []string, error) {
	envelopeRecipients, headerRecipients, err := splitEmailRecipients(receiver)
	if err != nil {
		return nil, nil, err
	}
	if len(envelopeRecipients) == 0 {
		return nil, nil, fmt.Errorf("no SMTP recipients")
	}
	fromAddress, err := smtpFromAddress()
	if err != nil {
		return nil, nil, err
	}

	var multipartBody bytes.Buffer
	writer := multipart.NewWriter(&multipartBody)
	if err := writeEmailMIMEPart(writer, "text/plain", htmlEmailToPlainText(content)); err != nil {
		return nil, nil, err
	}
	if err := writeEmailMIMEPart(writer, "text/html", content); err != nil {
		return nil, nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, nil, err
	}

	from := (&mail.Address{
		Name:    sanitizeEmailHeader(SystemName),
		Address: fromAddress,
	}).String()
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(sanitizeEmailHeader(subject))))

	var message bytes.Buffer
	fmt.Fprintf(&message, "To: %s\r\n", strings.Join(headerRecipients, ", "))
	fmt.Fprintf(&message, "From: %s\r\n", from)
	fmt.Fprintf(&message, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&message, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	id, err := generateMessageID()
	if err != nil {
		return nil, nil, err
	}
	fmt.Fprintf(&message, "Message-ID: %s\r\n", id)
	message.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", writer.Boundary())
	message.Write(multipartBody.Bytes())
	return message.Bytes(), envelopeRecipients, nil
}

func generateMessageID() (string, error) {
	fromAddress, err := smtpFromAddress()
	if err != nil {
		return "", err
	}
	domain := fromAddress[strings.LastIndex(fromAddress, "@")+1:]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func shouldUseSMTPLoginAuth() bool {
	if SMTPForceAuthLogin {
		return true
	}
	return isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer)
}

func getSMTPAuth() smtp.Auth {
	return AutoSMTPAuth(SMTPAccount, SMTPToken)
}

func shouldAuthenticateSMTP() bool {
	return SMTPAccount != "" && SMTPToken != ""
}

func smtpTLSConfig() *tls.Config {
	return &tls.Config{
		ServerName:         SMTPServer,
		InsecureSkipVerify: SMTPInsecureSkipVerify, // #nosec G402 -- admin-controlled SMTP compatibility option.
	}
}

func newSMTPClient(addr string) (*smtp.Client, error) {
	if SMTPSSLEnabled || (SMTPPort == 465 && !SMTPStartTLSEnabled) {
		conn, err := tls.Dial("tcp", addr, smtpTLSConfig())
		if err != nil {
			return nil, err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, err
	}

	if SMTPStartTLSEnabled {
		startTLSSupported, _ := client.Extension("STARTTLS")
		if !startTLSSupported {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(smtpTLSConfig()); err != nil {
			_ = client.Close()
			return nil, err
		}
	}

	return client, nil
}

func SendEmail(subject string, receiver string, content string) error {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return fmt.Errorf("SMTP 服务器未配置")
	}
	mailMessage, recipients, err := buildEmailMessage(receiver, subject, content)
	if err != nil {
		return err
	}
	auth := getSMTPAuth()
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	client, err := newSMTPClient(addr)
	if err != nil {
		return err
	}
	defer client.Close()
	if shouldAuthenticateSMTP() {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	fromAddress, err := smtpFromAddress()
	if err != nil {
		return err
	}
	if err = client.Mail(fromAddress); err != nil {
		return err
	}
	for _, recipient := range recipients {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(mailMessage)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	err = client.Quit()
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}

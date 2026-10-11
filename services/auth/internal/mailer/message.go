package mailer

import (
	"bytes"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"time"
)

type message struct {
	From     string
	FromAddr string
	To       string
	Subject  string
	Text     string
	HTML     string
	Date     time.Time
}

func (m message) bytes() ([]byte, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range []struct{ typ, text string }{
		{"text/plain", m.Text},
		{"text/html", m.HTML},
	} {
		if err := writePart(mw, p.typ, p.text); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	id, err := messageID(m.FromAddr)
	if err != nil {
		return nil, err
	}

	var msg bytes.Buffer
	for _, h := range [][2]string{
		{"From", m.From},
		{"To", m.To},
		{"Subject", mime.QEncoding.Encode("utf-8", m.Subject)},
		{"Date", m.Date.Format(time.RFC1123Z)},
		{"Message-ID", id},
		{"MIME-Version", "1.0"},
		{"Auto-Submitted", "auto-generated"},
		{"X-Auto-Response-Suppress", "All"},
		{"Content-Type", "multipart/alternative; boundary=" + mw.Boundary()},
	} {
		fmt.Fprintf(&msg, "%s: %s\r\n", h[0], h[1])
	}
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())

	return msg.Bytes(), nil
}

func writePart(mw *multipart.Writer, contentType, text string) error {
	w, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {contentType + "; charset=UTF-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("create %s part: %w", contentType, err)
	}
	qw := quotedprintable.NewWriter(w)
	if _, err = qw.Write([]byte(text)); err != nil {
		return fmt.Errorf("write %s part: %w", contentType, err)
	}
	return qw.Close()
}

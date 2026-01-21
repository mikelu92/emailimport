package target

import (
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var exp *regexp.Regexp

func init() {
	exp = regexp.MustCompile("(?s)Hello .*,.*A transaction of (?P<amt>\\$\\d+\\.\\d+)[\\s\\p{Zs}]+at[\\s\\p{Zs}]+(?P<payee>.+?)[\\s\\p{Zs}]+has been approved on your.*Target Circle.*Card")
}

type ProviderTarget struct {
	Account string
}

func getBodyText(msg *gmail.Message) string {
	if msg.Payload == nil {
		return ""
	}

	var extractText func(part *gmail.MessagePart) string
	extractText = func(part *gmail.MessagePart) string {
		if part == nil {
			return ""
		}
		if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err == nil {
				return string(data)
			}
		}
		if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
			data, err := base64.URLEncoding.DecodeString(part.Body.Data)
			if err == nil {
				return string(data)
			}
		}
		for _, child := range part.Parts {
			if text := extractText(child); text != "" {
				return text
			}
		}
		return ""
	}

	return extractText(msg.Payload)
}

func (p *ProviderTarget) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}
	body := getBodyText(msg)
	match := exp.FindStringSubmatch(body)
	if len(match) == 0 {
		return nil, nil
	}
	result := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	t.Amount = result["amt"]
	t.Payee = result["payee"]
	var dateString string
	for _, h := range msg.Payload.Headers {
		if h.Name == "Received" {
			dateString = strings.TrimSpace(strings.Split(h.Value, ";")[1])
			break
		}
	}
	d, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", dateString)
	if err != nil {
		return nil, err
	}
	t.Date = d

	return &t, nil
}

func (p *ProviderTarget) GetAccount() string {
	return p.Account
}

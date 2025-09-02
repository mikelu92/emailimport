package discover

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var (
	reDate     *regexp.Regexp
	reMerchant *regexp.Regexp
	reAmount   *regexp.Regexp
)

func init() {
	// Multiline, order-agnostic label matchers
	reDate = regexp.MustCompile(`(?m)^(?:Transaction Date|Date):\s*(?P<date>.+)$`)
	reMerchant = regexp.MustCompile(`(?m)^Merchant:\s*(?P<payee>.+)$`)
	reAmount = regexp.MustCompile(`(?m)^Amount:\s*(?P<amt>[\$\d,]+\.\d{2})$`)
}

type ProviderDiscover struct {
	Account string
}

func (p *ProviderDiscover) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}

	// ID (optional)
	for _, h := range msg.Payload.Headers {
		if h.Name == "X-MSG-ID" {
			t.ID = h.Value
			break
		}
	}

	// Determine plain text body: prefer text/plain part, fallback to top-level body, then snippet
	var bodyText string

	if body, err := findPlainText(msg.Payload); err == nil && body != nil && body.Data != "" {
		if decoded, err := decodeBase64(body.Data); err == nil {
			bodyText = string(decoded)
		} else {
			return nil, err
		}
	} else if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		if decoded, err := decodeBase64(msg.Payload.Body.Data); err == nil {
			bodyText = string(decoded)
		} else {
			return nil, err
		}
	} else {
		// As a last resort, use the snippet (may or may not contain labeled lines)
		bodyText = msg.Snippet
	}

	// Extract fields using independent label regexes
	fields := map[string]string{}

	if m := reDate.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reDate.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}
	if m := reMerchant.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reMerchant.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}
	if m := reAmount.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reAmount.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}

	// Require all three fields to consider this a Discover transaction
	dateStr, hasDate := fields["date"]
	payee, hasPayee := fields["payee"]
	amt, hasAmt := fields["amt"]
	if !hasDate || !hasPayee || !hasAmt {
		return nil, nil
	}

	// Parse date ("January 2, 2006" handles single- and double-digit days; fallback to "January 02, 2006")
	d, err := time.Parse("January 2, 2006", dateStr)
	if err != nil {
		d, err = time.Parse("January 02, 2006", dateStr)
		if err != nil {
			return nil, err
		}
	}

	t.Payee = payee
	t.Amount = amt
	t.Date = d

	return &t, nil
}

func (p *ProviderDiscover) GetAccount() string {
	return p.Account
}

// --- helpers ---

var ErrPartNotFound = errors.New("part not found")

func findPlainText(msg *gmail.MessagePart) (*gmail.MessagePartBody, error) {
	for _, header := range msg.Headers {
		if header.Name != "Content-Type" {
			continue
		}
		if strings.Contains(header.Value, "multipart") {
			for _, part := range msg.Parts {
				body, err := findPlainText(part)
				if errors.Is(err, ErrPartNotFound) {
					continue
				}
				return body, nil
			}
		} else if strings.Contains(header.Value, "text/plain") {
			return msg.Body, nil
		}
	}
	return nil, ErrPartNotFound
}

func decodeBase64(s string) ([]byte, error) {
	// Try padded base64url first, then raw (unpadded)
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

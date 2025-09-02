package discover

import (
	"encoding/base64"
	"errors"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	htmlparser "golang.org/x/net/html"
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

	log.Printf("discover.GetTransaction: account=%q msgID=%q", p.Account, t.ID)

	// Determine plain text body: prefer text/plain part, fallback to top-level body, then snippet
	var bodyText string
	var bodySource string

	if body, err := findPlainText(msg.Payload); err == nil && body != nil && body.Data != "" {
		if decoded, err := decodeBase64(body.Data); err == nil {
			bodyText = string(decoded)
			bodySource = "text/plain part"
		} else {
			log.Printf("discover.GetTransaction: failed to decode text/plain part for account=%q msgID=%q: %v", p.Account, t.ID, err)
			return nil, err
		}
	} else if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		if decoded, err := decodeBase64(msg.Payload.Body.Data); err == nil {
			bodyText = string(decoded)
			bodySource = "top-level body"
		} else {
			log.Printf("discover.GetTransaction: failed to decode top-level body for account=%q msgID=%q: %v", p.Account, t.ID, err)
			return nil, err
		}
	} else {
		// As a last resort, use the snippet (may or may not contain labeled lines)
		bodyText = msg.Snippet
		bodySource = "snippet"
	}

	log.Printf("discover.GetTransaction: using body source %s for account=%q msgID=%q", bodySource, p.Account, t.ID)

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

	// If fields incomplete, try HTML fallback
	if !hasDate || !hasPayee || !hasAmt {
		if body, err := findHTML(msg.Payload); err == nil && body != nil && body.Data != "" {
			if decoded, err := decodeBase64(body.Data); err == nil {
				htmlText := htmlToText(string(decoded))
				// Re-run regexes on htmlText
				if m := reDate.FindStringSubmatch(htmlText); len(m) > 0 {
					for i, name := range reDate.SubexpNames() {
						if i != 0 && name != "" {
							fields[name] = strings.TrimSpace(m[i])
						}
					}
				}
				if m := reMerchant.FindStringSubmatch(htmlText); len(m) > 0 {
					for i, name := range reMerchant.SubexpNames() {
						if i != 0 && name != "" {
							fields[name] = strings.TrimSpace(m[i])
						}
					}
				}
				if m := reAmount.FindStringSubmatch(htmlText); len(m) > 0 {
					for i, name := range reAmount.SubexpNames() {
						if i != 0 && name != "" {
							fields[name] = strings.TrimSpace(m[i])
						}
					}
				}
				// Update has variables
				dateStr, hasDate = fields["date"]
				payee, hasPayee = fields["payee"]
				amt, hasAmt = fields["amt"]
				if hasDate && hasPayee && hasAmt {
					bodySource = "text/html part"
				}
			}
		}
	}

	if !hasDate || !hasPayee || !hasAmt {
		preview := bodyText
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("discover.GetTransaction: unrecognized transaction format for account=%q msgID=%q fields=%v bodyPreview=%q", p.Account, t.ID, fields, preview)
		return nil, nil
	}

	// Parse date ("January 2, 2006" handles single- and double-digit days; fallback to "January 02, 2006")
	d, err := time.Parse("January 2, 2006", dateStr)
	if err != nil {
		d, err = time.Parse("January 02, 2006", dateStr)
		if err != nil {
			log.Printf("discover.GetTransaction: failed to parse date %q for account=%q msgID=%q: %v", dateStr, p.Account, t.ID, err)
			return nil, err
		}
	}

	t.Payee = payee
	t.Amount = amt
	t.Date = d

	log.Printf("discover.GetTransaction: parsed transaction for account=%q msgID=%q date=%s payee=%q amt=%q bodySource=%s", p.Account, t.ID, d.Format("2006-01-02"), payee, amt, bodySource)

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

func findHTML(msg *gmail.MessagePart) (*gmail.MessagePartBody, error) {
	for _, header := range msg.Headers {
		if header.Name != "Content-Type" {
			continue
		}
		if strings.Contains(header.Value, "multipart") {
			for _, part := range msg.Parts {
				body, err := findHTML(part)
				if errors.Is(err, ErrPartNotFound) {
					continue
				}
				return body, nil
			}
		} else if strings.Contains(header.Value, "text/html") {
			return msg.Body, nil
		}
	}
	return nil, ErrPartNotFound
}

func htmlToText(s string) string {
	var result strings.Builder
	tokenizer := htmlparser.NewTokenizer(strings.NewReader(s))
	for {
		tt := tokenizer.Next()
		switch tt {
		case htmlparser.ErrorToken:
			text := result.String()
			text = strings.ReplaceAll(text, "\r\n", "\n")
			// Collapse multiple spaces and tabs, but keep newlines
			text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
			return html.UnescapeString(text)
		case htmlparser.TextToken:
			text := strings.TrimSpace(string(tokenizer.Text()))
			if text != "" {
				result.WriteString(text)
			}
		case htmlparser.StartTagToken, htmlparser.SelfClosingTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "br", "p", "div", "li", "tr", "td", "h1", "h2", "h3", "h4", "h5", "h6":
				result.WriteString("\n")
			}
		}
	}
}

func decodeBase64(s string) ([]byte, error) {
	// Try padded base64url first, then raw (unpadded)
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

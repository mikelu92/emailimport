package chase

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"golang.org/x/net/html"
	"google.golang.org/api/gmail/v1"
)

var (
	subject *regexp.Regexp
	last4   *regexp.Regexp
)

func init() {
	// Accept both “Your …” and “You made a …”, allow thousands separators, and “with” or “at”
	subject, _ = regexp.Compile(`^(?:Your|You made a) (?P<amt>\$\d{1,3}(?:,\d{3})*\.\d{2}) transaction(?: with| at) (?P<payee>.+)$`)
	last4, _ = regexp.Compile(`\d{4}`)
}

type ProviderChase struct {
	Accounts map[int]string
}

func (p *ProviderChase) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{}
	var match []string
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			match = subject.FindStringSubmatch(header.Value)
			break
		}
	}
	if len(match) == 0 {
		return nil, nil
	}

	result := make(map[string]string)
	for _, header := range msg.Payload.Headers {
		if header.Name == "Date" {
			result["date"] = header.Value
			break
		}
	}

	for i, name := range subject.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	d, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", result["date"])
	if err != nil {
		return nil, err
	}

	t.Payee = result["payee"]
	t.Amount = result["amt"]
	t.Date = d

	// Now get account
	var bodyData string
	if part, err := findHTML(msg.Payload); err == nil && part != nil && part.Data != "" {
		bodyData = part.Data
	} else if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		// fallback to top-level body
		bodyData = msg.Payload.Body.Data
	} else {
		// no body we can parse
		return nil, nil
	}

	body, err := decodeBase64(bodyData)
	if err != nil {
		return nil, err
	}

	ht := html.NewTokenizer(strings.NewReader(string(body)))

	var actFound bool
loop:
	for {
		tt := ht.Next()
		var token html.Token
		switch tt {
		case html.ErrorToken:
			break loop
		case html.StartTagToken:
			token = ht.Token()
			if token.Data == "tr" {
				for {
					tt = ht.Next()
					token = ht.Token()
					if tt == html.StartTagToken {
						if token.Data == "td" {
							tt = ht.Next()
							break
						} else {
							continue loop
						}
					}
					if tt == html.ErrorToken {
						break loop
					}
				}
				token = ht.Token()
				if strings.TrimSpace(token.Data) == "Account" {
					actFound = true
					break
				}
			}
			continue loop
		default:
			continue loop
		}
		ht.Next()      // should be /td
		tt = ht.Next() // should be td or text token
		if tt == html.TextToken {
			ht.Next() // in case it's a text token
		}
		ht.Next() // should be the actual account
		token = ht.Token()
		digits := last4.Find([]byte(token.Data))
		i, _ := strconv.Atoi(string(digits))

		act, ok := p.Accounts[i]
		if !ok {
			return nil, nil
		}
		t.Account = act
		break
	}
	if !actFound {
		return nil, nil
	}
	return &t, nil
}

func (p *ProviderChase) GetAccount() string {
	return "chase"

}

// decodeBase64 tries padded base64url first, then raw (unpadded) base64url.
func decodeBase64(s string) ([]byte, error) {
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

var ErrPartNotFound = errors.New("part not found")

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

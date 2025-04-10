package capitalone

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var data *regexp.Regexp

func init() {
	data, _ = regexp.Compile("(?m)on (?P<date>[A-Z][a-z]+ \\d{1,2}, \\d{4}), at (?P<payee>.+), a pending authorization or purchase in the amount of (?P<amt>\\$\\d{0,}.\\d{2}) was placed")
}

type ProviderCapitalOne struct {
	Account string
}

func (p *ProviderCapitalOne) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}
	var match bool
	for _, header := range msg.Payload.Headers {
		if header.Name == "Subject" {
			if strings.Contains(header.Value, "transaction was charged to your account") {
				match = true
				break
			}
		}
	}
	if !match {
		return nil, nil
	}
	result := make(map[string]string)

	body, err := findPlainText(msg.Payload)
	if err != nil {
		return nil, err
	}
	bodyMsg, err := base64.URLEncoding.DecodeString(body.Data)
	if err != nil {
		return nil, fmt.Errorf("unable to decode email message: %w", err)
	}
	transactionParts := data.FindStringSubmatch(string(bodyMsg))

	for i, name := range data.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = transactionParts[i]
		}
	}
	d, err := time.Parse("January 2, 2006", result["date"])
	if err != nil {
		return nil, err
	}
	t.Payee = result["payee"]
	t.Amount = result["amt"]
	t.Date = d
	return &t, nil
}

func (p *ProviderCapitalOne) GetAccount() string {
	return p.Account
}

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

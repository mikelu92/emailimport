package target

import (
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var exp *regexp.Regexp

func init() {
	exp = regexp.MustCompile("Hello .*, A transaction of (?P<amt>\\$\\d+\\.\\d+) at (?P<payee>.*) was Approved with your Target RedCard")

}

type ProviderTarget struct {
	Account string
}

func (p *ProviderTarget) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}
	match := exp.FindStringSubmatch(msg.Snippet)
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

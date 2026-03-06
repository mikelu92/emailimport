package paypal

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var (
	expSent  *regexp.Regexp
	expSent2 *regexp.Regexp
	expRec   *regexp.Regexp
)

func init() {
	expSent, _ = regexp.Compile(`You (?:paid|sent|contributed) (?P<amt>\$\d+\.\d+) USD to (?P<payee>.+?)(?: Transaction date| Transaction Details| Contribution details| Transaction ID|$)`)
	expSent2, _ = regexp.Compile("Transaction ID (?P<id>\\S+)")
	expRec, _ = regexp.Compile("Hello, \\S+\\s\\S+ (?P<payee>.+?) sent you (?P<amt>\\$\\d+\\.\\d+) USD.*?Note from \\S+\\s\\S+ (?P<note>.+?) Transaction date (?P<date>.+?)(?: Transaction ID(?P<id>\\S*))?")
}

type ProviderPaypal struct {
	Account string
}

func (p *ProviderPaypal) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}
	exp := expSent
	match := exp.FindStringSubmatch(msg.Snippet)
	if len(match) == 0 {
		exp = expRec
		match = exp.FindStringSubmatch(msg.Snippet)
		if len(match) == 0 {
			fmt.Println("No match found for snippet:", msg.Snippet)
			return nil, nil
		}
		t.IsReceive = true
	} else {
	}
	result := make(map[string]string)
	for i, name := range exp.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = strings.TrimSpace(match[i])
		}
	}

	if !t.IsReceive {
		expSentNote, _ := regexp.Compile("YOUR NOTE TO " + result["payee"] + " (.*) Transaction Details")
		note := expSentNote.FindStringSubmatch(msg.Snippet)
		if len(note) > 0 {
			result["note"] = html.UnescapeString(note[1])
		}

		match = expSent2.FindStringSubmatch(msg.Snippet)
		if len(match) != 0 {
			for i, name := range expSent2.SubexpNames() {
				if i != 0 && name != "" {
					result[name] = match[i]
				}
			}
		}
	}

	d, err := time.Parse("January 2, 2006", result["date"])
	if err != nil {
		var dateString string
		for _, h := range msg.Payload.Headers {
			if h.Name == "Date" {
				dateString = h.Value
				break
			}
		}
		if dateString == "" {
			return nil, errors.New("Unable to find date header")
		}
		d, err = time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", dateString)
		if err != nil {
			return nil, err
		}
	}

	t.ID = result["id"]
	t.Payee = result["payee"]
	t.Amount = result["amt"]
	t.Note = result["note"]
	t.Date = d
	return &t, nil
}

func (p *ProviderPaypal) GetAccount() string {
	return p.Account
}

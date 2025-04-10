package discover

import (
	"encoding/base64"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var expTr *regexp.Regexp

func init() {
	expTr, _ = regexp.Compile("Transaction Date:+ (?P<date>.*)\\s[\\r\\n]+Merchant: (?P<payee>.*)\\s[\\r\\n]+Amount: (?P<amt>.*)")
}

type ProviderDiscover struct {
	Account string
}

func (p *ProviderDiscover) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: "liabilities:anna:discover"}
	for _, h := range msg.Payload.Headers {
		if h.Name == "X-MSG-ID" {
			t.ID = h.Value
			break
		}
	}

	s, err := base64.URLEncoding.DecodeString(msg.Payload.Parts[0].Body.Data)
	if err != nil {
		return nil, err
	}

	b := expTr.FindStringSubmatch(string(s))
	tMap := make(map[string]string)
	for i, name := range expTr.SubexpNames() {
		if i != 0 && name != "" {
			tMap[name] = b[i]
		}
	}
	t.Payee = tMap["payee"]
	t.Amount = tMap["amt"]
	d, err := time.Parse("January 02, 2006", strings.TrimSpace(tMap["date"]))
	if err != nil {
		return nil, err
	}
	t.Date = d

	return &t, nil
}

func (p *ProviderDiscover) GetAccount() string {
	return p.Account
}

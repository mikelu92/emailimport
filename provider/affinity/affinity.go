package affinity

import (
	"regexp"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var (
	exp *regexp.Regexp
)

func init() {
	exp, _ = regexp.Compile("Service Charge for (?P<amt>\\$\\d+\\.\\d+) on (?P<date>.*) at (?P<payee>.*) on card ending in")
}

type ProviderAffinity struct {
	Account string
}

func (p *ProviderAffinity) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{}
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
	d, err := time.Parse("01/02 15:04 MST", result["date"])
	if err != nil {
		return nil, err
	}
	d = d.AddDate(time.Now().Year(), 0, 0)

	t.Payee = result["payee"]
	t.Amount = result["amt"]
	t.Date = d
	return &t, nil
}

func (p *ProviderAffinity) GetAccount() string {
	return p.Account
}

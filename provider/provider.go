package provider

import (
	"github.com/mikelu92/emailimport/pkg/ledger"
	"github.com/mikelu92/emailimport/provider/affinity"
	"github.com/mikelu92/emailimport/provider/capitalone"
	"github.com/mikelu92/emailimport/provider/chase"
	"github.com/mikelu92/emailimport/provider/discover"
	"github.com/mikelu92/emailimport/provider/paypal"
	"github.com/mikelu92/emailimport/provider/target"
	"google.golang.org/api/gmail/v1"
)

type ProviderConfig struct {
	Account  string
	Accounts map[int]string
	Label    string
	Type     string
}

type Provider interface {
	GetTransaction(msg *gmail.Message) (*ledger.Transaction, error)
	GetAccount() string
}

func Get(conf ProviderConfig) Provider {
	switch conf.Type {
	case "paypal":
		return &paypal.ProviderPaypal{Account: conf.Account}
	case "discover":
		return &discover.ProviderDiscover{Account: conf.Account}
	case "target":
		return &target.ProviderTarget{Account: conf.Account}
	case "affinity":
		return &affinity.ProviderAffinity{Account: conf.Account}
	case "chase":
		return &chase.ProviderChase{Accounts: conf.Accounts}
	case "capitalone":
		return &capitalone.ProviderCapitalOne{Account: conf.Account}

	}
	return nil
}

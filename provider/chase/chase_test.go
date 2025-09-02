package chase

import (
	"encoding/base64"
	"testing"
	"time"

	"google.golang.org/api/gmail/v1"
)

func TestGetTransaction_SampleEmail(t *testing.T) {
	htmlBody := `<html><body><table><tr><td>Account</td><td>Chase Freedom Visa (...8719)</td></tr></table></body></html>`
	encoded := base64.URLEncoding.EncodeToString([]byte(htmlBody))

	msg := &gmail.Message{
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Subject", Value: "You made a $4.04 transaction with PAYPAL *NY TIMES NYT"},
				{Name: "Date", Value: "Sun, 17 Aug 2025 09:50:14 +0000 (UTC)"},
				{Name: "Content-Type", Value: "text/html; charset=UTF-8"},
			},
			Body: &gmail.MessagePartBody{Data: encoded},
		},
	}

	p := &ProviderChase{Accounts: map[int]string{8719: "chase:freedom"}}
	tx, err := p.GetTransaction(msg)
	if err != nil {
		t.Fatalf("GetTransaction returned error: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected transaction, got nil")
	}
	if tx.Amount != "$4.04" {
		t.Fatalf("expected amount $4.04, got %q", tx.Amount)
	}
	if tx.Payee != "PAYPAL *NY TIMES NYT" {
		t.Fatalf("expected payee %q, got %q", "PAYPAL *NY TIMES NYT", tx.Payee)
	}
	expectedDate, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", "Sun, 17 Aug 2025 09:50:14 +0000 (UTC)")
	if err != nil {
		t.Fatalf("failed to parse expected date: %v", err)
	}
	if !tx.Date.Equal(expectedDate) {
		t.Fatalf("expected date %v, got %v", expectedDate, tx.Date)
	}
	if tx.Account != "chase:freedom" {
		t.Fatalf("expected account %q, got %q", "chase:freedom", tx.Account)
	}
}

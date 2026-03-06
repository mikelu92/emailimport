package paypal

import (
	"testing"

	"google.golang.org/api/gmail/v1"
)

func TestGetTransaction(t *testing.T) {
	testCases := []struct {
		name     string
		snippet  string
		wantErr  bool
		expected map[string]string
	}{
		{
			name:    "received money",
			snippet: "Michael Lu, here are the details. Hello, Michael Lu Kenneth Kao sent you $20.00 USD Amount $20.00 USD Note from Kenneth Kao wolves vs warriors tix Transaction date January 27, 2026 Transaction ID",
			wantErr: false,
			expected: map[string]string{
				"amount": "$20.00",
				"payee":  "Kenneth Kao",
				"note":   "wolves vs warriors tix",
			},
		},
		{
			name:    "sent money",
			snippet: "Payment details are inside. Hello, Michael Lu You paid $4.04 USD to The New York Times C... Transaction date Feb 1, 2026 Transaction ID ABC123",
			wantErr: false,
			expected: map[string]string{
				"amount": "$4.04",
				"payee":  "The New York Times C...",
			},
		},
		{
			name:    "sent money with receipt",
			snippet: "Michael Lu, here's your receipt. Hello, Michael Lu You sent $24.00 USD to Daryl Wong Transaction Details Transaction ID 5PS81628G40863813 Transaction date March 7, 2026 Money sent $24.00 USD Fee",
			wantErr: false,
			expected: map[string]string{
				"amount": "$24.00",
				"payee":  "Daryl Wong",
			},
		},
		{
			name:    "contribution",
			snippet: "Michael Lu, check out the details. Hello, Michael Lu You contributed $8.00 USD to Caroline Vang Contribution details Pool name Onigiri Fundraiser Contribution sent to Caroline Vang Transaction ID",
			wantErr: false,
			expected: map[string]string{
				"amount": "$8.00",
				"payee":  "Caroline Vang",
			},
		},
		{
			name:    "sent money with receipt 2",
			snippet: "Michael Lu, here's your receipt. Hello, Michael Lu You sent $40.00 USD to Kam Cheung Transaction Details Transaction ID 2UR63057978639747 Transaction date March 8, 2026 Money sent $40.00 USD Fee",
			wantErr: false,
			expected: map[string]string{
				"amount": "$40.00",
				"payee":  "Kam Cheung",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &gmail.Message{
				Snippet: tc.snippet,
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{Name: "Date", Value: "Sun, 26 Jan 2026 00:00:00 +0000"},
					},
				},
			}

			p := &ProviderPaypal{Account: "paypal"}
			tx, err := p.GetTransaction(msg)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tx.Amount != tc.expected["amount"] {
				t.Fatalf("expected amount %q, got %q", tc.expected["amount"], tx.Amount)
			}

			if tx.Payee != tc.expected["payee"] {
				t.Fatalf("expected payee %q, got %q", tc.expected["payee"], tx.Payee)
			}

			if note, ok := tc.expected["note"]; ok {
				if tx.Note != note {
					t.Fatalf("expected note %q, got %q", note, tx.Note)
				}
			}
		})
	}
}

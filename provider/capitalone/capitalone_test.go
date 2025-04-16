package capitalone

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/gmail/v1"
)

func TestGetTransaction(t *testing.T) {
	testCases := []struct {
		name     string
		message  *gmail.Message
		expected *ledger.Transaction
		wantErr  bool
	}{
		{
			name: "valid transaction email",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Subject",
							Value: "A new transaction was charged to your account",
						},
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=\"_----iXOnu5gH/sAz9UXjulqIXQ===_11/85-42097-5F7BE116\"",
						},
					},
					Parts: []*gmail.MessagePart{
						{
							Headers: []*gmail.MessagePartHeader{
								{
									Name:  "Content-Type",
									Value: "text/plain; charset=\"UTF-8\"",
								},
							},
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte(`View posted transaction details.
--
Capital One | Venture X
--

A purchase was charged to your account. 

About your Venture X Card ending in 1807

As requested, we're notifying you that on April 16, 2025, at Grocery Store, a pending authorization or purchase in the amount of $22.43 was placed or charged on your Venture X Card.

Note: You'll receive this notification for both purchases and pending authorizations, such as car rentals, hotel reservations and gas purchases, even if an actual transaction hasn't taken place.

Please visit your account to view your pending and posted transactions.`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Capital One", // assuming this is set in provider
				Payee:   "Grocery Store",
				Amount:  "$22.43",
				Date:    time.Date(2025, 4, 16, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "valid transaction email 2",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Subject",
							Value: "A new transaction was charged to your account",
						},
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=\"_----iXOnu5gH/sAz9UXjulqIXQ===_11/85-42097-5F7BE116\"",
						},
					},
					Parts: []*gmail.MessagePart{
						{
							Headers: []*gmail.MessagePartHeader{
								{
									Name:  "Content-Type",
									Value: "text/plain; charset=\"UTF-8\"",
								},
							},
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte(`View posted transaction details.
--
Capital One | Venture X
--

A purchase was charged to your account. 

About your Venture X Card ending in 1807

As requested, we're notifying you that on April 15, 2025, at Large Box Store #5, a pending authorization or purchase in the amount of $3,096.00 was placed or charged on your Venture X Card.

Note: You'll receive this notification for both purchases and pending authorizations, such as car rentals, hotel reservations and gas purchases, even if an actual transaction hasn't taken place.

Please visit your account to view your pending and posted transactions.`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Capital One",
				Payee:   "Large Box Store #5",
				Amount:  "$3,096.00",
				Date:    time.Date(2025, 4, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &ProviderCapitalOne{
				Account: "Capital One", // Set account for testing
			}

			result, err := p.GetTransaction(tc.message)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

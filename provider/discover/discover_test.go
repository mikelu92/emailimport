package discover

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
			name: "old format, text/plain part present",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=abc",
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
								Data: base64.URLEncoding.EncodeToString([]byte(`Transaction Date: August 18, 2025
Merchant: HOLIDAY STATIONS 3826
Amount: $1.00`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Discover",
				Payee:   "HOLIDAY STATIONS 3826",
				Amount:  "$1.00",
				Date:    time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "new format, text/plain part present",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=def",
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
								Data: base64.URLEncoding.EncodeToString([]byte(`Merchant: HOLIDAY STATIONS 3826
Date: August 27, 2025
Amount: $1.00`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Discover",
				Payee:   "HOLIDAY STATIONS 3826",
				Amount:  "$1.00",
				Date:    time.Date(2025, 8, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "html-only (no text/plain)",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=ghi",
						},
					},
					Parts: []*gmail.MessagePart{
						{
							Headers: []*gmail.MessagePartHeader{
								{
									Name:  "Content-Type",
									Value: "text/html; charset=\"UTF-8\"",
								},
							},
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte(`<html><body>Merchant: HOLIDAY STATIONS 3826<br/>Date: August 27, 2025<br/>Amount: $1.00</body></html>`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Discover",
				Payee:   "HOLIDAY STATIONS 3826",
				Amount:  "$1.00",
				Date:    time.Date(2025, 8, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "plain text non-matching, html present",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Content-Type",
							Value: "multipart/alternative; boundary=jkl",
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
								Data: base64.URLEncoding.EncodeToString([]byte(`This is some random text without the required labels.`)),
							},
						},
						{
							Headers: []*gmail.MessagePartHeader{
								{
									Name:  "Content-Type",
									Value: "text/html; charset=\"UTF-8\"",
								},
							},
							Body: &gmail.MessagePartBody{
								Data: base64.URLEncoding.EncodeToString([]byte(`<html><body>Merchant: HOLIDAY STATIONS 3826<br/>Date: August 27, 2025<br/>Amount: $1.00</body></html>`)),
							},
						},
					},
				},
			},
			expected: &ledger.Transaction{
				Account: "Discover",
				Payee:   "HOLIDAY STATIONS 3826",
				Amount:  "$1.00",
				Date:    time.Date(2025, 8, 27, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "non-matching content",
			message: &gmail.Message{
				Payload: &gmail.MessagePart{
					Headers: []*gmail.MessagePartHeader{
						{
							Name:  "Content-Type",
							Value: "text/plain; charset=\"UTF-8\"",
						},
					},
					Body: &gmail.MessagePartBody{
						Data: base64.URLEncoding.EncodeToString([]byte(`This is some unrelated content without labeled lines.`)),
					},
				},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &ProviderDiscover{Account: "Discover"}
			result, err := p.GetTransaction(tc.message)
			if tc.expected == nil {
				assert.NoError(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

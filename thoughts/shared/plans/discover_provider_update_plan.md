# Implementation Plan: Discover provider supports both “Transaction Date” and “Date” formats

## Overview
- Update provider/discover/discover.go to parse Discover emails that present:
  - Original: “Transaction Date: …” then “Merchant: …” then “Amount: …”
  - New: “Merchant: …” then “Date: …” then “Amount: …”
- Make parsing order-agnostic, avoid panics, and use p.Account instead of a hardcoded account.
- Add unit tests for both formats.

## Current State Analysis
- File: provider/discover/discover.go currently:
  - Assumes text/plain body is at msg.Payload.Parts[0] and decodes it directly.
  - Uses a single positional regex requiring “Transaction Date” first.
  - Does not guard regex matching, causing a panic when the pattern doesn’t match.
  - Hardcodes account to "liabilities:anna:discover".
- Other providers show better patterns:
  - provider/capitalone/capitalone.go finds text/plain part recursively (findPlainText).
  - Providers guard regex matches (target, chase, paypal, affinity).

## Desired End State
- Discover provider:
  - Finds the text/plain body robustly (multipart-safe), with safe fallbacks (top-level body, snippet).
  - Extracts date, merchant, and amount by independent label-based regexes; works in any order and for both “Transaction Date” and “Date”.
  - Uses p.Account, returns nil (no panic) when format unrecognized.
  - Parses dates using "January 2, 2006" (and fallback to "January 02, 2006").
- Tests cover both email variants and non-matching cases.

## Key Discoveries
- discover.go:16 uses a single regex requiring Transaction Date first.
- discover.go:37-43 indexes regex submatches without checking match success.
- capitalone.go:72-90 implements findPlainText for multipart traversal.

## What We’re NOT Doing
- No HTML parsing.
- No broader label vocabulary beyond Date/Transaction Date, Merchant, Amount.
- No shared helper extraction refactor (can be a follow-up).

## Implementation Approach
- Replace single regex with three independent, multiline regexes:
  - Date: ^(?:Transaction Date|Date):\s*(?P<date>.+)$
  - Merchant: ^Merchant:\s*(?P<payee>.+)$
  - Amount: ^Amount:\s*(?P<amt>[\$\d,]+\.\d{2})$
- Add local findPlainText (copied from Capital One provider) to locate text/plain recursively.
- Decode body via base64.URLEncoding, with fallback to base64.RawURLEncoding.
- If no text/plain, try top-level msg.Payload.Body, then msg.Snippet.
- Guard all regex matches; if any field missing, return nil, nil.
- Parse date with "January 2, 2006", fallback "January 02, 2006".
- Initialize t := ledger.Transaction{Account: p.Account}.

## Phase 1: Refactor discover.go

### Overview
- Make parsing robust and order-agnostic. Remove panic risk. Use p.Account.

### Changes Required
1) Update imports in provider/discover/discover.go
- Ensure these are present:
  - "encoding/base64"
  - "errors"
  - "regexp"
  - "strings"
  - "time"
  - "github.com/mikelu92/emailimport/pkg/ledger"
  - "google.golang.org/api/gmail/v1"

2) Replace global regex and init with three independent regexes
- Remove:
  - var expTr *regexp.Regexp
  - init() compiling Transaction Date → Merchant → Amount pattern
- Add:
  - var reDate, reMerchant, reAmount *regexp.Regexp
  - init() compiling the three patterns below

3) Add findPlainText helper (copied from Capital One provider)
- ErrPartNotFound error
- findPlainText(msg *gmail.MessagePart) (*gmail.MessagePartBody, error)

4) Replace GetTransaction logic
- Use p.Account
- Extract X-MSG-ID if present
- Determine body text: try findPlainText; then msg.Payload.Body; then msg.Snippet
- Decode base64 using URLEncoding with RawURLEncoding fallback
- Run label-based regexes independently, guard matches
- Parse date with 2 layouts
- Return &t or nil if not recognized

### Specific code to add/modify

Replace provider/discover/discover.go with:

```go
package discover

import (
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/mikelu92/emailimport/pkg/ledger"
	"google.golang.org/api/gmail/v1"
)

var (
	reDate     *regexp.Regexp
	reMerchant *regexp.Regexp
	reAmount   *regexp.Regexp
)

func init() {
	// Multiline, order-agnostic label matchers
	reDate = regexp.MustCompile(`(?m)^(?:Transaction Date|Date):\s*(?P<date>.+)$`)
	reMerchant = regexp.MustCompile(`(?m)^Merchant:\s*(?P<payee>.+)$`)
	reAmount = regexp.MustCompile(`(?m)^Amount:\s*(?P<amt>[\$\d,]+\.\d{2})$`)
}

type ProviderDiscover struct {
	Account string
}

func (p *ProviderDiscover) GetTransaction(msg *gmail.Message) (*ledger.Transaction, error) {
	t := ledger.Transaction{Account: p.Account}

	// ID (optional)
	for _, h := range msg.Payload.Headers {
		if h.Name == "X-MSG-ID" {
			t.ID = h.Value
			break
		}
	}

	// Determine plain text body: prefer text/plain part, fallback to top-level body, then snippet
	var bodyText string

	if body, err := findPlainText(msg.Payload); err == nil && body != nil && body.Data != "" {
		if decoded, err := decodeBase64(body.Data); err == nil {
			bodyText = string(decoded)
		} else {
			return nil, err
		}
	} else if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		if decoded, err := decodeBase64(msg.Payload.Body.Data); err == nil {
			bodyText = string(decoded)
		} else {
			return nil, err
		}
	} else {
		// As a last resort, use the snippet (may or may not contain labeled lines)
		bodyText = msg.Snippet
	}

	// Extract fields using independent label regexes
	fields := map[string]string{}

	if m := reDate.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reDate.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}
	if m := reMerchant.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reMerchant.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}
	if m := reAmount.FindStringSubmatch(bodyText); len(m) > 0 {
		for i, name := range reAmount.SubexpNames() {
			if i != 0 && name != "" {
				fields[name] = strings.TrimSpace(m[i])
			}
		}
	}

	// Require all three fields to consider this a Discover transaction
	dateStr, hasDate := fields["date"]
	payee, hasPayee := fields["payee"]
	amt, hasAmt := fields["amt"]
	if !hasDate || !hasPayee || !hasAmt {
		return nil, nil
	}

	// Parse date ("January 2, 2006" handles single- and double-digit days; fallback to "January 02, 2006")
	d, err := time.Parse("January 2, 2006", dateStr)
	if err != nil {
		d, err = time.Parse("January 02, 2006", dateStr)
		if err != nil {
			return nil, err
		}
	}

	t.Payee = payee
	t.Amount = amt
	t.Date = d

	return &t, nil
}

func (p *ProviderDiscover) GetAccount() string {
	return p.Account
}

// --- helpers ---

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

func decodeBase64(s string) ([]byte, error) {
	// Try padded base64url first, then raw (unpadded)
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}
```

## Success Criteria

### Automated Verification:
- go test ./... passes.
- New Discover tests pass:
  - Old format (Transaction Date → Merchant → Amount) recognized.
  - New format (Merchant → Date → Amount) recognized.
  - Non-matching or HTML-only returns (nil, nil), no panic.
- Static checks (go vet) report no issues.

### Manual Verification:
- Run the app against actual Discover emails:
  - Both formats produce correct date, payee, amount.
  - Accounts are assigned from config (not hardcoded).
  - Unrecognized formats are skipped without crashing.

## Phase 2: Tests

### Overview
- Add unit tests to verify both formats and safety on non-matching cases.

### Changes Required
- Create provider/discover/discover_test.go with cases:

1) Old format, text/plain part present
- Body:
  Transaction Date: August 18, 2025
  Merchant: HOLIDAY STATIONS 3826
  Amount: $1.00
- Expect:
  - Payee: "HOLIDAY STATIONS 3826"
  - Amount: "$1.00"
  - Date: time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC)
  - Account: set via ProviderDiscover{Account: "Discover"}

2) New format, text/plain part present
- Body:
  Merchant: HOLIDAY STATIONS 3826
  Date: August 27, 2025
  Amount: $1.00
- Expect:
  - Date: 2025-08-27
  - Same payee/amount/account as above

3) HTML-only (no text/plain)
- Provide only a text/html part with similar content.
- Expect: result is nil, err is nil.

4) Non-matching content
- Provide text/plain without labeled lines.
- Expect: result is nil, err is nil.

### Test fixture guidance:
- Use gmail.Message with nested gmail.MessagePart structure similar to provider/capitalone/capitalone_test.go.
- Encode text/plain bodies with base64.URLEncoding.EncodeToString([]byte(...)).
- Set Subject/Header fields only as needed; X-MSG-ID optional.

### Testing Strategy

#### Unit Tests:
- Extract date/payee/amt correctly for both formats.
- Guarded regex (no panic) when labels are missing.
- Fallback to snippet when no body decode possible (optional test).

#### Integration Tests:
- Not required.

### Manual Testing Steps:
1) Run the program and label actual Discover emails to be processed.
2) Verify that transactions are emitted with correct fields for both formats.
3) Confirm unrecognized messages are skipped with a log, not a crash.

## Performance Considerations
- Negligible; regex and base64 decode on small message parts.

## Migration Notes
- None. No data persistence or schema changes.

## References
- provider/discover/discover.go (to modify)
- provider/capitalone/capitalone.go (findPlainText reference)
- provider/capitalone/capitalone_test.go (Gmail message test fixtures pattern)
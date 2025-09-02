# Discover: HTML fallback when plain text yields no match

## Overview
Add an HTML fallback for Discover emails: if parsing the text/plain body yields incomplete fields (or there’s no text/plain), decode the text/html part, convert it to text, and re-run the same label regexes before falling back to snippet.

## Current State Analysis
- Body selection: prefer text/plain part; else top-level body; else snippet (provider/discover/discover.go:45-71)
- Extraction: order-agnostic label regexes for Date/Transaction Date, Merchant, Amount (21-26); applied at 76-96
- Guards: returns nil if any field missing (98-109)
- Helper: findPlainText (138-156)
- Tests: HTML-only currently expected nil (provider/discover/discover_test.go:89-114)
- Logging: good observability of body source and failures (43, 54, 62, 71, 107, 116, 125)

## Desired End State
- Discover tries text/plain first; if fields incomplete, tries text/html.
- HTML is converted to text preserving line breaks so the existing regexes match.
- If still incomplete, retain existing fallbacks (top-level body, then snippet).
- Tests updated so HTML-only messages parse successfully.

## Key Discoveries
- Existing robust regexes can be reused: discover.go:21-26, 76-96
- findPlainText is recursive; we can mirror it for HTML: 138-156
- Chase already uses x/net/html; dependency is in go.sum: provider/chase/chase.go:69-123

## What We’re NOT Doing
- No provider-wide refactor beyond Discover.
- No new labels beyond Date/Transaction Date, Merchant, Amount.
- No heavy HTML scraping; just HTML-to-text conversion.
- No changes to main.go logging/flow.

## Implementation Approach
- Add findHTML(msg *gmail.MessagePart) mirroring findPlainText but matching text/html.
- Add htmlToText(htmlStr string) using golang.org/x/net/html:
  - Convert <br>, <p>, <div>, <li>, etc. into newlines.
  - Concatenate text nodes, then html.UnescapeString and normalize whitespace.
- Modify GetTransaction to:
  1) Try text/plain body and parse (existing behavior).
  2) If fields incomplete, try text/html:
     - Decode via decodeBase64
     - htmlToText -> re-run same regexes
     - If fields complete, continue; else proceed to existing fallbacks (top-level body, then snippet).
- Keep current logging, but add which body source ultimately succeeded (“text/html part”).

## Phase 1: Parser updates

### Overview
Implement HTML fallback and reuse existing regex extraction.

### Changes Required
- File: provider/discover/discover.go
  1) Imports:
     - Add "golang.org/x/net/html"
     - Add "html" (for html.UnescapeString)
  2) Add findHTML (mirrors findPlainText):
     - Recursively traverse multipart parts
     - Return msg.Body when Content-Type contains "text/html"
     - Optionally, if Content-Type header is absent, check part.MimeType contains "text/html"
  3) Add htmlToText:
     - Tokenize HTML
     - On StartTag/SelfClosingTag for br/p/div/li/tr/td/h1–h6: insert newline
     - On TextToken: append trimmed text
     - At end: html.UnescapeString, replace CRLF with LF, collapse multiple spaces where appropriate
  4) Update GetTransaction flow:
     - After first extraction attempt on bodyText, if any of date/payee/amt missing, try HTML:
       - body, err := findHTML(msg.Payload)
       - if found → decodeBase64 → htmlToText → re-run reDate/reMerchant/reAmount extraction
       - if complete → parse date and return transaction
     - If still incomplete, keep existing preview log and return nil.

### Specific code to add/modify (sketch)
- New helper:
  - func findHTML(msg *gmail.MessagePart) (*gmail.MessagePartBody, error) { … } // mirror findPlainText; match "text/html"
  - func htmlToText(s string) string { … } // use html tokenizer to collect text and newlines
- In GetTransaction:
  - Right before returning nil due to missing fields (currently 102-109), insert HTML-fallback attempt; if success, proceed to date parsing and return.

### Success Criteria

#### Automated Verification:
- go test ./... passes
- Discover tests updated:
  - Old plain-text format parsed
  - New plain-text format parsed
  - HTML-only now parsed (previously expected nil)
  - Case where plain-text present but non-matching and HTML present parses via HTML fallback
- go vet ./... has no new issues
- go build ./... succeeds

#### Manual Verification:
- Run against real Discover alerts (like your example):
  - Fields parsed correctly from HTML when plain text lacks them
  - Log indicates body source “text/html part” when applicable
  - Unrecognized messages still log and continue without crashing

## Phase 2: Tests

### Overview
Expand tests to cover HTML fallback.

### Changes Required
- File: provider/discover/discover_test.go
  - Update “html-only (no text/plain)” to expect a parsed transaction using the sample HTML structure:
    - Merchant: HOLIDAY STATIONS 3826
    - Date: August 27, 2025
    - Amount: $1.00
  - Add case: “plain text non-matching, html present”:
    - multipart/alternative with text/plain lacking labels + text/html containing labels → expect parsed result
  - Keep existing old/new plain-text cases passing.

### Testing Strategy

#### Unit Tests:
- Verify extraction from HTML preserves line semantics for regexes to match.
- Verify fallback order: plain-text preferred; HTML only used if plain-text fails.
- Verify date parsing unchanged.

#### Integration Tests:
- Not required.

#### Manual Testing Steps:
1) Run the program against a Discover HTML-only alert (your provided payload shape).
2) Confirm parsed transaction and log showing text/html fallback.
3) Test a message with plain text lacking labels but HTML with labels; verify HTML fallback.

## Performance Considerations
- Minimal impact; HTML tokenization on small email parts is cheap.

## Migration Notes
- None.

## References
- provider/discover/discover.go: 21-26, 45-71, 73-109, 138-156
- provider/discover/discover_test.go: 21-87, 89-114
- provider/chase/chase.go: 69-123 (x/net/html usage)
- thoughts/shared/research/2025-09-02_10-58-36_discover_panic_index_out_of_range.md
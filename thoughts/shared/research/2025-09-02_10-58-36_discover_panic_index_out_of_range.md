---
date: 2025-09-02T11:00:00-05:00
researcher: mlu
jj_commit: 0b7ba6d55c5a512f31e8461ebd4cad633c0be23d
branch: master
repository: emailimport
topic: "Panic: index out of range at provider/discover/discover.go:41 during GetTransaction"
tags: [research, codebase, provider, discover, regex, gmail]
status: complete
last_updated: 2025-09-02
last_updated_by: mlu
---

# Research: Panic: index out of range at provider/discover/discover.go:41 during GetTransaction

**Date**: 2025-09-02 10:58:36 CDT
**Researcher**: mlu
**JJ Change**: kpsmwqytsktqlzxmmlnpuzwlpslrnquo
**Bookmark(s)**: master
**Repository**: emailimport

## Research Question
I got this error, research why this is the case:
panic: runtime error: index out of range [1] with length 0

goroutine 1 [running]:
github.com/mikelu92/emailimport/provider/discover.(*ProviderDiscover).GetTransaction(0x14000607e30?, 0x14000318180)
        /Users/mlu/code/emailimport/provider/discover/discover.go:41 +0x394
main.main()
        /Users/mlu/code/emailimport/main.go:190 +0x534

## Summary
- The panic is caused by indexing into a nil/empty slice of regex submatches in ProviderDiscover.GetTransaction when the email body does not match the expected pattern. Specifically, b := expTr.FindStringSubmatch(...) returns nil/empty, and the code accesses b[i] without checking length.
- This most likely happens because the code decodes and parses the first MIME part (often HTML or a different part), not guaranteed to be the plain text body required by the regex, so the pattern doesn’t match.

## Detailed Findings

### Discover provider parsing (root cause)
- provider/discover/discover.go:37-43
  - b := expTr.FindStringSubmatch(string(s))
  - The subsequent loop assigns tMap[name] = b[i] without verifying that a match succeeded.
  - If no match, b is nil/empty → indexing b[i] panics with "index out of range" (observed at line 41).
- The message body is obtained from msg.Payload.Parts[0].Body.Data (line 32). This assumes:
  - The message is multipart
  - The desired text/plain part is at index 0
  - The content is plain text (not HTML)
  - Base64 URL encoding with padding (URLEncoding) is used
  Any of these being false can result in the regex not matching, leaving b empty and causing the panic.

### Main call site and panic behavior
- main.go:190-196 calls p.GetTransaction(msg) and checks for error return, but there is no panic recovery. A panic inside GetTransaction bypasses error handling and crashes the program. A similar call exists in thread processing at main.go:243-249.

### Safer patterns in other providers
- provider/target/target.go:25-33 guards against empty regex matches before indexing.
- provider/chase/chase.go:38-41 checks len(match)==0 and returns early.
- provider/paypal/paypal.go:33-41,57-64 checks match length for primary/secondary patterns before indexing.
- provider/capitalone/capitalone.go lacks a guard and could suffer a similar issue if the regex fails to match, though it first verifies the Subject contains a known phrase.

## Code References
- provider/discover/discover.go:37-43 — Direct indexing of regex submatches without checking match success
- provider/discover/discover.go:32-35 — Assumes first MIME part contains decodable/plain-text content
- main.go:190-196 — Calls GetTransaction; errors handled but panics are unhandled
- provider/target/target.go:25-33 — Example of guarding len(match) before indexing
- provider/chase/chase.go:31-41,50-53 — Guarded match access
- provider/paypal/paypal.go:33-41,57-64 — Guarded match and optional submatch handling

## Architecture Insights
- Gmail messages are often multipart. Relying on msg.Payload.Parts[0] is brittle; selecting the text/plain part (like capitalone.findPlainText) is more robust.
- Regex extraction should always check len(match) > 0 before indexing submatches. Named-group extraction loops must be conditioned on a successful match.
- Consider using base64.RawURLEncoding for unpadded base64url bodies, or simply handle the error already returned by DecodeString.
- HTML bodies won’t match plain-text line-based regex patterns; if only HTML is present, either parse HTML or fall back to snippet/headers.

## Historical Context (from thoughts/)
- No relevant documents found.

## Related Research
- N/A

## Open Questions
- What exact MIME structure do Discover notification emails use in your mailbox (text/plain vs text/html order)?
- Do Discover emails ever omit "Transaction Date", "Merchant", or "Amount" or vary their labels?
- Should ProviderDiscover adopt a findPlainText approach and add len(match) checks similar to other providers?

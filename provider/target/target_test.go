package target

import (
	"encoding/base64"
	"testing"
	"time"

	"google.golang.org/api/gmail/v1"
)

var exampleEmail = `<center class="darkMode-background" style="width: 100%; background-color: #f7f7f7;">
<div style="display: none; font-size: 1px; line-height: 1px; max-height: 0px; max-width: 0px; opacity: 0; overflow: hidden; mso-hide: all; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;"> </div>
<div style="display: none; font-size: 1px; line-height: 1px; max-height: 0px; max-width: 0px; opacity: 0; overflow: hidden; mso-hide: all; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif;">‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ ‌ </div>
<div class="email-container" style="max-width: 640px; margin: 0 auto;">
<table class="wrapper" style="margin: auto;" role="presentation" border="0" width="640" cellspacing="0" cellpadding="0" align="center">
<tbody>
<tr>
<td class="w270" style="width: 610px;" align="left" width="610px"><img src="https://target.com/img.jpg" alt="Target Circle Card" width="100%" height="auto"></td>
</tr>
<tr>
<td class="side-margin_mobile w290 darkMode-background" style="background-color: #f7f7f7; padding: 20px 30px; width: 540px;">
<table border="0" cellspacing="0" cellpadding="0">
<tbody>
<tr>
<td class="pad15 w290 darkMode-background-light" style="background-color: #ffffff; padding: 10px 20px; text-align: center; width: 560px; color: #333333; border-radius: 10px;" width="560">
<table class="block-table" border="0" cellspacing="0" cellpadding="0">
<tbody>
<tr>
<td class="w260 show center vertPad0" style="font-size: 28px; line-height: 32px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; width: 560px; text-align: left;" width="560">
<table role="presentation" border="0" cellspacing="0" cellpadding="0" align="center">
<tbody>
<tr>
<td class="w260 darkMode-text" style="font-size: 28px; line-height: 32px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; width: 560px; text-align: left;"><strong>Your Target Circle Card transaction exceeds $0.01</strong></td>
</tr>
<tr>
<td class="w260 show center darkMode-text" style="font-size: 18px; line-height: 22px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; width: 560px; text-align: left; padding: 10px 0 16px; font-weight: normal;">
<p style="margin-top: 16px; margin-bottom: 16px;">For <span class="darkMode-text-red" style="color: #cc0000; font-weight: bold;">Target Circle Card</span> ending in 5432</p>
<p style="margin-top: 16px; margin-bottom: 16px;">Hello NAME,</p>
<p style="margin-top: 16px; margin-bottom: 16px;">A transaction of $19.99 at TARGET T-1234 has been approved on your <span class="darkMode-text-red" style="color: #cc0000; font-weight: bold;">Target Circle™ Card</span>.</p>
<p style="margin-top: 16px; margin-bottom: 16px;">To view your card activity or update your alert settings, go to <a style="text-decoration: none; color: #4a90e2;" href="https://Target.com/myCircleCard" target="_blank" rel="noopener">Target.com/myCircleCard</a>.</p>
</td>
</tr>
<tr>
<td><a style="background-color: #cc0000; font-size: 16px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; text-decoration: none; padding: 10px 30px; color: #ffffff; display: inline-block; miso-padding-alt: 0;" href="https://target.com"> <span style="mso-text-raise: 15pt;">View Your Account</span> </a></td>
</tr>
</tbody>
</table>
</td>
</tr>
</tbody>
</table>
</td>
</tr>
</tbody>
</table>
</td>
</tr>
<tr>
<td>
<table border="0" width="100%" cellspacing="0" cellpadding="0px 0 0px 0">
<tbody>
<tr>
<td>
<table border="0" width="100%" cellspacing="0" cellpadding="0">
<tbody>
<tr>
<td class="mobile-description-pad" style="padding: 30px 50px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; line-height: 24px; background-color: #cc0000;" align="center"> </td>
</tr>
</tbody>
</table>
</td>
</tr>
<tr>
<td class="mobile-padding" style="padding-bottom: 20px;" align="center">
<table class="wrapper mobile-padding" style="width: 640px;" border="0" width="640" cellspacing="0" cellpadding="0" align="center">
<tbody>
<tr>
<td class="darkMode-background darkMode-text" style="text-align: left; padding-top: 24px; padding-right: 20px; padding-left: 20px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #ffffff; font-size: 11px; line-height: 12px;" bgcolor="#ffffff"><strong>Target Circle™ Card</strong>: Target Debit Card, Target Credit Card, and Target<sup style="font-size: 60%; line-height: 1.2em; vertical-align: text-top;">TM</sup> Mastercard<sup style="font-size: 60%; line-height: 1.2em; vertical-align: text-top;">®</sup>. Subject to application approval. The Target Circle debit card is issued by Target Corporation. The Target Circle credit cards (Target Credit Card and Target Mastercard) are issued by TD Bank USA, N.A. Mastercard is a registered trademark of Mastercard International, Inc.</td>
</tr>
<tr>
<td class="darkMode-background darkMode-text" style="text-align: left; padding-top: 20px; padding-right: 20px; padding-left: 20px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #ffffff; font-size: 11px; line-height: 12px;" bgcolor="#ffffff"><span style="font-style: italic;"><strong>This email was sent from a notification-only address that cannot accept incoming email. Please do not reply to this message.</strong></span></td>
</tr>
<tr>
<td class="darkMode-background darkMode-text" style="text-align: left; padding-top: 20px; padding-right: 20px; padding-left: 20px; font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #ffffff; font-size: 11px; line-height: 12px;" bgcolor="#ffffff">Target Card Services, Target Corporation, Mail Stop NCB-0461, P.O. Box 673, Minneapolis, MN 55440<br>©2021 Target Brands Inc. Target and Bullseye Design are registered trademarks of Target Brands, Inc.</td>
</tr>
</tbody>
</table>
</td>
</tr>
</tbody>
</table>
</td>
</tr>
</tbody>
</table>
</div>
</center>`

func TestGetTransactionSampleEmail(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte(exampleEmail))

	msg := &gmail.Message{
		Payload: &gmail.MessagePart{
			Headers: []*gmail.MessagePartHeader{
				{Name: "Received", Value: "by 2002:a05:7301:3d18:b0:2a4:605a:ae3c with SMTP id oe24csp620026dyb; Fri, 16 Jan 2026 15:17:15 -0800 (PST)"},
			},
			MimeType: "text/html",
			Body:     &gmail.MessagePartBody{Data: encoded},
		},
	}

	p := &ProviderTarget{Account: "target:circle"}
	tx, err := p.GetTransaction(msg)
	if err != nil {
		t.Fatalf("GetTransaction returned error: %v", err)
	}
	if tx == nil {
		t.Fatalf("expected transaction, got nil")
	}
	if tx.Amount != "$19.99" {
		t.Fatalf("expected amount $19.99, got %q", tx.Amount)
	}
	if tx.Payee != "TARGET T-1234" {
		t.Fatalf("expected payee %q, got %q", "TARGET T-1234", tx.Payee)
	}
	expectedDate, err := time.Parse("Mon, 2 Jan 2006 15:04:05 -0700 (MST)", "Fri, 16 Jan 2026 15:17:15 -0800 (PST)")
	if err != nil {
		t.Fatalf("failed to parse expected date: %v", err)
	}
	if !tx.Date.Equal(expectedDate) {
		t.Fatalf("expected date %v, got %v", expectedDate, tx.Date)
	}
	if tx.Account != "target:circle" {
		t.Fatalf("expected account %q, got %q", "target:circle", tx.Account)
	}
}

func TestGetTransactionNoMatch(t *testing.T) {
	htmlBody := `<html><body><p>This is not a Target email</p></body></html>`
	encoded := base64.URLEncoding.EncodeToString([]byte(htmlBody))

	msg := &gmail.Message{
		Payload: &gmail.MessagePart{
			Headers:  []*gmail.MessagePartHeader{},
			MimeType: "text/html",
			Body:     &gmail.MessagePartBody{Data: encoded},
		},
	}

	p := &ProviderTarget{Account: "target:circle"}
	tx, err := p.GetTransaction(msg)
	if err != nil {
		t.Fatalf("GetTransaction returned error: %v", err)
	}
	if tx != nil {
		t.Fatalf("expected nil transaction, got %v", tx)
	}
}

package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Matchowl rebrand of the DB-persisted system emails (verification, password
// reset, login alert from 0025; email change from 0026): the "NIGHT OWL"
// shell — warm near-black, owl orange, Matchowl wordmark — replacing the
// WM Tips floodlight look. Structure and placeholders are unchanged.

// owlEmail wraps content in the Matchowl-branded shell, with an optional CTA
// button. Palette mirrors frontend/src/lib/theme.css (dark).
func owlEmail(content, ctaText, ctaURL string) string {
	cta := ""
	if ctaURL != "" {
		cta = `<table role="presentation" cellpadding="0" cellspacing="0" style="margin:26px 0 6px;"><tr><td bgcolor="#ff7700" style="border-radius:999px;background:#ff7700;">
<a href="` + ctaURL + `" target="_blank" rel="noopener" style="display:inline-block;padding:13px 30px;font-size:15px;font-weight:800;letter-spacing:.01em;color:#1c0e00;text-decoration:none;">` + ctaText + ` &rarr;</a>
</td></tr></table>`
	}
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="dark">
<meta name="supported-color-schemes" content="dark">
</head>
<body style="margin:0;padding:0;background:#120c07;color:#fff3e6;font-family:'Figtree',system-ui,-apple-system,'Segoe UI',Roboto,sans-serif;-webkit-font-smoothing:antialiased;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" bgcolor="#120c07" style="background:#120c07;padding:28px 14px;">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background:#1c140c;border:1px solid #3d2e1e;border-radius:20px;overflow:hidden;">

<tr><td height="5" style="height:5px;line-height:5px;font-size:0;background:#ff7700;">&nbsp;</td></tr>

<tr><td style="padding:18px 28px;border-bottom:1px solid #3d2e1e;">
<table role="presentation" cellpadding="0" cellspacing="0"><tr>
<td style="padding-right:10px;vertical-align:middle;"><img src="cid:mark" width="28" height="28" alt="" style="display:block;border:0;height:28px;width:28px;border-radius:7px;"></td>
<td style="vertical-align:middle;font-size:18px;font-weight:800;letter-spacing:.01em;color:#fff3e6;">Match<span style="color:#ff7700;">owl</span></td>
</tr></table>
</td></tr>

<tr><td style="padding:30px 28px 8px;">
` + content + cta + `
</td></tr>

<tr><td style="padding:20px 28px 24px;border-top:1px solid #3d2e1e;color:#8a7460;font-size:12px;line-height:1.6;">
<strong style="color:#b39a82;">Matchowl</strong> — football predictions with your friends.<br>
You're getting this about your Matchowl account. If you didn't request it, you can ignore this email.
</td></tr>

</table>
</td></tr>
</table>
</body>
</html>`
}

func owlKicker(label string) string {
	return `<div style="font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:#ff7700;">` + label + `</div>`
}

var owlVerificationBody = owlEmail(owlKicker("Account")+`
<h1 style="margin:7px 0 14px;font-size:27px;line-height:1.12;font-weight:800;color:#fff3e6;">Verify your email</h1>
<p style="margin:0;font-size:15px;line-height:1.6;color:#b39a82;">
Confirm this address to activate kickoff reminders, matchday recaps and league alerts for your {APP_NAME} account.
</p>`,
	"Verify email", "{APP_URL}/confirm-verification/{TOKEN}")

var owlResetPasswordBody = owlEmail(owlKicker("Account")+`
<h1 style="margin:7px 0 14px;font-size:27px;line-height:1.12;font-weight:800;color:#fff3e6;">Reset your password</h1>
<p style="margin:0;font-size:15px;line-height:1.6;color:#b39a82;">
Click the button below to choose a new password for your {APP_NAME} account. The link is valid for a limited time.
</p>`,
	"Reset password", "{APP_URL}/confirm-password-reset/{TOKEN}")

var owlAuthAlertBody = owlEmail(owlKicker("Security")+`
<h1 style="margin:7px 0 14px;font-size:27px;line-height:1.12;font-weight:800;color:#fff3e6;">Login from a new location</h1>
<p style="margin:0 0 20px;font-size:15px;line-height:1.6;color:#b39a82;">
We noticed a login to your {APP_NAME} account from a new location:
</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr>
<td bgcolor="#271c11" style="background:#271c11;border:1px solid #3d2e1e;border-radius:12px;padding:14px 18px;font-size:14px;line-height:1.6;color:#fff3e6;">
{ALERT_INFO}
</td></tr></table>
<p style="margin:20px 0 0;font-size:15px;line-height:1.6;color:#b39a82;">
<strong style="color:#fff3e6;">If this wasn't you</strong>, change your password right away to revoke access from all other locations. If this was you, you can disregard this email.
</p>`,
	"Open settings", "{APP_URL}/settings")

var owlEmailChangeBody = owlEmail(owlKicker("Account")+`
<h1 style="margin:7px 0 14px;font-size:27px;line-height:1.12;font-weight:800;color:#fff3e6;">Confirm your new email</h1>
<p style="margin:0;font-size:15px;line-height:1.6;color:#b39a82;">
Click the button below to make this address the new email for your {APP_NAME} account. You'll be asked for your account password to confirm.
</p>`,
	"Confirm new email", "{APP_URL}/confirm-email-change/{TOKEN}")

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.VerificationTemplate.Body = owlVerificationBody
		users.ResetPasswordTemplate.Body = owlResetPasswordBody
		users.AuthAlert.EmailTemplate.Body = owlAuthAlertBody
		users.ConfirmEmailChangeTemplate.Body = owlEmailChangeBody
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		// Restore the WM Tips bodies from 0025/0026.
		users.VerificationTemplate.Body = sysVerificationBody
		users.ResetPasswordTemplate.Body = sysResetPasswordBody
		users.AuthAlert.EmailTemplate.Body = sysAuthAlertBody
		users.ConfirmEmailChangeTemplate.Body = sysEmailChangeBody
		return app.Save(users)
	})
}

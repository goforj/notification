# GoForj Mail integration

The optional `github.com/goforj/notification/driver/mail` module borrows an existing GoForj Mailer. Configure the email provider once in Mail, then pass that Mailer to `notificationmail.New(mailer, recipients...)`. The adapter does not duplicate credentials, select providers, close the Mailer, or start background work. The root notification module remains independent of email dependencies.

| Email transport | Mail package | Notification integration |
| --- | --- | --- |
| SMTP, including compatible hosted SMTP services | github.com/goforj/mail/mailsmtp | Pass its configured Mailer to the adapter |
| SendGrid | github.com/goforj/mail/mailsendgrid | Same adapter |
| Resend | github.com/goforj/mail/mailresend | Same adapter |
| Mailgun | github.com/goforj/mail/mailmailgun | Same adapter |
| Postmark | github.com/goforj/mail/mailpostmark | Same adapter |
| Amazon SES | github.com/goforj/mail/mailses | Same adapter; requires the separate Mail SES module |

The current adapter uses Mail v0.3.1. Mail owns provider configuration, default sender and reply-to settings, MIME encoding, address/header validation and network policy. Configure HTTP clients and transports on the Mail driver. Notification's HTTPConfig does not apply to an independently configured Mailer.

A common notification Message maps Title to the subject and Body to plain text. Destinations come from adapter configuration. For application recipient planning, the existing typed mail.Payload contains Subject, Text and HTML, while the planned endpoint supplies one recipient. Provider-specific email features such as attachments remain in Mail; they do not become untyped fields on the common notification Message.

The [runnable Mail example](../examples/mail/main.go) uses the real Resend driver with a local HTTP fixture and combines it with Slack through Fanout. The application can keep using the same Mailer for direct email. Both layers borrow it. The executable example is also projected into the [README](../README.md#mail-delivery).

Tests exercise real SMTP envelope/MIME handling against a local protocol fixture, and real Resend, SendGrid, Mailgun and Postmark drivers against local provider fixtures. They cover common Sender behavior, concurrent sends, Mail defaults, caller-owned Mailer reuse and safe handling of provider failures. These tests verify composition and wire contracts, not hosted delivery. SES remains available by delegation; this test suite does not claim a new SES hosted or emulator verification.

The error-only Mail interface cannot reliably prove whether a failed send reached a recipient or was rejected before acceptance. The adapter conservatively retains unknown acceptance and safe cancellation evidence. It does not infer retry safety from arbitrary provider error prose. A nil error means Mail accepted the operation under its own provider contract, never recipient receipt.

New email providers belong in Mail. They become usable here through the same borrowed Mailer interface without adding another notification driver package.

# Provider expansion implementation

The maintainer authorized implementing the integrations identified in the [coverage review](provider-coverage-review.md), together with the horizontal GoForj Mail integration. This work preserves Message, Sender, synchronous execution, explicit transport ownership and the root minimum Go version.

## Progress

- Implemented: Pushover (non-emergency priorities), Matrix (unencrypted room messages), Mattermost, Rocket.Chat, Pushbullet, Bark, Zulip, IFTTT and Join. Native configuration and response fixtures, common Sender conformance, malformed responses, owned/borrowed transport behavior and errors are tested. Hosted verification remains pending.
- Implemented: Mail composition tests using real SMTP, Resend, SendGrid, Mailgun and Postmark drivers against local protocol fixtures, plus a runnable real-driver Mail and Slack fanout example. Email providers remain owned by Mail. SES continues through its separate Mail module; no new hosted SES verification is claimed.
- Pending: Home Assistant, Nextcloud Talk, Signal relay and Apprise API.
- Pending: Web Push, FCM HTTP v1, APNs, SNS, OneSignal and Expo.
- Pending: native WhatsApp Cloud, including explicit template sends.
- Pending: Plivo, Vonage, Telnyx and ClickSend.
- Pending: GoAlert, SIGNL4, PagerTree, Better Stack, incident.io, ilert and FireHydrant.

The initial checkpoint passed 280 unit tests with exact 1428/1428 library statement coverage, every module's race and vet checks, pinned real ntfy/Gotify integration, and deterministic documentation generation. Subsequent checkpoints must regenerate evidence for their own source state. This file is a work record, not a statement that pending providers are supported.

## Boundaries

Pushover emergency receipts, encrypted Matrix rooms, Signal pairing, downstream Apprise providers, mobile delivery receipts and on-call lifecycle operations require explicit protocol scope. Unsupported operations must never be silently represented as a successful ordinary message send. A provider's own scheduling or repetition must not be presented as library-owned retries. Hosted and paired-device verification remains pending unless executed.

Cloud SDK or cryptographic protocol dependencies should remain optional modules when needed, following the existing Mail pattern. Any added module must be included in the module inventory, coverage, race, vet, security and release planning. The root module should remain standard-library-only.

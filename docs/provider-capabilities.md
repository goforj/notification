# Provider capabilities

All 51 drivers accept configured destinations and a small `notification.Message`. They are concurrency-safe and synchronous. A nil error describes provider or relay acceptance, never proof that a person received a message. See [driver contracts](drivers.md) for each provider's native acknowledgement and verification scope.

## Chat, push and automation

Pushover supports priorities -2 through 1. Emergency priority 2 requires receipt management and is deliberately rejected. Matrix sends unencrypted `m.text` to an already joined room, including modern room-version-12 IDs. It generates a fresh transaction ID for each call and does not perform room membership or encryption setup. Mattermost and Rocket.Chat neutralize broadcast mention characters.

Bark, Pushbullet, Zulip, IFTTT and Join use native APIs. Pushbullet selects at most one configured device, email or channel target. Join uses a bounded GET request and rejects an overlong query before sending. None of these drivers uploads attachments or manages devices.

Home Assistant invokes one configured `notify` service. Nextcloud Talk posts to an existing conversation. Signal requires an application-managed, already paired signal-cli REST relay; the library never pairs or stores Signal device credentials. Apprise uses a pre-existing stateful server configuration with an optional tag. It waits for non-streaming HTTP 200 relay acceptance and does not verify each downstream provider.

## Application push

| Driver | Authentication and target | Supported send boundary |
| --- | --- | --- |
| APNs | Optional module; copied P-256 signing key, key/team IDs and topic/device, or an explicit caller-owned certificate transport | Native HTTP/2 alert acceptance and request ID; priority 5 or 10, sound and collapse ID; JSON at most 4096 bytes |
| FCM | Caller-owned concurrency-safe `TokenSource(context.Context) (string, error)` and project/device or topic | HTTP v1 notification, a project-scoped message name; JSON at most 4096 bytes |
| SNS | Optional module; borrowed AWS SDK v2 `Publisher`, with topic ARN, endpoint ARN or E.164 phone | Native message ID; SDK credentials, routing and client lifetime remain caller-owned; one attempt via a per-call retryer override |
| OneSignal | Application ID, API key and one subscription or external identity | Native push message ID; explicit zero recipients is rejected; JSON at most 4096 bytes |
| Expo | Expo/Exponent push token and optional access token | One native accepted ticket; JSON at most 4096 bytes; no receipt polling |
| Web Push | Optional module; browser subscription and VAPID public/private identity | AES128GCM encryption, VAPID authentication and HTTP 201 acceptance; JSON at most 3900 bytes to leave framing room |

APNs caches signed JWTs for 50 minutes and refreshes under a lock. Configure an HTTP/2-capable transport when providing one. Certificate mode requires an explicitly supplied transport with the correct client certificate; selecting the production or sandbox endpoint remains application configuration.

Web Push accepts the browser's `endpoint`, `p256dh` and `auth` values directly through `Subscription` and `Keys`. Subscriber is a contact email or HTTPS URL. TTL is seconds from 0 through 2419200: zero requests immediate delivery without storage. Optional topic is at most 32 URL-safe characters. Payload JSON uses `title` and `body`; the service worker decides how to display it. Expired subscriptions return a permanent error; applications own subscription removal, consent and refresh.

FCM's token callback may use the application's existing Google credential source. It is invoked for each validated Send and should cache/refresh credentials itself. No provider callback or SDK error prose is returned. Neither native acceptance nor an Expo ticket is a mobile delivery receipt. Device registration, platform payload extensions, batching and receipt polling are outside this API.

## WhatsApp Cloud

The Graph API version is required configuration. Text sends combine title and body and are limited to 4096 runes. They require an eligible customer-service window. `SendTemplate` selects an existing approved template name and language, with up to 32 positional body parameters. Header, media and button components are not supported. The full template request is bounded to 16 KiB after JSON encoding, including escaped characters and metadata. The library does not create templates, authorize recipients or infer policy eligibility. Native acknowledgement checks the recipient and WhatsApp message ID.

## SMS

Twilio, Plivo, Vonage, Telnyx and ClickSend reuse the plain-text message path with configured destinations. The four added SMS drivers require E.164 recipient numbers; sender formats follow the provider configuration. Title and body are joined without truncation. New drivers reject more than 1600 runes before I/O. This is an application bound, not a guarantee of one billable SMS segment. Providers own encoding, segmentation, sender registration, opt-out policy and billing. The library never sends an extra segment or retries independently. Native response parsing checks per-message results where available. Vonage checks every counted SMS segment, distinct message IDs and recipient identity. A response containing both accepted and rejected segments remains unknown, so neither fallback nor journal resubmission can resend the whole message. ClickSend preserves native throttling and authentication failures; missing status or internal-error evidence remains unknown.

## On-call lifecycle

| Driver | Correlation | Additional operations |
| --- | --- | --- |
| PagerDuty, JSM, Opsgenie, SIGNL4, PagerTree, ilert | Application-owned stable `IncidentKey` | Trigger, Acknowledge, Resolve |
| GoAlert | Application-owned stable key | Trigger, Resolve (close) |
| incident.io | HTTP alert-source deduplication key | Trigger, Resolve (resolved alert with required recovery title) |
| FireHydrant | Signals idempotency key | Trigger, Resolve (CLOSED event) |
| Better Stack | Native incident ID returned by `Create` | Acknowledge and Resolve by that returned ID |

incident.io accepts a success acknowledgement only when its deduplication key matches that specific trigger or recovery request.

Partial lifecycle drivers intentionally do not satisfy `IncidentSender`. Their ordinary Send requires a fixed configured key, while explicit Trigger accepts a key per incident. Better Stack Send creates an incident without retaining its ID; applications that need subsequent actions use Create and persist the native ID. Better Stack action acknowledgements must echo the native incident ID and the incident resource type. Escalation flags and policies are explicit configuration. No unsupported acknowledgement is simulated, and accepted asynchronous requests are not polled.

## Horizontal Mail integration

[GoForj Mail](mail.md) owns SMTP, Resend, SendGrid, Mailgun, Postmark and optional SES transports, sender defaults and MIME composition. Notification borrows the Mailer and recipients, translating Message to email or allowing its existing rich typed Payload. It does not introduce parallel email provider drivers. Mailer lifetime and retry policy remain application-owned.

## Verification

Local protocol tests cover native paths, authentication, configured targets, acknowledgements, provider errors, cancellation and bounds. Real pinned ntfy, Gotify and Matrix Synapse containers verify storage against actual servers. APNs, SNS and Web Push exercise real local protocol or cryptographic implementations. Mail composition uses actual Mail drivers and local SMTP/HTTP fixtures. These tests do not claim live hosted-provider delivery. Signal pairing and all other hosted-provider verification remain pending.

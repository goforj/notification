# Provider expansion implementation

All 31 integrations selected in the [coverage review](provider-coverage-review.md) are implemented, bringing the catalog to 51 driver packages. The small Message and Sender API, synchronous execution, explicit transport ownership and Go 1.24 minimum remain intact.

- Chat and personal push: Pushover, Matrix, Mattermost, Rocket.Chat, Pushbullet, Bark, Zulip, IFTTT and Join.
- Automation and relays: Home Assistant, Nextcloud Talk, Signal relay and Apprise API.
- Application push: Web Push, FCM HTTP v1, APNs, SNS, OneSignal and Expo.
- WhatsApp Cloud: ordinary text and explicit approved-template sends.
- SMS: Plivo, Vonage, Telnyx and ClickSend.
- On-call: GoAlert, SIGNL4, PagerTree, Better Stack, incident.io, ilert and FireHydrant.

Mail composition uses real SMTP, Resend, SendGrid, Mailgun and Postmark drivers against local fixtures. Its runnable example composes Mail and Slack through Fanout. Email provider implementations remain owned by GoForj Mail; SES continues through Mail's optional SES module.

Native request fixtures check literal paths, authentication, recipients and response acknowledgements. APNs uses real local HTTP/2 and TLS, including certificate authentication. SNS uses an actual AWS SDK client with SigV4 and XML. Web Push tests independently decrypt ciphertext and validate VAPID signatures. Matrix additionally passes a pinned real Synapse homeserver test, joining the existing real ntfy and Gotify tests.

[Unit evidence](unit-results.json) and [integration evidence](integration-results.json) contain the measured counts, source fingerprints and exact server pins. All eleven modules participate in unit, race, vet, module-inventory and security checks. Five runtime modules have independent release tag plans; this work does not publish them.

See [provider capabilities](provider-capabilities.md) for explicit protocol boundaries. Hosted-provider delivery and paired Signal verification remain pending. Historical generic webhook aliases are inventory records, not claims of native support.

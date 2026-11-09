# Notification destination coverage review

This is a historical research snapshot. The selected expansion is now implemented; see [current implementation](provider-expansion.md) and [driver contracts](drivers.md).

Reviewed 2026-11-09 against notification commit 57ff449. The library has 20 driver packages, including Mail and local log capture. This review assesses destination breadth separately from the [API and release-readiness review](api-review.md). A usable API and passing driver tests do not establish broad ecosystem coverage.

The current library covers many chat and incident destinations, but it does not yet justify a claim to cover most notification needs. The clearest omissions are established personal push services and self-hosted chat, followed by browser and application push. No reliable percentage of users covered can be calculated from public provider catalogs.

## Research method

Compared source catalogs from three Go projects, plus Apprise and Uptime Kuma for broader automation and monitoring use cases. The two Shoutrrr catalogs share ancestry and are not independent demand measurements. Repeated inclusion is a useful prioritization signal, not market-share evidence. The comparison uses branch snapshots, not a claim that every entry ships in a particular release.

| Project | Inspected catalog | Why it matters |
| --- | --- | --- |
| nikoksr/notify | [service packages](https://github.com/nikoksr/notify/tree/7f8b6878de7438c90eabcf58d66370a9e4eb3ea6/service) | Go service integrations, including app push and SMS |
| Original Shoutrrr | [service packages](https://github.com/containrrr/shoutrrr/tree/ccf81390b700948f85ce1c79f6de9ee75c68d9e5/pkg/services) | Configured destinations for infrastructure alerts |
| Maintained Shoutrrr fork | [categorized service packages](https://github.com/nicholas-fedor/shoutrrr/tree/d01268755569d91fbb755fda9c59f4eac71b5c77/pkg/services) | Additional self-hosted and gateway destinations |
| Apprise | [plugins](https://github.com/caronc/apprise/tree/dcbd38584f8a9855f2343d369fd1f664ab06b3a7/apprise/plugins) | Broad protocol and regional-provider coverage |
| Uptime Kuma | [notification providers](https://github.com/louislam/uptime-kuma/tree/b376409720f6df948707a3366398248d370bbf51/server/notification-providers) | Monitoring-oriented integration breadth |

The [machine-readable evidence](providers/coverage-research.json) records exact commits, catalog entries, source exceptions, published Mail versions and protocol references. Catalog presence is not a certification of correctness or live service availability. For example, notify marks WhatsApp unsupported despite retaining its package, and its README still lists LINE Notify. Our active LINE Messaging API implementation should remain the target. Infrastructure folders and local-device plugins must not inflate comparable driver counts.

## What is already covered

| Use case | Implemented surface | Boundary |
| --- | --- | --- |
| Mainstream chat | Slack, Discord, Telegram, Teams, Google Chat | Configured destinations and supported native acknowledgement rules |
| Regional chat | DingTalk, Feishu, WeCom, LINE Messaging API | Specific protocols, not interchangeable regional product names |
| Operational incidents | PagerDuty, JSM, legacy Opsgenie | Explicit trigger, acknowledge and resolve capability |
| Self-hosted push | ntfy, Gotify | Local protocol tests plus pinned real-server verification |
| SMS | Twilio | One SMS provider, not broad regional SMS coverage |
| WhatsApp gateways | WAHA, Whapi | Gateway integrations; not native Meta Cloud API support |
| Email | Mail adapter | Delegates to the independently maintained Mail library |
| Extensibility and testing | Generic webhook, SenderFunc, local log and fake | Extensibility does not certify another provider's native protocol |

Published goforj/mail v0.3.1 contains SMTP, SendGrid, Resend, Mailgun and Postmark packages. Amazon SES is available through the separate goforj/mail/mailses v0.3.1 module. Both versions were checked against the public Go module proxy; root Mail source was inspected from its published module cache. These are Mail composition paths, not six new notification driver packages or six newly executed hosted tests. Brevo, Microsoft Graph and Mailtrap should be assessed in Mail when needed.

## Highest-confidence missing destinations

These six appear in all five inspected ecosystems. They are the strongest first additions based on this comparison, without claiming that equal catalog presence means equal demand.

| Destination | Current state | Implementation requirement |
| --- | --- | --- |
| Pushover | Missing | Native Message API, priority limits and status parsing; emergency receipts need an explicit capability |
| Matrix | Missing | Room/token configuration, transaction identity and event acknowledgement; encrypted rooms need separate support |
| Mattermost | Missing | Native incoming webhook configuration, mention policy and documented acknowledgement |
| Rocket.Chat | Missing | Native integration payload and acknowledgement, including server-side integration differences |
| Pushbullet | Missing | Note-push API, configured receiver/device and native acknowledgement |
| Bark | Missing | Configured server/device key, message bounds and native response parsing |

Pushover's [API](https://pushover.net/api) makes emergency priority materially different from a normal message because it returns a receipt and supports provider-side repetition. Do not silently discard that distinction or describe provider-side repetition as notification-owned retrying. Matrix encryption must not be implied by a plain room-send implementation.

Mattermost documents both its [incoming webhook payload and successful response](https://docs.mattermost.com/developers/integrate/webhooks/incoming). Its Slack compatibility is useful evidence, but our Slack mode has not been certified against a Mattermost server. Native naming, configuration, mention behavior and conformance tests still matter. The same caution applies to [Rocket.Chat integrations](https://docs.rocket.chat/docs/integrations).

## Next additions by audience

| Audience | Missing destinations | Priority reasoning |
| --- | --- | --- |
| Remaining common Go-library destinations | Zulip, IFTTT, Join | Along with the six above, these are the nine missing destination families from original Shoutrrr's 20-service catalog, excluding shared infrastructure |
| Self-hosted automation | Home Assistant, Nextcloud Talk, Signal relay | Recurring automation use cases; Signal must be labeled as a paired signal-cli relay integration |
| Browser applications | Web Push | Subscription keys, VAPID and encrypted payloads constitute a separate protocol |
| Mobile applications | FCM HTTP v1, APNs | Required backend families for direct Android/iOS application push |
| Managed application push | OneSignal, Expo | Different audience, token and receipt models; add based on the target application audience |
| AWS applications | SNS | Topic, SMS and platform-endpoint publishing with native credentials and acknowledgement |
| Business messaging | WhatsApp Cloud API | Current WAHA/Whapi coverage does not cover Meta's authenticated, template-aware API |
| SMS provider choice | Plivo, Vonage, Telnyx, ClickSend | Adds alternatives to Twilio; further regional providers should follow concrete user/region requirements |
| Additional on-call platforms | GoAlert, SIGNL4, PagerTree, Better Stack, incident.io, ilert, FireHydrant | Valuable incident-provider candidates; lifecycle operations need protocol-specific review |
| Optional relay breadth | Apprise API | One explicit gateway driver can reach a wider catalog, with a deployed service dependency and relay-specific acceptance semantics |

[Zulip](https://zulip.com/api/send-message) has its own recipient/topic and acknowledgement contract. [Web Push](https://web.dev/articles/push-notifications-web-push-protocol) requires VAPID and payload encryption. [FCM HTTP v1](https://firebase.google.com/docs/cloud-messaging/send/v1-api) uses OAuth-backed authenticated sends, while [APNs](https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns) requires its own HTTP/2, TLS and device-token semantics. A generic JSON webhook is not support for any of these protocols.

[SNS](https://docs.aws.amazon.com/sns/latest/api/API_Publish.html), [OneSignal](https://documentation.onesignal.com/reference/create-message) and [Expo](https://docs.expo.dev/push-notifications/sending-notifications/) target different notification workflows. Expo distinguishes push tickets from later receipts. Meta's [official template-send collection](https://www.postman.com/meta/whatsapp-business-platform/request/o65u5m5/send-message-template-text) requires a template object and native message acknowledgement. These differences should shape provider-specific capabilities rather than enlarge Message with arbitrary option maps.

Signal's relevant integration is [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api), which needs an application-owned paired service. An [Apprise API](https://appriseit.com/api/usage/) driver would likewise be relay support, not native support for every downstream Apprise plugin. If implemented, start with a configured stateful destination rather than exposing credential-bearing downstream URLs in Message.

## Implementation order and acceptance

1. Add the six cross-ecosystem destinations first, followed by Zulip, IFTTT and Join. This closes concrete gaps against the original Go alerting catalog.
2. Add Home Assistant, Nextcloud Talk and an explicitly named Signal relay for the self-hosted audience. Consider Apprise API as optional long-tail coverage.
3. Add Web Push, FCM HTTP v1, APNs and SNS for broader application notifications. Assess OneSignal and Expo alongside that work. An alerting-focused first release should not advertise complete application-push coverage before this group exists.
4. Add native WhatsApp Cloud support with an explicit template-capable boundary, then expand SMS and incident platforms based on audience requirements.
5. Keep email transport expansion in Mail and document those composition paths clearly. Retain the current root API and module requirements unless a specific implementation constraint requires a separate decision.

Every native addition needs typed construction, ownership and transport-injection contracts, the shared Sender suite where applicable, native payload and response tests, direct boundary/failure tests, safe error evidence, cancellation/concurrency coverage, runnable examples and generated catalog/assets. Use real local services or emulators where they actually model the contract. Hosted and paired-device verification must remain explicitly pending until executed. Never label a generic compatibility alias as a native driver.

This research changes no runtime support claims. It is a prioritized coverage plan, not a statement that the missing destinations have been implemented. The prior 67 compatibility aliases remain tracked in the [historical extraction inventory](providers/uptime-gopher.md); not all of them merit equal priority or a native package before release.

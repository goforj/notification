# Driver configuration and delivery contracts

Generated from the driver catalog by `go -C docs run ./readme`. See [portable delivery contracts](contracts.md) and [executed integration evidence](integration-results.json). Provider acceptance never proves recipient delivery.

## Amazon SNS

[Package source](../driver/sns)

- Configuration: Optional module; borrowed AWS SDK v2 publisher and one topic, endpoint or phone.
- Accepted when: Native SNS message ID.
- Verification: Local protocol tests; hosted verification pending.
- Optional module: `github.com/goforj/notification/driver/sns`. See [installation](../README.md#installation).
- [Provider protocol reference](https://docs.aws.amazon.com/sns/latest/api/API_Publish.html).

## Apple APNs

[Package source](../driver/apns)

- Configuration: Optional module; device/topic and ES256 credentials or explicit certificate transport.
- Accepted when: HTTP/2 acceptance and request ID.
- Verification: Local protocol tests; hosted verification pending.
- Optional module: `github.com/goforj/notification/driver/apns`. See [installation](../README.md#installation).
- [Provider protocol reference](https://developer.apple.com/documentation/usernotifications/sending-notification-requests-to-apns).

## Apprise API

[Package source](../driver/apprise)

- Configuration: Stateful server configuration key and optional credentials/tag.
- Accepted when: Non-streaming relay acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://appriseit.com/api/usage/).

## Bark

[Package source](../driver/bark)

- Configuration: Server, device key and optional group.
- Accepted when: Native response code.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://github.com/Finb/bark-server).

## Better Stack

[Package source](../driver/betterstack)

- Configuration: API token, requester and escalation choices; Create returns a native incident ID.
- Accepted when: Native incident ID; actions preserve ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://betterstack.com/docs/uptime/api/create-a-new-incident/).

## ClickSend

[Package source](../driver/clicksend)

- Configuration: Username/API key, recipient and optional sender.
- Accepted when: Native envelope and per-message result.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developers.clicksend.com/docs/).

## DingTalk

[Package source](../driver/dingtalk)

- Configuration: Webhook and optional signing secret.
- Accepted when: Native errcode; HMAC signature at send time.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://open.dingtalk.com/document/robots/custom-robot-access).

## Discord

[Package source](../driver/discord)

- Configuration: Webhook URL and optional thread.
- Accepted when: Native message acknowledgement; mentions disabled.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.discord.com/developers/resources/webhook#execute-webhook).

## Expo

[Package source](../driver/expo)

- Configuration: Expo push token and optional access token.
- Accepted when: Native push ticket; not a delivery receipt.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.expo.dev/push-notifications/sending-notifications/).

## Feishu

[Package source](../driver/feishu)

- Configuration: V2 bot webhook and optional signing secret.
- Accepted when: Native code; timestamped HMAC when configured.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot).

## Firebase Cloud Messaging

[Package source](../driver/fcm)

- Configuration: Project, device or topic and borrowed refreshing OAuth token function.
- Accepted when: Project-scoped native message name.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://firebase.google.com/docs/cloud-messaging/send/v1-api).

## FireHydrant

[Package source](../driver/firehydrant)

- Configuration: Signals webhook and stable key; OPEN and CLOSED only.
- Accepted when: Native Signals acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.firehydrant.com/docs/generic-webhook-event-source).

## GoAlert

[Package source](../driver/goalert)

- Configuration: Server/integration token and stable key; trigger and close only.
- Accepted when: Generic Incoming acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://github.com/target/goalert).

## Google Chat

[Package source](../driver/googlechat)

- Configuration: Space webhook URL.
- Accepted when: Native message resource acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developers.google.com/workspace/chat/quickstart/webhooks).

## Gotify

[Package source](../driver/gotify)

- Configuration: Server, application token and integer priority.
- Accepted when: Native message identity and content acknowledgement.
- Verification: Real Gotify 3.1.1 server verified.
- [Provider protocol reference](https://gotify.net/docs/pushmsg).

## Home Assistant

[Package source](../driver/homeassistant)

- Configuration: Server, long-lived access token and notify service.
- Accepted when: Notify service response.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://contributors.home-assistant.io/core/internals/rest-api/).

## IFTTT

[Package source](../driver/ifttt)

- Configuration: Webhooks key and event; title/body map to value1/value2.
- Accepted when: Event-trigger acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://ifttt.com/maker_webhooks).

## ilert

[Package source](../driver/ilert)

- Configuration: Integration key and stable alert key; ALERT, ACCEPT and RESOLVE.
- Accepted when: HTTP 202 event acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.ilert.com/developer-docs/rest-api/api-reference/events).

## incident.io

[Package source](../driver/incidentio)

- Configuration: HTTP alert source, token and stable key; firing and resolve only.
- Accepted when: HTTP 202 with native success and deduplication key.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.incident.io/openapi/tags/alert-events-v2.json).

## Jira Service Management

[Package source](../driver/jsm)

- Configuration: Cloud ID, email and API token.
- Accepted when: Asynchronous request acceptance; stable alias lookup for actions.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/api-group-alerts/).

## Join

[Package source](../driver/join)

- Configuration: API key and device ID.
- Accepted when: Native success flag.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://joaoapps.com/join/api/).

## LINE

[Package source](../driver/line)

- Configuration: Channel access token and recipient ID.
- Accepted when: Messaging API push acknowledgement; LINE Notify excluded.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developers.line.biz/en/reference/messaging-api/).

## Log

[Package source](../driver/log)

- Configuration: Borrowed slog logger.
- Accepted when: Local capture.
- Verification: Local capture tests.

## Mail

[Package source](../driver/mail)

- Configuration: Borrowed Mailer and recipients.
- Accepted when: Mail acceptance.
- Verification: Mail composition tests; hosted verification pending.

## Matrix

[Package source](../driver/matrix)

- Configuration: Homeserver, access token and existing unencrypted room ID.
- Accepted when: Event ID; fresh transaction per Send.
- Verification: Real Synapse 1.164.2 server verified.
- [Provider protocol reference](https://spec.matrix.org/v1.16/client-server-api/).

## Mattermost

[Package source](../driver/mattermost)

- Configuration: Incoming webhook URL; mentions neutralized.
- Accepted when: Native ok acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.mattermost.com/developers/integrate/webhooks/incoming).

## Nextcloud Talk

[Package source](../driver/nextcloudtalk)

- Configuration: Server, user/app password and existing room token.
- Accepted when: OCS status and message ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://github.com/nextcloud/spreed/blob/main/docs/chat.md).

## ntfy

[Package source](../driver/ntfy)

- Configuration: Topic, optional authentication and integer priority.
- Accepted when: Native message identity and topic acknowledgement.
- Verification: Real ntfy 2.37.2 server verified.
- [Provider protocol reference](https://docs.ntfy.sh/publish/).

## OneSignal

[Package source](../driver/onesignal)

- Configuration: Application API key and one subscription or external user identity.
- Accepted when: Native message ID and recipient checks.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://documentation.onesignal.com/reference/create-message).

## Opsgenie

[Package source](../driver/opsgenie)

- Configuration: Legacy integration key and incident alias.
- Accepted when: Asynchronous request acceptance; legacy migration support.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.opsgenie.com/docs/alert-api).

## PagerDuty

[Package source](../driver/pagerduty)

- Configuration: Routing key and explicit incident key.
- Accepted when: Events v2 acknowledgement; trigger, acknowledge, resolve.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.pagerduty.com/developer/send-alert-event).

## PagerTree

[Package source](../driver/pagertree)

- Configuration: Incoming integration URL and stable key; full incident lifecycle.
- Accepted when: Native integration acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://pagertree.com/docs/integration-guides).

## Plivo

[Package source](../driver/plivo)

- Configuration: Account ID/token and E.164 sender/recipient.
- Accepted when: Native request and message UUID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.plivo.com/docs/messaging/api/message/send-a-message).

## Pushbullet

[Package source](../driver/pushbullet)

- Configuration: Access token and optional device, email or channel.
- Accepted when: Native push ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.pushbullet.com/).

## Pushover

[Package source](../driver/pushover)

- Configuration: Application token, user/group, optional device; priorities -2 through 1.
- Accepted when: Native status and request ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://pushover.net/api).

## Rocket.Chat

[Package source](../driver/rocketchat)

- Configuration: Incoming integration URL; mentions neutralized.
- Accepted when: Native success flag.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.rocket.chat/docs/integrations).

## Signal relay

[Package source](../driver/signal)

- Configuration: Application-owned paired signal-cli REST relay, sender and recipient.
- Accepted when: Native send timestamp.
- Verification: Local protocol tests; paired-session verification pending.
- [Provider protocol reference](https://github.com/bbernhard/signal-cli-rest-api).

## SIGNL4

[Package source](../driver/signl4)

- Configuration: Webhook and stable key; trigger, acknowledge and resolve.
- Accepted when: Native webhook acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.signl4.com/integrations/webhook/).

## Slack

[Package source](../driver/slack)

- Configuration: Incoming webhook URL.
- Accepted when: HTTP 2xx and ok acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/).

## Teams

[Package source](../driver/teams)

- Configuration: Workflows webhook URL.
- Accepted when: Adaptive Card submitted; HTTP 2xx acceptance.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using).

## Telegram

[Package source](../driver/telegram)

- Configuration: Bot token, chat and optional topic.
- Accepted when: Native message identity; JSON retry advice retained.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://core.telegram.org/bots/api#sendmessage).

## Telnyx

[Package source](../driver/telnyx)

- Configuration: API token, E.164 sender/recipient and optional messaging profile.
- Accepted when: Recipient-correlated message status.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developers.telnyx.com/api-reference/messages/send-a-message).

## Twilio

[Package source](../driver/twilio)

- Configuration: Account, token, E.164 recipient and sender or service.
- Accepted when: Native Message SID and accepted status.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://www.twilio.com/docs/messaging/api/message-resource).

## Vonage

[Package source](../driver/vonage)

- Configuration: API key/secret, sender and E.164 recipient; Unicode SMS.
- Accepted when: Per-message native status and ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developer.vonage.com/en/api/sms).

## WAHA

[Package source](../driver/waha)

- Configuration: Server, API key, existing session and chat.
- Accepted when: HTTP acceptance across engine response schemas.
- Verification: Local protocol tests; paired-session verification pending.
- [Provider protocol reference](https://waha.devlike.pro/docs/how-to/send-messages/).

## Web Push

[Package source](../driver/webpush)

- Configuration: Optional module; browser subscription, VAPID keys/contact and delivery headers.
- Accepted when: Encrypted push-service acceptance.
- Verification: Local protocol tests; hosted verification pending.
- Optional module: `github.com/goforj/notification/driver/webpush`. See [installation](../README.md#installation).
- [Provider protocol reference](https://web.dev/articles/push-notifications-web-push-protocol).

## Webhook

[Package source](../driver/webhook)

- Configuration: Trusted endpoint.
- Accepted when: HTTP 2xx acceptance.
- Verification: Local protocol tests; hosted verification pending.

## WeCom

[Package source](../driver/wecom)

- Configuration: Group robot webhook URL.
- Accepted when: Native errcode acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developer.work.weixin.qq.com/document/path/91770).

## Whapi

[Package source](../driver/whapi)

- Configuration: Channel token and recipient.
- Accepted when: Native sent acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://whapi.readme.io/reference/sendmessagetext).

## WhatsApp Cloud

[Package source](../driver/whatsapp)

- Configuration: Meta token, phone ID, explicit API version and recipient; approved templates are explicit.
- Accepted when: Native message and recipient acknowledgement.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://www.postman.com/meta/whatsapp-business-platform/request/o65u5m5/send-message-template-text).

## Zulip

[Package source](../driver/zulip)

- Configuration: Server, bot email/key, channel and topic.
- Accepted when: Success result and message ID.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://zulip.com/api/send-message).

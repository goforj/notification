# Driver configuration and delivery contracts

Generated from the driver catalog by `go -C docs run ./readme`. See [portable delivery contracts](contracts.md) and [executed integration evidence](integration-results.json). Provider acceptance never proves recipient delivery.

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

## Mail

[Package source](../driver/mail)

- Configuration: Borrowed Mailer and recipients.
- Accepted when: Mail acceptance.
- Verification: Mail composition tests; hosted verification pending.

## Feishu

[Package source](../driver/feishu)

- Configuration: V2 bot webhook and optional signing secret.
- Accepted when: Native code; timestamped HMAC when configured.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot).

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

## Jira Service Management

[Package source](../driver/jsm)

- Configuration: Cloud ID, email and API token.
- Accepted when: Asynchronous request acceptance; stable alias lookup for actions.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/api-group-alerts/).

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

## ntfy

[Package source](../driver/ntfy)

- Configuration: Topic, optional authentication and integer priority.
- Accepted when: Native message identity and topic acknowledgement.
- Verification: Real ntfy 2.37.2 server verified.
- [Provider protocol reference](https://docs.ntfy.sh/publish/).

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

## Twilio

[Package source](../driver/twilio)

- Configuration: Account, token, E.164 recipient and sender or service.
- Accepted when: Native Message SID and accepted status.
- Verification: Local protocol tests; hosted verification pending.
- [Provider protocol reference](https://www.twilio.com/docs/messaging/api/message-resource).

## WAHA

[Package source](../driver/waha)

- Configuration: Server, API key, existing session and chat.
- Accepted when: HTTP acceptance across engine response schemas.
- Verification: Local protocol tests; paired-session verification pending.
- [Provider protocol reference](https://waha.devlike.pro/docs/how-to/send-messages/).

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

// Package sns adapts an application-owned AWS SDK v2 SNS publisher without duplicating credential policy.
package sns

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/goforj/notification"
	"github.com/goforj/notification/internal/provider"
)

// Publisher is the borrowed AWS SDK v2 publish capability, including application credential and HTTP policy.
type Publisher interface {
	Publish(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// Config fixes exactly one SNS destination. Client is a required concurrency-safe wired dependency.
// Configure endpoints, refreshing credentials and HTTP transport on the AWS client before passing it here.
type Config struct {
	Client      Publisher
	TopicARN    string
	TargetARN   string
	PhoneNumber string
}

// Driver borrows a publisher and submits one SDK operation per valid send.
type Driver struct{ config Config }

// New validates the destination without retrieving credentials or contacting AWS.
func New(config Config) (*Driver, error) {
	count := 0
	for _, arn := range []string{config.TopicARN, config.TargetARN} {
		if arn != "" {
			count++
			if !provider.Token(arn) || !strings.HasPrefix(arn, "arn:") || !strings.Contains(arn, ":sns:") {
				return nil, notification.ErrInvalidRequest
			}
		}
	}
	if config.PhoneNumber != "" {
		count++
		if !provider.Phone(config.PhoneNumber) {
			return nil, notification.ErrInvalidRequest
		}
	}
	if count != 1 {
		return nil, notification.ErrInvalidRequest
	}
	return &Driver{config: config}, nil
}

// Send preserves native acceptance evidence and disables SDK retries for this operation without mutating the client.
func (d *Driver) Send(ctx context.Context, message notification.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := message.Validate(); err != nil {
		return err
	}
	if len(message.Title) > 100 {
		return notification.NewDeliveryError("sns", notification.Outcome{Status: notification.Failed}, notification.ErrRender)
	}
	text := provider.Text(message)
	if d.config.PhoneNumber != "" {
		var err error
		text, err = provider.SMSText(message)
		if err != nil {
			return notification.NewDeliveryError("sns", notification.Outcome{Status: notification.Failed}, err)
		}
	}
	input := &sns.PublishInput{Message: aws.String(text)}
	if message.Title != "" {
		input.Subject = aws.String(message.Title)
	}
	if d.config.TopicARN != "" {
		input.TopicArn = aws.String(d.config.TopicARN)
	}
	if d.config.TargetARN != "" {
		input.TargetArn = aws.String(d.config.TargetARN)
	}
	if d.config.PhoneNumber != "" {
		input.PhoneNumber = aws.String(d.config.PhoneNumber)
	}
	output, err := d.config.Client.Publish(ctx, input, func(options *sns.Options) { options.Retryer = aws.NopRetryer{} })
	if err != nil {
		status, cause := classify(ctx, err)
		return notification.NewDeliveryError("sns", notification.Outcome{Status: status}, cause)
	}
	if output == nil || output.MessageId == nil || *output.MessageId == "" {
		return notification.NewDeliveryError("sns", notification.Outcome{Status: notification.Unknown}, notification.ErrOutcomeUnknown)
	}
	return nil
}

// classify strips SDK diagnostics while preserving cancellation and native API rejection categories.
func classify(ctx context.Context, err error) (notification.Status, error) {
	if ctx.Err() != nil {
		return notification.Unknown, ctx.Err()
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, cause) {
			return notification.Unknown, cause
		}
	}
	var api smithy.APIError
	if errors.As(err, &api) {
		switch api.ErrorCode() {
		case "AuthorizationError", "AuthorizationErrorException", "InvalidSecurity", "InvalidClientTokenId", "UnrecognizedClientException":
			return notification.Failed, notification.ErrUnauthorized
		case "Throttled", "ThrottlingException", "TooManyRequestsException":
			return notification.Failed, notification.ErrThrottled
		}
		switch api.ErrorFault() {
		case smithy.FaultClient:
			return notification.Failed, notification.ErrPermanent
		case smithy.FaultServer:
			return notification.Failed, notification.ErrRetryable
		}
	}
	return notification.Unknown, notification.ErrOutcomeUnknown
}

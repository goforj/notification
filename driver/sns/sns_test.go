package sns

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/goforj/notification"
	"github.com/goforj/notification/notificationtest"
)

// publisherFunc scripts SDK states without weakening the native SDK wire test.
type publisherFunc func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error)

// Publish preserves the SDK's operation options in failure fixtures.
func (f publisherFunc) Publish(ctx context.Context, input *sns.PublishInput, options ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return f(ctx, input, options...)
}

// TestNativeSDK verifies SigV4, native form parameters, XML acknowledgements and one SDK attempt.
func TestNativeSDK(t *testing.T) {
	var reject atomic.Bool
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.ParseForm() != nil || r.Form.Get("Action") != "Publish" || r.Form.Get("TopicArn") != "arn:aws:sns:us-east-1:123456789012:alerts" || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("native SDK authentication or parameters")
		}
		w.Header().Set("Content-Type", "text/xml")
		if reject.Load() {
			w.WriteHeader(503)
			_, _ = io.WriteString(w, `<ErrorResponse><Error><Type>Receiver</Type><Code>InternalError</Code><Message>private response</Message></Error></ErrorResponse>`)
			return
		}
		_, _ = io.WriteString(w, `<PublishResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><PublishResult><MessageId>message-42</MessageId></PublishResult></PublishResponse>`)
	}))
	defer server.Close()
	client := sns.New(sns.Options{Region: "us-east-1", BaseEndpoint: aws.String(server.URL), HTTPClient: server.Client(), Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return aws.Credentials{AccessKeyID: "fixture", SecretAccessKey: "fixture-secret"}, nil
	}), Retryer: retry.NewStandard(func(options *retry.StandardOptions) { options.MaxAttempts = 3 })})
	driver, err := New(Config{Client: client, TopicARN: "arn:aws:sns:us-east-1:123456789012:alerts"})
	if err != nil {
		t.Fatal(err)
	}
	notificationtest.Sender(t, func() notification.Sender { return driver })
	before := calls.Load()
	reject.Store(true)
	err = driver.Send(nil, notification.Message{Body: "one operation"})
	if !errors.Is(err, notification.ErrRetryable) || strings.Contains(err.Error(), "private") || calls.Load() != before+1 {
		t.Fatal("SDK retry or error evidence", err, calls.Load()-before)
	}
	if client.Options().Retryer.MaxAttempts() != 3 {
		t.Fatal("borrowed SDK client was mutated")
	}
}

// TestValidationAndEvidence checks every destination family and contradictory or absent SDK evidence.
func TestValidationAndEvidence(t *testing.T) {
	published := 0
	client := publisherFunc(func(_ context.Context, input *sns.PublishInput, options ...func(*sns.Options)) (*sns.PublishOutput, error) {
		published++
		settings := sns.Options{}
		for _, option := range options {
			option(&settings)
		}
		if settings.Retryer.MaxAttempts() != 1 || input.Message == nil {
			t.Fatal("SDK operation policy")
		}
		return &sns.PublishOutput{MessageId: aws.String("message-42")}, nil
	})
	for _, config := range []Config{
		{Client: client, TopicARN: "arn:aws:sns:us-east-1:123456789012:alerts"},
		{Client: client, TargetARN: "arn:aws:sns:us-east-1:123456789012:endpoint/APNS/app/device"},
		{Client: client, PhoneNumber: "+12125550101"},
	} {
		driver, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := driver.Send(nil, notification.Message{Title: "Ticket", Body: "Changed"}); err != nil {
			t.Fatal(err)
		}
		if err := driver.Send(nil, notification.Message{Title: strings.Repeat("x", 101), Body: "body"}); !errors.Is(err, notification.ErrRender) {
			t.Fatal("SNS subject bound", err)
		}
		if config.PhoneNumber != "" {
			if err := driver.Send(nil, notification.Message{Body: strings.Repeat("x", 1601)}); !errors.Is(err, notification.ErrRender) {
				t.Fatal("SNS SMS bound", err)
			}
		}
	}
	if published != 3 {
		t.Fatal("invalid work reached SDK")
	}
	for _, config := range []Config{{}, {TopicARN: "invalid"}, {TargetARN: "arn:wrong"}, {PhoneNumber: "invalid"}, {TopicARN: "arn:aws:sns:r:a:t", PhoneNumber: "+12125550101"}} {
		if _, err := New(config); err == nil {
			t.Fatal("invalid destination accepted")
		}
	}
	for _, output := range []*sns.PublishOutput{nil, {}, {MessageId: aws.String("")}} {
		driver, _ := New(Config{TopicARN: "arn:aws:sns:r:a:t", Client: publisherFunc(func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return output, nil
		})})
		if err := driver.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, notification.ErrOutcomeUnknown) {
			t.Fatal("missing acknowledgement accepted", err)
		}
	}
	for _, entry := range []struct{ cause, want error }{
		{&smithy.GenericAPIError{Code: "AuthorizationError", Message: "private", Fault: smithy.FaultClient}, notification.ErrUnauthorized},
		{&smithy.GenericAPIError{Code: "ThrottlingException", Message: "private", Fault: smithy.FaultClient}, notification.ErrThrottled},
		{&smithy.GenericAPIError{Code: "InvalidParameter", Message: "private", Fault: smithy.FaultClient}, notification.ErrPermanent},
		{&smithy.GenericAPIError{Code: "InternalError", Message: "private", Fault: smithy.FaultServer}, notification.ErrRetryable},
		{&smithy.GenericAPIError{Code: "Other", Message: "private", Fault: smithy.FaultUnknown}, notification.ErrOutcomeUnknown},
		{context.Canceled, context.Canceled}, {context.DeadlineExceeded, context.DeadlineExceeded}, {errors.New("private"), notification.ErrOutcomeUnknown},
	} {
		driver, _ := New(Config{TopicARN: "arn:aws:sns:r:a:t", Client: publisherFunc(func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
			return nil, entry.cause
		})})
		if err := driver.Send(nil, notification.Message{Body: "body"}); !errors.Is(err, entry.want) || strings.Contains(err.Error(), "private") {
			t.Fatal("SDK error evidence", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	driver, _ := New(Config{TopicARN: "arn:aws:sns:r:a:t", Client: publisherFunc(func(context.Context, *sns.PublishInput, ...func(*sns.Options)) (*sns.PublishOutput, error) {
		cancel()
		return nil, errors.New("private")
	})})
	if err := driver.Send(ctx, notification.Message{Body: "body"}); !errors.Is(err, context.Canceled) || !errors.Is(err, notification.ErrOutcomeUnknown) {
		t.Fatal("SDK in-flight cancellation", err)
	}
}

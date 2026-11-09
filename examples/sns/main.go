// Command sns reuses an AWS SDK publisher against a local protocol fixture.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/goforj/notification"
	notificationsns "github.com/goforj/notification/driver/sns"
)

// main demonstrates caller-owned AWS credentials, region and HTTP connection policy.
func main() {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = io.WriteString(w, `<PublishResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><PublishResult><MessageId>message-42</MessageId></PublishResult></PublishResponse>`)
	}))
	defer service.Close()
	// snippet:start
	publisher := sns.New(sns.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(service.URL),
		HTTPClient:   service.Client(),
		Credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "fixture",
				SecretAccessKey: "local-fixture-secret",
			}, nil
		}),
	})
	driver, err := notificationsns.New(notificationsns.Config{
		Client:   publisher,
		TopicARN: "arn:aws:sns:us-east-1:123456789012:alerts",
	})
	if err != nil {
		panic(err)
	}
	err = driver.Send(context.Background(), notification.Message{
		Title: "Ticket updated",
		Body:  "Ticket T-42 has a new owner.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("SNS accepted through the borrowed AWS client")
	// SNS accepted through the borrowed AWS client
	// snippet:end
}

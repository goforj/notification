module github.com/goforj/notification/examples/sns

go 1.24.0

require (
	github.com/aws/aws-sdk-go-v2 v1.47.8
	github.com/aws/aws-sdk-go-v2/service/sns v1.47.10
	github.com/goforj/notification v0.1.0
	github.com/goforj/notification/driver/sns v0.1.0
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.11 // indirect
	github.com/aws/smithy-go v1.28.8 // indirect
)

replace github.com/goforj/notification => ../..

replace github.com/goforj/notification/driver/sns => ../../driver/sns

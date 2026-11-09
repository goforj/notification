module github.com/goforj/notification/driver/sns

go 1.24.0

require (
	github.com/aws/aws-sdk-go-v2 v1.47.8
	github.com/aws/aws-sdk-go-v2/service/sns v1.47.10
	github.com/aws/smithy-go v1.28.8
	github.com/goforj/notification v0.0.0-20261109213702-eafcce898c23
)

require (
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.11 // indirect
)

replace github.com/goforj/notification => ../..

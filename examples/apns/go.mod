module github.com/goforj/notification/examples/apns

go 1.24.0

require (
	github.com/goforj/notification v0.0.0-20261109213702-eafcce898c23
	github.com/goforj/notification/driver/apns v0.0.0-20261109213702-eafcce898c23
)

require github.com/golang-jwt/jwt/v5 v5.3.1 // indirect

replace github.com/goforj/notification => ../..

replace github.com/goforj/notification/driver/apns => ../../driver/apns

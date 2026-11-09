module github.com/goforj/notification/examples/apns

go 1.24.0

require (
	github.com/goforj/notification v0.0.0-20261109224454-a1a26f8d3182
	github.com/goforj/notification/driver/apns v0.0.0-20261109224454-a1a26f8d3182
)

require github.com/golang-jwt/jwt/v5 v5.3.1 // indirect

replace github.com/goforj/notification => ../..

replace github.com/goforj/notification/driver/apns => ../../driver/apns

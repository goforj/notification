module github.com/goforj/notification/examples/webpush

go 1.24.0

require (
	github.com/goforj/notification v0.0.0-20261109224454-a1a26f8d3182
	github.com/goforj/notification/driver/webpush v0.0.0-20261109224454-a1a26f8d3182
)

require (
	github.com/SherClockHolmes/webpush-go v1.4.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	golang.org/x/crypto v0.46.0 // indirect
)

replace github.com/goforj/notification => ../..

replace github.com/goforj/notification/driver/webpush => ../../driver/webpush

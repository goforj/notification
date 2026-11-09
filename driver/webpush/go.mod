module github.com/goforj/notification/driver/webpush

go 1.24.0

require (
	github.com/SherClockHolmes/webpush-go v1.4.0
	github.com/goforj/notification v0.0.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	golang.org/x/crypto v0.46.0
)

replace github.com/goforj/notification => ../..

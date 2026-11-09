module github.com/goforj/notification/examples/mail

go 1.24.0

require (
	github.com/goforj/mail v0.3.1
	github.com/goforj/notification v0.0.0
	github.com/goforj/notification/driver/mail v0.0.0
)

replace github.com/goforj/notification => ../..

replace github.com/goforj/notification/driver/mail => ../../driver/mail

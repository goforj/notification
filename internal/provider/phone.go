package provider

import "regexp"

// phoneNumber enforces E.164 before SMS or paired-device protocols receive a destination.
var phoneNumber = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)

// Phone accepts bounded E.164 numbers without provider-specific formatting guesses.
func Phone(value string) bool { return phoneNumber.MatchString(value) }

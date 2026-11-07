package common

import "strings"

// ResolveSMTPTemplate returns the configured template when it contains text;
// an empty template means that the built-in system default is still active.
func ResolveSMTPTemplate(value string, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func EffectiveSMTPVerificationSubject() string {
	return ResolveSMTPTemplate(SMTPVerificationSubject, DefaultSMTPVerificationSubject)
}

func EffectiveSMTPVerificationContent() string {
	return ResolveSMTPTemplate(SMTPVerificationContent, DefaultSMTPVerificationContent)
}

func EffectiveSMTPPasswordResetSubject() string {
	return ResolveSMTPTemplate(SMTPPasswordResetSubject, DefaultSMTPPasswordResetSubject)
}

func EffectiveSMTPPasswordResetContent() string {
	return ResolveSMTPTemplate(SMTPPasswordResetContent, DefaultSMTPPasswordResetContent)
}

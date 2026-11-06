package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSMTPTemplate(t *testing.T) {
	const defaultTemplate = "<p>system default</p>"

	assert.Equal(t, defaultTemplate, ResolveSMTPTemplate("", defaultTemplate))
	assert.Equal(t, defaultTemplate, ResolveSMTPTemplate(" \t\n", defaultTemplate))
	assert.Equal(t, "<p>custom</p>", ResolveSMTPTemplate("<p>custom</p>", defaultTemplate))
}

func TestEffectiveSMTPTemplatesUseConfiguredValues(t *testing.T) {
	original := [4]string{
		SMTPVerificationSubject,
		SMTPVerificationContent,
		SMTPPasswordResetSubject,
		SMTPPasswordResetContent,
	}
	t.Cleanup(func() {
		SMTPVerificationSubject = original[0]
		SMTPVerificationContent = original[1]
		SMTPPasswordResetSubject = original[2]
		SMTPPasswordResetContent = original[3]
	})

	SMTPVerificationSubject = "custom verification subject"
	SMTPVerificationContent = "custom verification content"
	SMTPPasswordResetSubject = "custom reset subject"
	SMTPPasswordResetContent = "custom reset content"

	require.Equal(t, "custom verification subject", EffectiveSMTPVerificationSubject())
	require.Equal(t, "custom verification content", EffectiveSMTPVerificationContent())
	require.Equal(t, "custom reset subject", EffectiveSMTPPasswordResetSubject())
	require.Equal(t, "custom reset content", EffectiveSMTPPasswordResetContent())
}

func TestEffectiveSMTPTemplatesFallBackToDefaults(t *testing.T) {
	original := [4]string{
		SMTPVerificationSubject,
		SMTPVerificationContent,
		SMTPPasswordResetSubject,
		SMTPPasswordResetContent,
	}
	t.Cleanup(func() {
		SMTPVerificationSubject = original[0]
		SMTPVerificationContent = original[1]
		SMTPPasswordResetSubject = original[2]
		SMTPPasswordResetContent = original[3]
	})

	SMTPVerificationSubject = "\n"
	SMTPVerificationContent = "\t"
	SMTPPasswordResetSubject = "  "
	SMTPPasswordResetContent = ""

	assert.Equal(t, DefaultSMTPVerificationSubject, EffectiveSMTPVerificationSubject())
	assert.Equal(t, DefaultSMTPVerificationContent, EffectiveSMTPVerificationContent())
	assert.Equal(t, DefaultSMTPPasswordResetSubject, EffectiveSMTPPasswordResetSubject())
	assert.Equal(t, DefaultSMTPPasswordResetContent, EffectiveSMTPPasswordResetContent())
}

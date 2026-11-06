package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionMapRestoresDefaultSMTPTemplatesWhenCleared(t *testing.T) {
	originalTemplates := [4]string{
		common.SMTPVerificationSubject,
		common.SMTPVerificationContent,
		common.SMTPPasswordResetSubject,
		common.SMTPPasswordResetContent,
	}
	keys := []string{
		"SMTPVerificationSubject",
		"SMTPVerificationContent",
		"SMTPPasswordResetSubject",
		"SMTPPasswordResetContent",
	}
	originalValues := make(map[string]string, len(keys))
	originalExists := make(map[string]bool, len(keys))
	common.OptionMapRWMutex.RLock()
	for _, key := range keys {
		originalValues[key], originalExists[key] = common.OptionMap[key]
	}
	common.OptionMapRWMutex.RUnlock()
	t.Cleanup(func() {
		common.SMTPVerificationSubject = originalTemplates[0]
		common.SMTPVerificationContent = originalTemplates[1]
		common.SMTPPasswordResetSubject = originalTemplates[2]
		common.SMTPPasswordResetContent = originalTemplates[3]

		common.OptionMapRWMutex.Lock()
		for _, key := range keys {
			if originalExists[key] {
				common.OptionMap[key] = originalValues[key]
			} else {
				delete(common.OptionMap, key)
			}
		}
		common.OptionMapRWMutex.Unlock()
	})

	customValues := []string{"custom subject", "custom content", "custom reset subject", "custom reset content"}
	defaultValues := []string{
		common.DefaultSMTPVerificationSubject,
		common.DefaultSMTPVerificationContent,
		common.DefaultSMTPPasswordResetSubject,
		common.DefaultSMTPPasswordResetContent,
	}
	for index, key := range keys {
		require.NoError(t, updateOptionMap(key, customValues[index]))
	}
	require.Equal(t, customValues[0], common.SMTPVerificationSubject)
	require.Equal(t, customValues[1], common.SMTPVerificationContent)
	require.Equal(t, customValues[2], common.SMTPPasswordResetSubject)
	require.Equal(t, customValues[3], common.SMTPPasswordResetContent)

	for _, key := range keys {
		require.NoError(t, updateOptionMap(key, ""))
	}
	require.Equal(t, defaultValues[0], common.SMTPVerificationSubject)
	require.Equal(t, defaultValues[1], common.SMTPVerificationContent)
	require.Equal(t, defaultValues[2], common.SMTPPasswordResetSubject)
	require.Equal(t, defaultValues[3], common.SMTPPasswordResetContent)

	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	for index, key := range keys {
		require.Equal(t, defaultValues[index], common.OptionMap[key])
	}
}

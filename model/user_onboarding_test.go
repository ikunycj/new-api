package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserOnboardingStatus(t *testing.T) {
	pendingVersion := OnboardingPendingVersion
	currentVersion := CurrentOnboardingVersion
	previousVersion := CurrentOnboardingVersion - 1

	tests := []struct {
		name            string
		version         *int
		required        bool
		expectedVersion *int
	}{
		{
			name:            "legacy user is not enrolled",
			version:         nil,
			required:        false,
			expectedVersion: nil,
		},
		{
			name:            "new user requires onboarding",
			version:         &pendingVersion,
			required:        true,
			expectedVersion: &pendingVersion,
		},
		{
			name:            "previous version requires current onboarding",
			version:         &previousVersion,
			required:        true,
			expectedVersion: &previousVersion,
		},
		{
			name:            "acknowledged user does not require onboarding",
			version:         &currentVersion,
			required:        false,
			expectedVersion: &currentVersion,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := (&User{OnboardingVersion: test.version}).OnboardingStatus()

			assert.Equal(t, test.required, status.Required)
			assert.Equal(t, test.expectedVersion, status.Version)
		})
	}
}

func TestCompleteUserOnboarding(t *testing.T) {
	truncateTables(t)

	newUser := func(username string, onboardingVersion *int) *User {
		user := &User{
			Username:          username,
			Password:          "password123",
			Role:              common.RoleCommonUser,
			Status:            common.UserStatusEnabled,
			AffCode:           username,
			OnboardingVersion: onboardingVersion,
		}
		require.NoError(t, DB.Create(user).Error)
		return user
	}

	pendingUser := newUser("onboarding-pending", NewUserOnboardingVersion())
	legacyUser := newUser("onboarding-legacy", nil)

	completedUser, err := CompleteUserOnboarding(pendingUser.Id)
	require.NoError(t, err)
	require.NotNil(t, completedUser.OnboardingVersion)
	assert.Equal(t, CurrentOnboardingVersion, *completedUser.OnboardingVersion)
	assert.False(t, completedUser.OnboardingStatus().Required)

	completedAgain, err := CompleteUserOnboarding(pendingUser.Id)
	require.NoError(t, err)
	require.NotNil(t, completedAgain.OnboardingVersion)
	assert.Equal(t, CurrentOnboardingVersion, *completedAgain.OnboardingVersion)

	unenrolledUser, err := CompleteUserOnboarding(legacyUser.Id)
	require.NoError(t, err)
	assert.Nil(t, unenrolledUser.OnboardingVersion)
	assert.False(t, unenrolledUser.OnboardingStatus().Required)
}

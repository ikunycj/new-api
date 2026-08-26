/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelConcurrencyAcquireAndRelease(t *testing.T) {
	channelID := 987654321
	require.True(t, TryAcquireChannelConcurrency(channelID, 2))
	require.True(t, TryAcquireChannelConcurrency(channelID, 2))
	assert.False(t, TryAcquireChannelConcurrency(channelID, 2))
	assert.Equal(t, 2, CurrentChannelConcurrency(channelID))

	ReleaseChannelConcurrency(channelID)
	assert.Equal(t, 1, CurrentChannelConcurrency(channelID))
	require.True(t, TryAcquireChannelConcurrency(channelID, 2))
	ReleaseChannelConcurrency(channelID)
	ReleaseChannelConcurrency(channelID)
	ReleaseChannelConcurrency(channelID)
	assert.Equal(t, 0, CurrentChannelConcurrency(channelID))
}

func TestChannelConcurrencyRejectsInvalidInputs(t *testing.T) {
	assert.False(t, TryAcquireChannelConcurrency(0, 100))
	assert.False(t, TryAcquireChannelConcurrency(1, 0))
	assert.Equal(t, 0, CurrentChannelConcurrency(1))
}

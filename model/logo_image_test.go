package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoImageOptionsReloadPostgres(t *testing.T) {
	setupPostgresAnalyticsTestDB(t, &Option{})
	preservePricingGroupConfiguration(t)
	previous := logoImageSnapshot.Load()
	t.Cleanup(func() { logoImageSnapshot.Store(previous) })

	content := []byte("logo image bytes")
	require.NoError(t, AddLogoImage("brand.png", content))
	logoImageSnapshot.Store(nil)
	loadOptionsFromDatabase()
	image, found := GetLogoImage("brand.png")
	require.True(t, found)
	assert.Equal(t, content, image.Content)
	common.OptionMapRWMutex.RLock()
	_, exposed := common.OptionMap[logoImagesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	assert.False(t, exposed)

	require.ErrorContains(t, UpdateOption(logoImagesOptionKey, "{}"), "徽标图片管理接口")
	require.ErrorContains(t, UpdateOptionsBulk(map[string]string{logoImagesOptionKey: "{}"}), "徽标图片管理接口")
	loadOptionsFromDatabase()
	assert.Equal(t, []string{"brand.png"}, ListLogoImageNames())

	// Simulate another node persisting an empty library before this node syncs.
	require.NoError(t, DB.Model(&Option{Key: logoImagesOptionKey}).Update("value", "{}").Error)
	loadOptionsFromDatabase()
	assert.Empty(t, ListLogoImageNames())
}

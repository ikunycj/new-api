package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func signAccountUsageCredential(token *model.Token) string {
	payload := fmt.Sprintf("%s%d.%d.%s", common.AccountUsageCredentialPrefix, token.Id, token.UserId, token.Key)
	return common.GenerateHMAC(payload)
}

func AuthorizeAccountUsage(c *gin.Context) {
	tokenID, err := strconv.Atoi(c.Param("id"))
	if err != nil || tokenID <= 0 {
		accountUsageError(c, http.StatusBadRequest, i18n.MsgInvalidParams)
		return
	}
	userID := c.GetInt("id")
	user, err := model.GetUserById(userID, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		accountUsageAuthError(c, err, http.StatusUnauthorized)
		return
	}
	token, err := model.GetTokenByIds(tokenID, userID)
	if err != nil || token.Status == common.TokenStatusDisabled {
		accountUsageAuthError(c, err, http.StatusNotFound)
		return
	}
	credential := fmt.Sprintf("%s%d.%s", common.AccountUsageCredentialPrefix, token.Id, signAccountUsageCredential(token))
	common.ApiSuccess(c, gin.H{"access_token": credential, "user_id": userID})
}

func GetAccountUsage(c *gin.Context) {
	userID, err := strconv.Atoi(c.GetHeader("New-Api-User"))
	if err != nil || userID <= 0 {
		accountUsageAuthError(c, nil, http.StatusUnauthorized)
		return
	}
	scheme, credential, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || !strings.HasPrefix(credential, common.AccountUsageCredentialPrefix) {
		accountUsageAuthError(c, nil, http.StatusUnauthorized)
		return
	}
	encodedID, signature, ok := strings.Cut(strings.TrimPrefix(credential, common.AccountUsageCredentialPrefix), ".")
	tokenID, err := strconv.Atoi(encodedID)
	if !ok || err != nil || tokenID <= 0 || len(signature) != 2*sha256.Size {
		accountUsageAuthError(c, nil, http.StatusUnauthorized)
		return
	}
	token, err := model.GetTokenByIds(tokenID, userID)
	if err != nil || token.Status == common.TokenStatusDisabled || !hmac.Equal([]byte(signature), []byte(signAccountUsageCredential(token))) {
		accountUsageAuthError(c, err, http.StatusUnauthorized)
		return
	}
	user, err := model.GetUserById(userID, false)
	if err != nil || user.Status != common.UserStatusEnabled {
		accountUsageAuthError(c, err, http.StatusUnauthorized)
		return
	}

	unit, multiplier := accountUsageDisplay()
	common.ApiSuccess(c, gin.H{
		"remaining": float64(user.Quota) * multiplier,
		"used":      float64(user.UsedQuota) * multiplier,
		"unit":      unit,
	})
}

func accountUsageDisplay() (string, float64) {
	unit := operation_setting.GetQuotaDisplayType()
	switch unit {
	case operation_setting.QuotaDisplayTypeTokens:
		return "Tokens", 1
	case operation_setting.QuotaDisplayTypeCustom:
		unit = operation_setting.GetCurrencySymbol()
	case operation_setting.QuotaDisplayTypeCNY:
	default:
		unit = operation_setting.QuotaDisplayTypeUSD
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	return unit, rate / common.QuotaPerUnit
}

func accountUsageAuthError(c *gin.Context, err error, status int) {
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		common.SysError("account usage authorization database failure: " + err.Error())
		accountUsageError(c, http.StatusInternalServerError, i18n.MsgDatabaseError)
		return
	}
	accountUsageError(c, status, i18n.MsgAuthAccessTokenInvalid)
}

func accountUsageError(c *gin.Context, status int, key string) {
	c.JSON(status, gin.H{"success": false, "message": common.TranslateMessage(c, key)})
}

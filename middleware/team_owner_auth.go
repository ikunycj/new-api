package middleware

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// TeamOwnerAuth gates routes that a parent account uses to manage its children.
// It runs after UserAuth (caller identity is already in context) and rejects:
//   - sub-accounts (parent_id != 0)
//   - regular accounts that have no children
func TeamOwnerAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		callerId := c.GetInt("id")
		caller, err := model.GetUserById(callerId, false)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "获取用户信息失败"})
			c.Abort()
			return
		}
		if caller.ParentId != 0 {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "子账户无法使用团队管理接口"})
			c.Abort()
			return
		}
		hasChildren, err := model.HasChildren(callerId)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "获取团队信息失败"})
			c.Abort()
			return
		}
		if !hasChildren {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "当前账户没有子账户"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AssertChildOf verifies that targetUserId is a direct child of parentId.
// Returns false and writes a 403 response if the check fails.
func AssertChildOf(c *gin.Context, parentId, targetUserId int) bool {
	target, err := model.GetUserById(targetUserId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return false
	}
	if target.ParentId != parentId {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "该用户不属于你的团队"})
		return false
	}
	return true
}

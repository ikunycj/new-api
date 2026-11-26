package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AffiliateSubAgent struct {
	ChildUserID  int   `gorm:"primaryKey;autoIncrement:false"`
	ParentUserID int   `gorm:"index;not null"`
	RateBps      int64 `gorm:"not null"`
	EffectiveAt  int64 `gorm:"not null"`
}

type AffiliateSubAgentUpdate struct {
	UserID  int    `json:"user_id"`
	RateBps *int64 `json:"rate_bps"` // -1 removes the relationship.
}

type AffiliateSubAgentView struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	RateBps  int64  `json:"rate_bps"`
}

func GetAffiliateAdminSubAgents(parentUserID int) ([]AffiliateSubAgentView, error) {
	if parentUserID <= 0 {
		return nil, errors.New("代理人 ID 无效")
	}
	byParent, err := getAffiliateAdminSubAgentsByParent([]int{parentUserID})
	if err != nil {
		return nil, err
	}
	if items := byParent[parentUserID]; items != nil {
		return items, nil
	}
	return []AffiliateSubAgentView{}, nil
}

func getAffiliateAdminSubAgentsByParent(parentUserIDs []int) (map[int][]AffiliateSubAgentView, error) {
	if len(parentUserIDs) == 0 {
		return nil, nil
	}
	var rows []struct {
		ParentUserID int
		AffiliateSubAgentView
	}
	if err := DB.Table("affiliate_sub_agents AS association").
		Select("association.parent_user_id, association.child_user_id AS user_id, COALESCE(child.username, '') AS username, COALESCE(child.email, '') AS email, association.rate_bps").
		Joins("LEFT JOIN users AS child ON child.id = association.child_user_id").
		Where("association.parent_user_id IN ?", parentUserIDs).
		Order("association.child_user_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	byParent := make(map[int][]AffiliateSubAgentView)
	for _, row := range rows {
		byParent[row.ParentUserID] = append(byParent[row.ParentUserID], row.AffiliateSubAgentView)
	}
	return byParent, nil
}

func saveAffiliateSubAgentsWithTx(tx *gorm.DB, parentUserID int, adminUserID int, updates []AffiliateSubAgentUpdate) error {
	if len(updates) > 100 {
		return errors.New("一次最多修改 100 名子代理")
	}
	userIDs := make([]int, 0, len(updates)+1)
	userIDs = append(userIDs, parentUserID)
	seen := make(map[int]bool, len(updates))
	for _, update := range updates {
		if update.UserID <= 0 || update.UserID == parentUserID || seen[update.UserID] {
			return fmt.Errorf("子代理 %d 无效或重复", update.UserID)
		}
		if update.RateBps == nil || *update.RateBps < -1 {
			return fmt.Errorf("子代理 %d 的返现比例必须大于或等于 0%%", update.UserID)
		}
		seen[update.UserID] = true
		userIDs = append(userIDs, update.UserID)
	}

	// Serialize changes that share a parent or child so two concurrent saves
	// cannot give a child two parents or introduce another level.
	var users []User
	if err := lockForUpdate(tx).Select("id").Where("id IN ?", userIDs).Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
	active := make(map[int]bool, len(users))
	for _, user := range users {
		active[user.Id] = true
	}
	if !active[parentUserID] {
		return errors.New("代理人不存在")
	}

	for _, update := range updates {
		childID, rate := update.UserID, *update.RateBps
		if rate == -1 {
			if err := tx.Where("parent_user_id = ? AND child_user_id = ?", parentUserID, childID).
				Delete(&AffiliateSubAgent{}).Error; err != nil {
				return err
			}
			continue
		}
		if !active[childID] {
			return fmt.Errorf("子代理 %d 不存在", childID)
		}
		var association AffiliateSubAgent
		err := tx.Where("child_user_id = ?", childID).Take(&association).Error
		if err == nil {
			if association.ParentUserID != parentUserID {
				return fmt.Errorf("子代理 %d 已有上级，请先解除原关系", childID)
			}
			if err := tx.Model(&association).Update("rate_bps", rate).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var conflict int64
		if err := tx.Model(&AffiliateSubAgent{}).Where("child_user_id = ? OR parent_user_id = ?", parentUserID, childID).
			Count(&conflict).Error; err != nil {
			return err
		}
		if conflict > 0 {
			return fmt.Errorf("无法添加子代理 %d，子代理关系仅支持一层", childID)
		}
		association = AffiliateSubAgent{
			ChildUserID: childID, ParentUserID: parentUserID, RateBps: rate, EffectiveAt: common.GetTimestamp(),
		}
		if err := tx.Create(&association).Error; err != nil {
			return err
		}
		override := AffiliateUserOverride{UserID: childID, UpdatedBy: adminUserID}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true,
		}).Create(&override).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteAffiliateAgent(userID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		err := lockForUpdate(tx).Unscoped().Select("id").Where("id = ?", userID).Take(&user).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Where("child_user_id = ? OR parent_user_id = ?", userID, userID).
			Delete(&AffiliateSubAgent{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&AffiliateUserOverride{}).Error
	})
}

func affiliateSubAgentForTopUp(tx *gorm.DB, childUserID int, completedAt int64) (*AffiliateSubAgent, error) {
	var association AffiliateSubAgent
	err := tx.Table("affiliate_sub_agents AS association").Select("association.*").
		Joins("JOIN users AS parent ON parent.id = association.parent_user_id AND parent.deleted_at IS NULL").
		Where("association.child_user_id = ? AND association.rate_bps > 0 AND association.effective_at <= ? AND parent.status = ?",
			childUserID, completedAt, common.UserStatusEnabled).
		Take(&association).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &association, nil
}

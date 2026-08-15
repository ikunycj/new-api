package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type ChannelFailoverBinding struct {
	ChannelId     int     `json:"channel_id"`
	ChannelName   string  `json:"channel_name"`
	BaseURL       *string `json:"base_url"`
	Status        int     `json:"status"`
	ClusterId     int     `json:"cluster_id"`
	ClusterPoolId int     `json:"cluster_pool_id"`
}

type ChannelFailoverBindingUpdate struct {
	ChannelId     int `json:"channel_id"`
	ClusterId     int `json:"cluster_id"`
	ClusterPoolId int `json:"cluster_pool_id"`
}

type ChannelFailoverBindingsUpdate struct {
	Bindings []ChannelFailoverBindingUpdate `json:"bindings"`
}

func GetChannelFailoverBindings() ([]ChannelFailoverBinding, error) {
	bindings := make([]ChannelFailoverBinding, 0)
	err := DB.Model(&Channel{}).
		Select("id AS channel_id", "name AS channel_name", "base_url", "status", "cluster_id", "cluster_pool_id").
		Order("id ASC").
		Find(&bindings).Error
	return bindings, err
}

func SaveChannelFailoverBindings(bindings []ChannelFailoverBindingUpdate) error {
	if DB == nil {
		return errors.New("database is not initialized")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		channelIDs := make([]int, 0, len(bindings))
		clusterIDs := make([]int, 0, len(bindings))
		poolIDs := make([]int, 0, len(bindings))
		seenChannels := make(map[int]struct{}, len(bindings))
		for _, binding := range bindings {
			if binding.ChannelId <= 0 {
				return errors.New("channel_id must be positive")
			}
			if _, exists := seenChannels[binding.ChannelId]; exists {
				return fmt.Errorf("channel %d is duplicated", binding.ChannelId)
			}
			seenChannels[binding.ChannelId] = struct{}{}
			channelIDs = append(channelIDs, binding.ChannelId)

			unbound := binding.ClusterId == 0 && binding.ClusterPoolId == 0
			bound := binding.ClusterId > 0 && binding.ClusterPoolId > 0
			if !unbound && !bound {
				return fmt.Errorf("channel %d must set both cluster_id and cluster_pool_id", binding.ChannelId)
			}
			if bound {
				clusterIDs = append(clusterIDs, binding.ClusterId)
				poolIDs = append(poolIDs, binding.ClusterPoolId)
			}
		}

		if len(channelIDs) > 0 {
			var count int64
			if err := tx.Model(&Channel{}).Where("id IN ?", channelIDs).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(channelIDs)) {
				return errors.New("one or more channels do not exist")
			}
		}

		clusters := make(map[int]Cluster, len(clusterIDs))
		if len(clusterIDs) > 0 {
			var storedClusters []Cluster
			if err := tx.Where("id IN ? AND archived = ?", clusterIDs, false).Find(&storedClusters).Error; err != nil {
				return err
			}
			for _, cluster := range storedClusters {
				clusters[cluster.Id] = cluster
			}
		}

		pools := make(map[int]ClusterPool, len(poolIDs))
		if len(poolIDs) > 0 {
			var storedPools []ClusterPool
			if err := tx.Where("id IN ?", poolIDs).Find(&storedPools).Error; err != nil {
				return err
			}
			for _, pool := range storedPools {
				pools[pool.Id] = pool
			}
		}

		for _, binding := range bindings {
			if binding.ClusterId > 0 {
				if _, exists := clusters[binding.ClusterId]; !exists {
					return fmt.Errorf("cluster %d does not exist or is archived", binding.ClusterId)
				}
				pool, exists := pools[binding.ClusterPoolId]
				if !exists {
					return fmt.Errorf("cluster pool %d does not exist", binding.ClusterPoolId)
				}
				if pool.ClusterId != binding.ClusterId {
					return fmt.Errorf("cluster pool %d does not belong to cluster %d", binding.ClusterPoolId, binding.ClusterId)
				}
			}
			if err := tx.Model(&Channel{}).Where("id = ?", binding.ChannelId).Updates(map[string]any{
				"cluster_id":      binding.ClusterId,
				"cluster_pool_id": binding.ClusterPoolId,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

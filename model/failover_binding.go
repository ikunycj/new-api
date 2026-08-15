package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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

type ClusterRouteConfig struct {
	ChannelId  int     `json:"channel_id"`
	PoolTier   int     `json:"pool_tier"`
	PoolName   string  `json:"pool_name"`
	RouteOrder int     `json:"route_order"`
	Weight     uint    `json:"weight"`
	CostFactor float64 `json:"cost_factor"`
}

type ClusterConfiguration struct {
	Id                      int                  `json:"id"`
	Name                    string               `json:"name"`
	Type                    string               `json:"type"`
	Status                  int                  `json:"status"`
	BillingGroup            string               `json:"billing_group"`
	BillingGroupDescription string               `json:"billing_group_description"`
	BillingGroupRatio       float64              `json:"billing_group_ratio"`
	PolicyId                int                  `json:"policy_id"`
	FailoverPriority        int                  `json:"failover_priority"`
	Remark                  string               `json:"remark"`
	Routes                  []ClusterRouteConfig `json:"routes"`
}

type ClusterChannelOption struct {
	Id         int     `json:"id"`
	Name       string  `json:"name"`
	BaseURL    *string `json:"base_url"`
	Status     int     `json:"status"`
	Type       int     `json:"type"`
	Group      string  `json:"group"`
	ClusterId  int     `json:"cluster_id"`
	IsMultiKey bool    `json:"is_multi_key"`
}

type BillingGroupOption struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Ratio       float64 `json:"ratio"`
}

type ClusterConfigurationSnapshot struct {
	Clusters      []ClusterConfiguration `json:"clusters"`
	Channels      []ClusterChannelOption `json:"channels"`
	BillingGroups []BillingGroupOption   `json:"billing_groups"`
	Policies      []FailoverPolicy       `json:"policies"`
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
			var storedChannels []Channel
			if err := tx.Select("id", "channel_info").Where("id IN ?", channelIDs).Find(&storedChannels).Error; err != nil {
				return err
			}
			if len(storedChannels) != len(channelIDs) {
				return errors.New("one or more channels do not exist")
			}
			for _, channel := range storedChannels {
				if channel.ChannelInfo.IsMultiKey {
					return fmt.Errorf("channel %d uses multiple keys and cannot be assigned to a cluster", channel.Id)
				}
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
				cluster, exists := clusters[binding.ClusterId]
				if !exists {
					return fmt.Errorf("cluster %d does not exist or is archived", binding.ClusterId)
				}
				if strings.TrimSpace(cluster.BillingGroup) != "" {
					return fmt.Errorf("cluster %d is managed by cluster configuration", binding.ClusterId)
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

func GetClusterConfigurationSnapshot() (*ClusterConfigurationSnapshot, error) {
	snapshot := &ClusterConfigurationSnapshot{
		Clusters:      make([]ClusterConfiguration, 0),
		Channels:      make([]ClusterChannelOption, 0),
		BillingGroups: make([]BillingGroupOption, 0),
		Policies:      make([]FailoverPolicy, 0),
	}
	var clusters []Cluster
	if err := DB.Where("archived = ?", false).Order("failover_priority DESC, id ASC").Find(&clusters).Error; err != nil {
		return nil, err
	}
	var pools []ClusterPool
	if err := DB.Order("cluster_id ASC, tier ASC").Find(&pools).Error; err != nil {
		return nil, err
	}
	poolByID := make(map[int]ClusterPool, len(pools))
	for _, pool := range pools {
		poolByID[pool.Id] = pool
	}
	var channels []Channel
	if err := DB.Select("id", "name", "base_url", "status", "type", "group", "cluster_id", "cluster_pool_id", "channel_info", "priority", "weight").Order("id ASC").Find(&channels).Error; err != nil {
		return nil, err
	}
	channelRoutes := make(map[int][]ClusterRouteConfig)
	for _, channel := range channels {
		snapshot.Channels = append(snapshot.Channels, ClusterChannelOption{
			Id: channel.Id, Name: channel.Name, BaseURL: channel.BaseURL, Status: channel.Status,
			Type: channel.Type, Group: channel.Group, ClusterId: channel.ClusterId, IsMultiKey: channel.ChannelInfo.IsMultiKey,
		})
		pool, ok := poolByID[channel.ClusterPoolId]
		if channel.ClusterId <= 0 || !ok {
			continue
		}
		channelRoutes[channel.ClusterId] = append(channelRoutes[channel.ClusterId], ClusterRouteConfig{
			ChannelId: channel.Id, PoolTier: pool.Tier, PoolName: pool.Name,
			RouteOrder: pool.Tier, Weight: uint(channel.GetWeight()), CostFactor: pool.CostFactor,
		})
	}
	ratios := ratio_setting.GetGroupRatioCopy()
	descriptions := setting.GetUserUsableGroupsCopy()
	for _, cluster := range clusters {
		routes := channelRoutes[cluster.Id]
		sort.SliceStable(routes, func(i, j int) bool { return routes[i].RouteOrder < routes[j].RouteOrder })
		ratio := ratios[cluster.BillingGroup]
		if ratio == 0 {
			if _, exists := ratios[cluster.BillingGroup]; !exists {
				ratio = 1
			}
		}
		snapshot.Clusters = append(snapshot.Clusters, ClusterConfiguration{
			Id: cluster.Id, Name: cluster.Name, Type: cluster.Type, Status: cluster.Status,
			BillingGroup: cluster.BillingGroup, BillingGroupDescription: descriptions[cluster.BillingGroup],
			BillingGroupRatio: ratio, PolicyId: cluster.PolicyId, FailoverPriority: cluster.FailoverPriority,
			Remark: cluster.Remark, Routes: routes,
		})
	}
	groupNames := make([]string, 0, len(ratios))
	for name := range ratios {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	for _, name := range groupNames {
		snapshot.BillingGroups = append(snapshot.BillingGroups, BillingGroupOption{
			Name: name, Description: descriptions[name], Ratio: ratios[name],
		})
	}
	if err := DB.Where("enabled = ?", true).Order("id ASC").Find(&snapshot.Policies).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}

func SaveClusterConfiguration(config *ClusterConfiguration) error {
	if config == nil {
		return errors.New("cluster configuration is required")
	}
	config.Name = strings.TrimSpace(config.Name)
	config.Type = strings.ToLower(strings.TrimSpace(config.Type))
	config.BillingGroup = strings.TrimSpace(config.BillingGroup)
	config.BillingGroupDescription = strings.TrimSpace(config.BillingGroupDescription)
	config.Remark = strings.TrimSpace(config.Remark)
	if config.Name == "" || config.Type == "" || config.BillingGroup == "" {
		return errors.New("cluster name, type, and billing group are required")
	}
	if strings.Contains(config.BillingGroup, ",") || len(config.BillingGroup) > 64 {
		return errors.New("billing group must be at most 64 characters and cannot contain commas")
	}
	if config.Status != ClusterStatusDisabled && config.Status != ClusterStatusEnabled {
		return errors.New("cluster status is invalid")
	}
	if config.BillingGroupRatio < 0 {
		return errors.New("billing group ratio cannot be negative")
	}
	if len(config.Routes) < 3 || len(config.Routes) > 4 {
		return errors.New("a cluster must have three or four channel routes")
	}
	sort.SliceStable(config.Routes, func(i, j int) bool { return config.Routes[i].RouteOrder < config.Routes[j].RouteOrder })
	seenChannels := make(map[int]struct{}, len(config.Routes))
	for index := range config.Routes {
		route := &config.Routes[index]
		expectedOrder := index + 1
		if route.ChannelId <= 0 || route.RouteOrder != expectedOrder || route.PoolTier != expectedOrder {
			return errors.New("route order and pool tier must be contiguous from P1")
		}
		if _, exists := seenChannels[route.ChannelId]; exists {
			return fmt.Errorf("channel %d is duplicated", route.ChannelId)
		}
		seenChannels[route.ChannelId] = struct{}{}
		route.PoolName = strings.TrimSpace(route.PoolName)
		if route.PoolName == "" {
			route.PoolName = defaultPoolName(route.PoolTier)
		}
		if route.CostFactor < 0 {
			return errors.New("pool cost factor cannot be negative")
		}
	}

	ratios := ratio_setting.GetGroupRatioCopy()
	descriptions := setting.GetUserUsableGroupsCopy()
	ratios[config.BillingGroup] = config.BillingGroupRatio
	if config.BillingGroupDescription == "" {
		config.BillingGroupDescription = config.Name + " billing group"
	}
	descriptions[config.BillingGroup] = config.BillingGroupDescription
	ratioJSON, err := common.Marshal(ratios)
	if err != nil {
		return err
	}
	descriptionJSON, err := common.Marshal(descriptions)
	if err != nil {
		return err
	}
	optionValues := map[string]string{
		"GroupRatio":       string(ratioJSON),
		"UserUsableGroups": string(descriptionJSON),
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		if config.PolicyId > 0 {
			var policy FailoverPolicy
			if err := tx.Where("id = ? AND enabled = ?", config.PolicyId, true).First(&policy).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("selected failover policy does not exist or is disabled")
				}
				return err
			}
			if policy.MaxPoolAttempts < len(config.Routes) {
				return fmt.Errorf("selected failover policy allows %d pools, but this cluster has %d routes", policy.MaxPoolAttempts, len(config.Routes))
			}
		}

		cluster := Cluster{Id: config.Id}
		oldBillingGroup := ""
		if config.Id > 0 {
			if err := tx.First(&cluster, config.Id).Error; err != nil {
				return err
			}
			oldBillingGroup = strings.TrimSpace(cluster.BillingGroup)
		}
		now := common.GetTimestamp()
		cluster.Name = config.Name
		cluster.Type = config.Type
		cluster.Status = config.Status
		cluster.BillingGroup = config.BillingGroup
		cluster.PolicyId = config.PolicyId
		cluster.FailoverPriority = config.FailoverPriority
		cluster.Remark = config.Remark
		cluster.Archived = false
		cluster.UpdatedTime = now
		if cluster.CreatedTime == 0 {
			cluster.CreatedTime = now
		}
		if err := tx.Save(&cluster).Error; err != nil {
			return err
		}
		config.Id = cluster.Id

		channelIDs := make([]int, 0, len(config.Routes))
		for _, route := range config.Routes {
			channelIDs = append(channelIDs, route.ChannelId)
		}
		var routeChannels []Channel
		if err := lockForUpdate(tx).Where("id IN ?", channelIDs).Find(&routeChannels).Error; err != nil {
			return err
		}
		if len(routeChannels) != len(channelIDs) {
			return errors.New("one or more channels do not exist")
		}
		channelByID := make(map[int]*Channel, len(routeChannels))
		for index := range routeChannels {
			channel := &routeChannels[index]
			if channel.ChannelInfo.IsMultiKey {
				return fmt.Errorf("channel %d uses multiple keys and cannot be assigned to a cluster", channel.Id)
			}
			if channel.ClusterId > 0 && channel.ClusterId != cluster.Id {
				return fmt.Errorf("channel %d already belongs to cluster %d", channel.Id, channel.ClusterId)
			}
			channelByID[channel.Id] = channel
		}

		poolByTier := make(map[int]*ClusterPool)
		var existingPools []ClusterPool
		if err := tx.Where("cluster_id = ?", cluster.Id).Find(&existingPools).Error; err != nil {
			return err
		}
		for index := range existingPools {
			poolByTier[existingPools[index].Tier] = &existingPools[index]
		}
		for _, route := range config.Routes {
			pool := poolByTier[route.PoolTier]
			if pool == nil {
				pool = &ClusterPool{ClusterId: cluster.Id, Tier: route.PoolTier, CreatedTime: now}
				poolByTier[route.PoolTier] = pool
			}
			pool.Name = route.PoolName
			pool.Status = config.Status
			pool.CostFactor = route.CostFactor
			pool.UpdatedTime = now
			if err := tx.Save(pool).Error; err != nil {
				return err
			}

			channel := channelByID[route.ChannelId]
			priority := int64((len(config.Routes) - route.RouteOrder + 1) * 100)
			channel.Group = config.BillingGroup
			channel.ClusterId = cluster.Id
			channel.ClusterPoolId = pool.Id
			channel.Priority = &priority
			channel.Weight = &route.Weight
			if err := tx.Model(channel).Select("group", "cluster_id", "cluster_pool_id", "priority", "weight").Updates(channel).Error; err != nil {
				return err
			}
			if err := channel.UpdateAbilities(tx); err != nil {
				return err
			}
		}

		var removedChannels []Channel
		if err := lockForUpdate(tx).Where("cluster_id = ? AND id NOT IN ?", cluster.Id, channelIDs).Find(&removedChannels).Error; err != nil {
			return err
		}
		for index := range removedChannels {
			channel := &removedChannels[index]
			channel.ClusterId = 0
			channel.ClusterPoolId = 0
			channel.Group = removeChannelGroup(channel.Group, oldBillingGroup)
			if err := tx.Model(channel).Select("group", "cluster_id", "cluster_pool_id").Updates(channel).Error; err != nil {
				return err
			}
			if err := channel.UpdateAbilities(tx); err != nil {
				return err
			}
		}

		activeTiers := make([]int, 0, len(config.Routes))
		for _, route := range config.Routes {
			activeTiers = append(activeTiers, route.PoolTier)
		}
		if err := tx.Model(&ClusterPool{}).Where("cluster_id = ? AND tier NOT IN ?", cluster.Id, activeTiers).Update("status", ClusterStatusDisabled).Error; err != nil {
			return err
		}
		return persistOptions(tx, optionValues)
	})
	if err != nil {
		return err
	}
	return applyOptionValues(optionValues)
}

func defaultPoolName(tier int) string {
	switch tier {
	case PoolTierFree:
		return "Free"
	case PoolTierPremium:
		return "Pro/Plus"
	case PoolTierFallback:
		return "Fallback"
	case PoolTierEmergency:
		return "Emergency"
	default:
		return fmt.Sprintf("P%d", tier)
	}
}

func removeChannelGroup(groups string, groupToRemove string) string {
	if groupToRemove == "" {
		return groups
	}
	result := make([]string, 0)
	for _, group := range strings.Split(groups, ",") {
		group = strings.TrimSpace(group)
		if group != "" && group != groupToRemove {
			result = append(result, group)
		}
	}
	if len(result) == 0 {
		return "default"
	}
	return strings.Join(result, ",")
}

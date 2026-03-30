package service

import (
	"context"
	"math/rand"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/packagechannel"
)

// PackageChannelService 处理套餐渠道的业务逻辑
type PackageChannelService struct {
	db *ent.Client
}

// NewPackageChannelService 创建 PackageChannelService
func NewPackageChannelService(db *ent.Client) *PackageChannelService {
	return &PackageChannelService{db: db}
}

// GetChannelsByGroup 获取套餐关联的所有渠道
func (s *PackageChannelService) GetChannelsByGroup(ctx context.Context, groupID int64) ([]*ent.PackageChannel, error) {
	return s.db.PackageChannel.Query().
		Where(packagechannel.GroupID(groupID)).
		WithAccount().
		All(ctx)
}

// GetEnabledChannels 获取套餐关联的启用渠道
func (s *PackageChannelService) GetEnabledChannels(ctx context.Context, groupID int64) ([]*ent.PackageChannel, error) {
	return s.db.PackageChannel.Query().
		Where(
			packagechannel.GroupID(groupID),
			packagechannel.IsEnabled(true),
		).
		WithAccount().
		All(ctx)
}

// AssignAccountToGroup 添加账号到套餐渠道
func (s *PackageChannelService) AssignAccountToGroup(ctx context.Context, groupID, accountID int64, weight int) (*ent.PackageChannel, error) {
	return s.db.PackageChannel.Create().
		SetGroupID(groupID).
		SetAccountID(accountID).
		SetWeight(weight).
		SetIsEnabled(true).
		Save(ctx)
}

// UpdateChannel 更新渠道配置
func (s *PackageChannelService) UpdateChannel(ctx context.Context, groupID, accountID int64, weight, maxUsers int, isEnabled bool) error {
	_, err := s.db.PackageChannel.Update().
		Where(
			packagechannel.GroupID(groupID),
			packagechannel.AccountID(accountID),
		).
		SetWeight(weight).
		SetMaxUsers(maxUsers).
		SetIsEnabled(isEnabled).
		Save(ctx)
	return err
}

// RemoveChannel 移除账号渠道
func (s *PackageChannelService) RemoveChannel(ctx context.Context, groupID, accountID int64) error {
	_, err := s.db.PackageChannel.Delete().
		Where(
			packagechannel.GroupID(groupID),
			packagechannel.AccountID(accountID),
		).
		Exec(ctx)
	return err
}

// SelectAccountByChannel 根据权重选择账号
func (s *PackageChannelService) SelectAccountByChannel(ctx context.Context, groupID int64) (*ent.Account, error) {
	channels, err := s.GetEnabledChannels(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return nil, nil
	}

	// 加权随机选择
	totalWeight := 0
	for _, ch := range channels {
		totalWeight += ch.Weight
	}

	if totalWeight == 0 {
		return nil, nil
	}

	randVal := rand.Intn(totalWeight)
	cumulative := 0
	for _, ch := range channels {
		cumulative += ch.Weight
		if randVal < cumulative {
			return ch.Edges.Account, nil
		}
	}

	return channels[0].Edges.Account, nil
}

// CheckChannelExists 检查渠道是否存在
func (s *PackageChannelService) CheckChannelExists(ctx context.Context, groupID, accountID int64) (bool, error) {
	count, err := s.db.PackageChannel.Query().
		Where(
			packagechannel.GroupID(groupID),
			packagechannel.AccountID(accountID),
		).
		Count(ctx)
	return count > 0, err
}

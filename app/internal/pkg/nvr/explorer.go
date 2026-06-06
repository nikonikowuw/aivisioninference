package nvr

import (
	"context"

	"github.com/niko-admin/niko-admin/internal/model"
)

type ChannelInfo struct {
	ChannelID   string
	ChannelName string
	AccessURL   string
	Status      string // online/offline
}

type Explorer interface {
	ListChannels(ctx context.Context, device *model.Device) ([]ChannelInfo, error)
}

// DefaultExplorer 默认实现（返回空列表）
type DefaultExplorer struct{}

func (e *DefaultExplorer) ListChannels(ctx context.Context, device *model.Device) ([]ChannelInfo, error) {
	return nil, nil
}

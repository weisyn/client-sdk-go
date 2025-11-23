package event

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/weisyn/client-sdk-go/client"
)

// Service Event 业务服务接口
type Service interface {
	// GetEvents 获取事件列表
	GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error)

	// SubscribeEvents 订阅事件
	SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error)
}

// eventService Event 服务实现
type eventService struct {
	client client.Client
}

// NewService 创建 Event 服务
func NewService(client client.Client) Service {
	return &eventService{
		client: client,
	}
}

// EventFilters 事件查询过滤器
type EventFilters struct {
	ResourceID *[32]byte
	EventName  *string
	Limit      int
	Offset     int
}

// EventInfo 事件信息
type EventInfo struct {
	EventName   string
	ResourceID  [32]byte
	Data        []byte
	TxID        string
	BlockHeight *uint64
}

// GetEvents 获取事件列表
func (s *eventService) GetEvents(ctx context.Context, filters *EventFilters) ([]*EventInfo, error) {
	// 使用 WESClient 实现（如果可用）
	// 否则直接调用底层 RPC
	req := make(map[string]interface{})

	if filters != nil {
		if filters.ResourceID != nil {
			req["resourceId"] = "0x" + hex.EncodeToString(filters.ResourceID[:])
		}
		if filters.EventName != nil {
			req["eventName"] = *filters.EventName
		}
		if filters.Limit > 0 {
			req["limit"] = filters.Limit
		}
		if filters.Offset > 0 {
			req["offset"] = filters.Offset
		}
	}

	raw, err := s.client.Call(ctx, "wes_getEvents", []interface{}{map[string]interface{}{"filters": req}})
	if err != nil {
		return nil, fmt.Errorf("get events failed: %w", err)
	}

	// 解码事件数组
	events, err := decodeEventArray(raw)
	if err != nil {
		return nil, fmt.Errorf("decode event array failed: %w", err)
	}

	return events, nil
}

// SubscribeEvents 订阅事件
func (s *eventService) SubscribeEvents(ctx context.Context, filters *EventFilters) (<-chan *EventInfo, error) {
	// 构建事件过滤器
	eventFilter := &client.EventFilter{}

	if filters != nil {
		if filters.ResourceID != nil {
			eventFilter.To = filters.ResourceID[:]
		}
		if filters.EventName != nil {
			eventFilter.Topics = []string{*filters.EventName}
		}
	}

	// 使用底层 Client.Subscribe
	eventChan, err := s.client.Subscribe(ctx, eventFilter)
	if err != nil {
		return nil, fmt.Errorf("subscribe events failed: %w", err)
	}

	// 转换 Event 为 EventInfo
	infoChan := make(chan *EventInfo, 10)
	go func() {
		defer close(infoChan)
		for event := range eventChan {
			info := &EventInfo{
				EventName: event.Topic,
				Data:      event.Data,
			}
			if filters != nil && filters.ResourceID != nil {
				copy(info.ResourceID[:], filters.ResourceID[:])
			}
			infoChan <- info
		}
	}()

	return infoChan, nil
}

// decodeEventArray 解码事件数组
func decodeEventArray(raw interface{}) ([]*EventInfo, error) {
	if raw == nil {
		return []*EventInfo{}, nil
	}

	// TODO: 实现事件数组解码逻辑
	// 这里需要根据实际的 RPC 返回格式实现
	return []*EventInfo{}, nil
}


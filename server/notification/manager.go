package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kefu-server/utils/logger"
)

type Manager struct {
	mu             sync.RWMutex
	configs        map[ChannelType]*ChannelConfig
	bindStateCache *BindStateCache
}

type BindStateCache struct {
	mu    sync.RWMutex
	items map[string]*BindState
}

type BindState struct {
	Status   string
	UserID   string
	AgentID  uint
	Channel  ChannelType
	Expires  time.Time
}

var globalManager *Manager
var once sync.Once

func GetManager() *Manager {
	once.Do(func() {
		globalManager = &Manager{
			configs:        make(map[ChannelType]*ChannelConfig),
			bindStateCache: &BindStateCache{items: make(map[string]*BindState)},
		}
		go globalManager.cleanExpiredStates()
	})
	return globalManager
}

func (m *Manager) cleanExpiredStates() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		m.bindStateCache.mu.Lock()
		now := time.Now()
		for key, state := range m.bindStateCache.items {
			if now.After(state.Expires) {
				delete(m.bindStateCache.items, key)
			}
		}
		m.bindStateCache.mu.Unlock()
	}
}

func (m *Manager) LoadConfigs(configData map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.configs = make(map[ChannelType]*ChannelConfig)

	for channelStr, cfg := range configData {
		channelType := ChannelType(channelStr)
		if cfgMap, ok := cfg.(map[string]interface{}); ok {
			enabled := false
			if e, ok := cfgMap["enabled"].(bool); ok {
				enabled = e
			}

			config := &ChannelConfig{
				Type:    channelType,
				Enabled: enabled,
				Config:  cfgMap,
			}

			if metadata, ok := cfgMap["metadata"].(map[string]string); ok {
				config.Metadata = metadata
			}

			m.configs[channelType] = config
		}
	}
}

func (m *Manager) GetConfig(channelType ChannelType) (*ChannelConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, exists := m.configs[channelType]
	return config, exists
}

func (m *Manager) GetAllConfigs() map[ChannelType]*ChannelConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[ChannelType]*ChannelConfig)
	for k, v := range m.configs {
		result[k] = v
	}
	return result
}

func (m *Manager) UpdateConfig(channelType ChannelType, config *ChannelConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[channelType] = config
}

func (m *Manager) IsEnabled(channelType ChannelType) bool {
	config, exists := m.GetConfig(channelType)
	if !exists {
		return false
	}
	return config.Enabled
}

func (m *Manager) Send(ctx context.Context, channelType ChannelType, to string, notification *Notification) error {
	notifier, exists := GetNotifier(channelType)
	if !exists {
		return fmt.Errorf("unsupported notification channel: %s", channelType)
	}

	config, exists := m.GetConfig(channelType)
	if !exists || !config.Enabled {
		return fmt.Errorf("channel %s is not configured or disabled", channelType)
	}

	return notifier.Send(ctx, to, notification)
}

func (m *Manager) SendToAll(ctx context.Context, to string, notification *Notification) map[ChannelType]error {
	errors := make(map[ChannelType]error)

	for channelType := range m.configs {
		if m.IsEnabled(channelType) {
			if err := m.Send(ctx, channelType, to, notification); err != nil {
				errors[channelType] = err
				logger.Errorf("send notification via %s failed: %v", channelType, err)
			}
		}
	}

	return errors
}

func (m *Manager) TestConnection(channelType ChannelType, config map[string]interface{}) error {
	notifier, exists := GetNotifier(channelType)
	if !exists {
		return fmt.Errorf("unsupported notification channel: %s", channelType)
	}

	return notifier.TestConnection(config)
}

func (m *Manager) GetBindURL(channelType ChannelType, userID uint, callbackURL string) (string, string, error) {
	notifier, exists := GetNotifier(channelType)
	if !exists {
		return "", "", fmt.Errorf("unsupported notification channel: %s", channelType)
	}

	config, exists := m.GetConfig(channelType)
	if !exists || !config.Enabled {
		return "", "", fmt.Errorf("channel %s is not configured or disabled", channelType)
	}

	return notifier.GetBindURL(config.Config, userID, callbackURL)
}

func (m *Manager) HandleCallback(channelType ChannelType, code, state string) (string, error) {
	notifier, exists := GetNotifier(channelType)
	if !exists {
		return "", fmt.Errorf("unsupported notification channel: %s", channelType)
	}

	config, exists := m.GetConfig(channelType)
	if !exists {
		return "", fmt.Errorf("channel %s is not configured", channelType)
	}

	ctx := context.Background()
	return notifier.HandleCallback(ctx, config.Config, code, state)
}

func (m *Manager) StoreBindState(state string, bindState *BindState) {
	m.bindStateCache.mu.Lock()
	defer m.bindStateCache.mu.Unlock()
	m.bindStateCache.items[state] = bindState
}

func (m *Manager) GetBindState(state string) (*BindState, bool) {
	m.bindStateCache.mu.RLock()
	defer m.bindStateCache.mu.RUnlock()
	item, exists := m.bindStateCache.items[state]
	return item, exists
}

func (m *Manager) UpdateBindState(state string, status string, userID string) {
	m.bindStateCache.mu.Lock()
	defer m.bindStateCache.mu.Unlock()
	if item, exists := m.bindStateCache.items[state]; exists {
		item.Status = status
		item.UserID = userID
	}
}

func (m *Manager) DeleteBindState(state string) {
	m.bindStateCache.mu.Lock()
	defer m.bindStateCache.mu.Unlock()
	delete(m.bindStateCache.items, state)
}

func (m *Manager) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := make(map[string]interface{})
	for channelType, config := range m.configs {
		data[string(channelType)] = config.Config
	}

	return json.Marshal(data)
}

package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kefu-server/models"
	"kefu-server/store"
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
	Status  string
	UserID  string
	AgentID uint
	Channel ChannelType
	Expires time.Time
}

const bindStateSettingPrefix = "notification_bind_state:"

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
				m.deletePersistedBindState(key)
			}
		}
		m.bindStateCache.mu.Unlock()
	}
}

func bindStateSettingKey(state string) string {
	return bindStateSettingPrefix + state
}

func (m *Manager) persistBindState(state string, bindState *BindState) {
	if store.DB == nil || state == "" || bindState == nil {
		return
	}

	valueBytes, err := json.Marshal(bindState)
	if err != nil {
		logger.Errorf("notification bind state marshal failed state=%s err=%v", state, err)
		return
	}

	key := bindStateSettingKey(state)
	var setting models.SystemSetting
	result := store.DB.Where("key = ?", key).First(&setting)
	if result.Error != nil {
		setting.Key = key
		setting.Value = string(valueBytes)
		if err := store.DB.Create(&setting).Error; err != nil {
			logger.Errorf("notification bind state create failed state=%s err=%v", state, err)
		}
		return
	}

	setting.Value = string(valueBytes)
	if err := store.DB.Save(&setting).Error; err != nil {
		logger.Errorf("notification bind state update failed state=%s err=%v", state, err)
	}
}

func (m *Manager) loadPersistedBindState(state string) (*BindState, bool) {
	if store.DB == nil || state == "" {
		return nil, false
	}

	var setting models.SystemSetting
	if err := store.DB.Where("key = ?", bindStateSettingKey(state)).First(&setting).Error; err != nil {
		return nil, false
	}

	var bindState BindState
	if err := json.Unmarshal([]byte(setting.Value), &bindState); err != nil {
		logger.Errorf("notification bind state unmarshal failed state=%s err=%v", state, err)
		m.deletePersistedBindState(state)
		return nil, false
	}
	if time.Now().After(bindState.Expires) {
		m.deletePersistedBindState(state)
		return nil, false
	}
	return &bindState, true
}

func (m *Manager) deletePersistedBindState(state string) {
	if store.DB == nil || state == "" {
		return
	}
	if err := store.DB.Where("key = ?", bindStateSettingKey(state)).Delete(&models.SystemSetting{}).Error; err != nil {
		logger.Errorf("notification bind state delete failed state=%s err=%v", state, err)
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
			actualConfig := cfgMap
			if e, ok := cfgMap["enabled"].(bool); ok {
				enabled = e
			}
			if nested, ok := cfgMap["config"].(map[string]interface{}); ok {
				actualConfig = nested
			} else if _, ok := cfgMap["config"]; ok {
				continue
			}

			config := &ChannelConfig{
				Type:    channelType,
				Enabled: enabled,
				Config:  actualConfig,
			}

			if metadata, ok := cfgMap["metadata"].(map[string]string); ok {
				config.Metadata = metadata
			} else if metadataMap, ok := cfgMap["metadata"].(map[string]interface{}); ok {
				config.Metadata = make(map[string]string, len(metadataMap))
				for k, v := range metadataMap {
					if text, ok := v.(string); ok {
						config.Metadata[k] = text
					}
				}
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
	m.bindStateCache.items[state] = bindState
	m.bindStateCache.mu.Unlock()
	m.persistBindState(state, bindState)
}

func (m *Manager) GetBindState(state string) (*BindState, bool) {
	m.bindStateCache.mu.RLock()
	item, exists := m.bindStateCache.items[state]
	m.bindStateCache.mu.RUnlock()
	if exists {
		if time.Now().After(item.Expires) {
			m.DeleteBindState(state)
			return nil, false
		}
		return item, true
	}

	item, exists = m.loadPersistedBindState(state)
	if !exists {
		return nil, false
	}

	m.bindStateCache.mu.Lock()
	m.bindStateCache.items[state] = item
	m.bindStateCache.mu.Unlock()
	return item, true
}

func (m *Manager) UpdateBindState(state string, status string, userID string) {
	item, exists := m.GetBindState(state)
	if !exists || item == nil {
		return
	}
	item.Status = status
	item.UserID = userID

	m.bindStateCache.mu.Lock()
	m.bindStateCache.items[state] = item
	m.bindStateCache.mu.Unlock()
	m.persistBindState(state, item)
}

func (m *Manager) DeleteBindState(state string) {
	m.bindStateCache.mu.Lock()
	delete(m.bindStateCache.items, state)
	m.bindStateCache.mu.Unlock()
	m.deletePersistedBindState(state)
}

func (m *Manager) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data := make(map[string]interface{})
	for channelType, config := range m.configs {
		item := map[string]interface{}{
			"enabled": config.Enabled,
			"config":  config.Config,
		}
		if len(config.Metadata) > 0 {
			item["metadata"] = config.Metadata
		}
		data[string(channelType)] = item
	}

	return json.Marshal(data)
}

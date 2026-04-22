package notification

import "context"

type ChannelType string

const (
	ChannelWecom  ChannelType = "wecom"
	ChannelDing   ChannelType = "dingtalk"
	ChannelFeishu ChannelType = "feishu"
	ChannelTG     ChannelType = "telegram"
)

type Notification struct {
	Title   string
	Content string
	To      string
}

type ChannelConfig struct {
	Type     ChannelType
	Enabled  bool
	Config   map[string]interface{}
	Metadata map[string]string
}

type BindingInfo struct {
	UserID     string
	BindStatus int
	BindTime   interface{}
}

type Notifier interface {
	Type() ChannelType
	Name() string
	ValidateConfig(config map[string]interface{}) error
	TestConnection(config map[string]interface{}) error
	Send(ctx context.Context, to string, notification *Notification) error
	GetBindURL(config map[string]interface{}, userID uint, callbackURL string) (string, string, error)
	HandleCallback(ctx context.Context, config map[string]interface{}, code, state string) (string, error)
}

type NotifierFactory func() Notifier

var notifiers = make(map[ChannelType]NotifierFactory)

func RegisterNotifier(channelType ChannelType, factory NotifierFactory) {
	notifiers[channelType] = factory
}

func GetNotifier(channelType ChannelType) (Notifier, bool) {
	factory, exists := notifiers[channelType]
	if !exists {
		return nil, false
	}
	return factory(), true
}

func GetAllNotifiers() map[ChannelType]Notifier {
	result := make(map[ChannelType]Notifier)
	for channelType, factory := range notifiers {
		result[channelType] = factory()
	}
	return result
}

func GetSupportedChannels() []ChannelType {
	channels := make([]ChannelType, 0, len(notifiers))
	for channelType := range notifiers {
		channels = append(channels, channelType)
	}
	return channels
}

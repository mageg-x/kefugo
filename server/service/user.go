package service

import (
	"fmt"
	"strings"

	"github.com/golang-infrastructure/go-shuffle"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
)

type UserService struct {
}

var (
	instUserService *UserService
)

func GetUserService() *UserService {
	if instUserService == nil {
		instUserService = &UserService{}
	}
	return instUserService
}

func (us *UserService) GetUser(username string) (*models.User, error) {
	var user models.User
	if err := store.DB.Where("username = ?", username).First(&user).Error; err != nil {
		logger.Errorf("user does not exist: %s", username)
		return nil, fmt.Errorf("user does not exist: %s", username)
	}
	return &user, nil
}

func (us *UserService) CreateUser(username, password, avatar, role string, active bool, apps string) (*models.User, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		logger.Errorf("hash password failed: %v", err)
		return nil, fmt.Errorf("hash password failed")
	}

	user := models.User{
		Username: username,
		Password: hashedPassword,
		Role:     role,
		Active:   active,
		Avatar:   avatar,
		Apps:     apps,
	}
	if err := store.DB.Create(&user).Error; err != nil {
		logger.Errorf("create user failed: %s", username)
		return nil, fmt.Errorf("create user failed: %s", username)
	}
	return us.GetUser(username)
}

func (us *UserService) UpdateUser(username, password, avatar, role string, active bool, apps string) (*models.User, error) {
	updates := map[string]interface{}{
		"username": username,
		"role":     role,
		"active":   active,
		"avatar":   avatar,
		"apps":     apps,
	}

	if strings.TrimSpace(password) != "" {
		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			logger.Errorf("hash password failed: %v", err)
			return nil, fmt.Errorf("hash password failed")
		}
		updates["password"] = hashedPassword
	}

	if err := store.DB.Model(&models.User{}).Where("username = ?", username).Updates(updates).Error; err != nil {
		logger.Errorf("update user failed: %s", username)
		return nil, fmt.Errorf("update user failed: %s", username)
	}
	return us.GetUser(username)

}

func (us *UserService) UpdateUserByID(id uint, updates map[string]interface{}) (*models.User, error) {
	if len(updates) == 0 {
		return us.GetUserByID(id)
	}
	if err := store.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		logger.Errorf("update user by id failed: %d, err: %v", id, err)
		return nil, fmt.Errorf("update user failed")
	}
	return us.GetUserByID(id)
}

func (us *UserService) SetUserStatus(username string, status int) error {
	if err := store.DB.Model(&models.User{}).Where("username = ?", username).Update("status", status).Error; err != nil {
		logger.Errorf("failed to set user status: %v, username: %s, status: %d", err, username, status)
		return fmt.Errorf("failed to set user status: %v", err)
	}
	return nil
}

func (us *UserService) SetUserActive(username string, active bool) error {
	if err := store.DB.Model(&models.User{}).Where("username = ?", username).Update("active", active).Error; err != nil {
		logger.Errorf("failed to set user active: %v, username: %s, active: %t", err, username, active)
		return fmt.Errorf("failed to set user active: %v", err)
	}
	return nil
}

func (us *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := store.DB.First(&user, id).Error; err != nil {
		logger.Errorf("user does not exist: %d", id)
		return nil, fmt.Errorf("user does not exist: %d", id)
	}
	return &user, nil
}

// 查找一个能处理此业务的客服（最少连接数策略）
func (us *UserService) FindAgent(appID string) (*models.User, error) {
	var users []models.User
	lowerAppID := strings.ToLower(appID)

	if err := store.DB.Where("role = ? AND status = ? AND active = ?", "agent", 1, true).Find(&users).Error; err != nil {
		logger.Errorf("failed to get agents: %v", err)
		return nil, fmt.Errorf("failed to get agents")
	}

	var matchedAgents []models.User
	for _, user := range users {
		lowerApps := strings.ToLower(user.Apps)
		if strings.Contains(lowerApps, fmt.Sprintf("\"%s\"", lowerAppID)) {
			matchedAgents = append(matchedAgents, user)
		}
	}
	if len(matchedAgents) == 0 {
		for _, user := range users {
			if strings.Contains(strings.ToLower(user.Apps), "all") {
				matchedAgents = append(matchedAgents, user)
			}
		}
	}
	if len(matchedAgents) == 0 {
		logger.Errorf("no available agent found for appID: %s", appID)
		return nil, fmt.Errorf("no available agent found")
	}

	ss := GetSessionService()
	if ss == nil {
		shuffle.Shuffle(matchedAgents)
		return &matchedAgents[0], nil
	}

	type agentLoad struct {
		user     models.User
		count    int64
	}
	loads := make([]agentLoad, 0, len(matchedAgents))
	for _, agent := range matchedAgents {
		count, err := ss.CountActiveSessionsByAgent(agent.Username)
		if err != nil {
			logger.Errorf("count active sessions for agent %s failed: %v", agent.Username, err)
			count = 0
		}
		loads = append(loads, agentLoad{user: agent, count: count})
	}

	var best *agentLoad
	for i := range loads {
		if best == nil || loads[i].count < best.count {
			best = &loads[i]
		}
	}

	if best != nil {
		logger.Infof("find agent by least connections: agent=%s sessions=%d appID=%s", best.user.Username, best.count, appID)
		return &best.user, nil
	}

	shuffle.Shuffle(matchedAgents)
	return &matchedAgents[0], nil
}

func (us *UserService) ListUsers(role string) ([]models.User, error) {
	var users []models.User
	query := store.DB

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if err := query.Order("id DESC").Find(&users).Error; err != nil {
		logger.Errorf("failed to list users: %v", err)
		return nil, fmt.Errorf("failed to list users")
	}

	return users, nil
}

func (us *UserService) DeleteUser(id uint) error {
	if err := store.DB.Unscoped().Delete(&models.User{}, id).Error; err != nil {
		logger.Errorf("failed to delete user: %d", id)
		return fmt.Errorf("failed to delete user")
	}
	return nil
}

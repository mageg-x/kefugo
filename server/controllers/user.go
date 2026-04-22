package controllers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type captchaEntry struct {
	code      string
	expiresAt time.Time
}

const (
	captchaKVPrefix     = "captcha:"
	captchaStoreMaxSize = 1024
	captchaTTL          = 5 * time.Minute
)

var captchaStore = struct {
	mu      sync.RWMutex
	entries map[string]*captchaEntry
}{
	entries: make(map[string]*captchaEntry),
}

func generateCaptchaCode() string {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		code[i] = digits[n.Int64()]
	}
	return string(code)
}

func saveCaptchaToKV(captchaID string, entry *captchaEntry) {
	if strings.TrimSpace(captchaID) == "" || entry == nil {
		return
	}
	kv := store.GetStore()
	if kv == nil {
		return
	}
	payload, err := json.Marshal(map[string]interface{}{
		"code":       strings.TrimSpace(entry.code),
		"expires_at": entry.expiresAt.Unix(),
	})
	if err != nil {
		logger.Errorf("captcha kv marshal failed captcha_id=%s err=%v", captchaID, err)
		return
	}
	expireAt := time.Until(entry.expiresAt)
	if expireAt <= 0 {
		expireAt = 10 * time.Second
	}
	key := []byte(captchaKVPrefix + captchaID)
	err = kv.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry(key, payload).WithTTL(expireAt)
		return txn.SetEntry(entry)
	})
	if err != nil {
		logger.Errorf("captcha kv save failed captcha_id=%s err=%v", captchaID, err)
	}
}

func storeCaptcha(captchaID string) string {
	code := generateCaptchaCode()
	captchaStore.mu.Lock()
	defer captchaStore.mu.Unlock()
	now := time.Now()
	if len(captchaStore.entries) >= captchaStoreMaxSize {
		for k, v := range captchaStore.entries {
			if v == nil || now.After(v.expiresAt) {
				delete(captchaStore.entries, k)
			}
		}
	}
	if len(captchaStore.entries) >= captchaStoreMaxSize {
		// 仍然过多时，按过期时间淘汰最早到期的一批，避免无界增长。
		type pair struct {
			id    string
			entry *captchaEntry
		}
		items := make([]pair, 0, len(captchaStore.entries))
		for k, v := range captchaStore.entries {
			if v != nil {
				items = append(items, pair{id: k, entry: v})
			}
		}
		// 简单选择性淘汰，移除 25% 最早到期项。
		removeTarget := len(items) / 4
		if removeTarget <= 0 {
			removeTarget = 1
		}
		for removed := 0; removed < removeTarget && len(captchaStore.entries) > 0; removed++ {
			oldestID := ""
			var oldestAt time.Time
			for _, it := range items {
				if it.entry == nil {
					continue
				}
				if oldestID == "" || it.entry.expiresAt.Before(oldestAt) {
					oldestID = it.id
					oldestAt = it.entry.expiresAt
				}
			}
			if oldestID == "" {
				break
			}
			delete(captchaStore.entries, oldestID)
		}
	}
	entry := &captchaEntry{
		code:      code,
		expiresAt: now.Add(captchaTTL),
	}
	captchaStore.entries[captchaID] = entry
	saveCaptchaToKV(captchaID, entry)
	return code
}

func verifyCaptcha(captchaID, code string) bool {
	captchaStore.mu.Lock()
	defer captchaStore.mu.Unlock()
	entry, ok := captchaStore.entries[captchaID]
	if !ok || entry == nil {
		// 进程重启后，从 Badger 恢复一次验证码。
		if loaded := loadCaptchaFromKV(captchaID); loaded != nil {
			entry = loaded
			ok = true
		}
		if !ok || entry == nil {
			return false
		}
	}
	delete(captchaStore.entries, captchaID)
	deleteCaptchaFromKV(captchaID)
	if time.Now().After(entry.expiresAt) {
		return false
	}
	return strings.TrimSpace(code) == entry.code
}

func loadCaptchaFromKV(captchaID string) *captchaEntry {
	kv := store.GetStore()
	if kv == nil || strings.TrimSpace(captchaID) == "" {
		return nil
	}
	key := []byte(captchaKVPrefix + captchaID)
	var raw []byte
	err := kv.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		value, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		raw = value
		return nil
	})
	if err != nil || len(raw) == 0 {
		return nil
	}
	var payload struct {
		Code      string `json:"code"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if unmarshalErr := json.Unmarshal(raw, &payload); unmarshalErr != nil {
		return nil
	}
	if strings.TrimSpace(payload.Code) == "" || payload.ExpiresAt <= 0 {
		return nil
	}
	return &captchaEntry{
		code:      strings.TrimSpace(payload.Code),
		expiresAt: time.Unix(payload.ExpiresAt, 0),
	}
}

func deleteCaptchaFromKV(captchaID string) {
	kv := store.GetStore()
	if kv == nil || strings.TrimSpace(captchaID) == "" {
		return
	}
	key := []byte(captchaKVPrefix + captchaID)
	_ = kv.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(key); err != nil && err != badger.ErrKeyNotFound {
			return err
		}
		return nil
	})
}

type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Captcha   string `json:"captcha"`
	CaptchaID string `json:"captcha_id"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// UserController 负责用户认证、用户管理与个人资料相关接口。
type UserController struct{}

// normalizeUserRole 归一并校验用户角色，只允许 admin/agent。
func normalizeUserRole(role string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return "admin", true
	case "agent":
		return "agent", true
	default:
		return "", false
	}
}

// IsAdmin 判断当前上下文用户是否为管理员。
func IsAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	if !exists {
		return false
	}
	roleText, ok := role.(string)
	if !ok {
		return false
	}
	return roleText == "admin"
}

// Login 处理账号密码登录并签发 JWT。
func (uc *UserController) Login(c *gin.Context) {
	cfg := getSystemSettingsCached()
	if cfg.IPLimit {
		remoteIP := strings.TrimSpace(c.ClientIP())
		if !isIPAllowed(remoteIP, cfg.IPWhitelist) {
			logger.Errorf("login ip blocked ip=%s", remoteIP)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeSecurityIPBlocked)
			return
		}
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("login request parameter error: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeAuthLoginInvalidParams)
		return
	}
	if cfg.Captcha {
		if strings.TrimSpace(req.CaptchaID) == "" || strings.TrimSpace(req.Captcha) == "" {
			logger.Errorf("login captcha required username=%s", strings.TrimSpace(req.Username))
			response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSecurityCaptchaRequired)
			return
		}
		if !verifyCaptcha(strings.TrimSpace(req.CaptchaID), strings.TrimSpace(req.Captcha)) {
			logger.Errorf("login captcha invalid username=%s", strings.TrimSpace(req.Username))
			response.ResponseError(c, http.StatusBadRequest, response.ErrCodeSecurityCaptchaInvalid)
			return
		}
	}

	// 使用 UserService 获取用户
	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	user, err := userService.GetUser(req.Username)
	if err != nil || user == nil {
		logger.Errorf("user does not exist: %s", req.Username)
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthInvalidCredentials)
		return
	}

	if !utils.VerifyPassword(user.Password, req.Password) {
		logger.Errorf("password error: %s", req.Username)
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthInvalidCredentials)
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		logger.Errorf("generate token failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAuthTokenGenerateFailed)
		return
	}

	logger.Infof("user login successful: %s", req.Username)
	response.ResponseSuccess(c, LoginResponse{
		Token: token,
		User:  *user,
	})
}

// GetUserInfo 获取当前登录用户信息。
func (uc *UserController) GetUserInfo(c *gin.Context) {
	userName, exists := c.Get("userName")
	if !exists {
		logger.Errorf("failed get user name")
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}

	// 使用 UserService 获取用户
	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}
	user, err := userService.GetUser(userName.(string))
	if err != nil || user == nil {
		logger.Errorf("get user info failed: %v", err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeUserInfoNotFound)
		return
	}

	response.ResponseSuccess(c, gin.H{"user": user})
}

// Logout 当前实现为无状态登出，前端删除 token 即可。
func (uc *UserController) Logout(c *gin.Context) {
	logger.Infof("user logout")
	response.ResponseSuccess(c, gin.H{"message": "logout successful"})
}

// ListUsers 返回用户列表，支持角色、激活状态与按 id 集合查询。
func (uc *UserController) ListUsers(c *gin.Context) {
	role := strings.TrimSpace(c.Query("role"))
	if role != "" {
		normalizedRole, ok := normalizeUserRole(role)
		if !ok {
			response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserRoleInvalid)
			return
		}
		role = normalizedRole
	}
	activeStr := strings.TrimSpace(c.Query("active"))
	idsRaw := strings.TrimSpace(c.Query("ids"))

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	if idsRaw != "" {
		ids := make([]uint, 0)
		for _, p := range strings.Split(idsRaw, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if idNum, parseErr := strconv.ParseUint(p, 10, 32); parseErr == nil && idNum > 0 {
				ids = append(ids, uint(idNum))
			}
		}
		if len(ids) > 0 {
			var users []models.User
			if err := store.DB.Where("id IN ?", ids).Order("id DESC").Find(&users).Error; err != nil {
				logger.Errorf("list users by ids failed: %v", err)
				response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserListByIDsFailed)
				return
			}
			response.ResponseSuccess(c, gin.H{"users": users})
			return
		}
	}

	users, err := userService.ListUsers(role)
	if err != nil {
		logger.Errorf("list users failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserListFailed)
		return
	}
	if activeStr != "" {
		wantActive := activeStr == "1" || strings.EqualFold(activeStr, "true")
		filtered := make([]models.User, 0, len(users))
		for _, u := range users {
			if u.Active == wantActive {
				filtered = append(filtered, u)
			}
		}
		users = filtered
	}

	response.ResponseSuccess(c, gin.H{"users": users})
}

// BatchActive 批量启用/禁用用户（管理员账号不允许被批量操作）。
func (uc *UserController) BatchActive(c *gin.Context) {
	var req struct {
		IDs    []uint `json:"ids" binding:"required"`
		Active bool   `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		logger.Errorf("user batch active params invalid err=%v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserBatchActiveInvalidParams)
		return
	}
	if err := store.DB.Model(&models.User{}).
		Where("id IN ? AND role NOT IN ?", req.IDs, []string{"admin"}).
		Update("active", req.Active).Error; err != nil {
		logger.Errorf("user batch active update failed ids=%v active=%t err=%v", req.IDs, req.Active, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserBatchActiveFailed)
		return
	}
	RecordAudit(c, "user.batch_active", "user", "batch", "success", strconv.Itoa(len(req.IDs)))
	response.ResponseSuccess(c, gin.H{"updated": len(req.IDs)})
}

// DeleteUser 硬删除指定用户。
func (uc *UserController) DeleteUser(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		logger.Errorf("user id not provided")
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserDeleteInvalidID)
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Errorf("invalid user id: %s", idStr)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserDeleteInvalidID)
		return
	}

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	if err := userService.DeleteUser(uint(id)); err != nil {
		logger.Errorf("delete user failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserDeleteFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
	RecordAudit(c, "user.delete", "user", idStr, "success", "")
}

// UpdateUser 更新用户信息（含可选密码重置）。
func (uc *UserController) UpdateUser(c *gin.Context) {
	var req struct {
		ID       uint     `json:"id" binding:"required"`
		Username string   `json:"username" binding:"required"`
		Password string   `json:"password"`
		Avatar   string   `json:"avatar"`
		Role     string   `json:"role" binding:"required"`
		Active   bool     `json:"active"`
		Apps     []string `json:"apps"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("update user request parameter error: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserUpdateInvalidParams)
		return
	}
	normalizedRole, ok := normalizeUserRole(req.Role)
	if !ok {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserRoleInvalid)
		return
	}
	req.Role = normalizedRole

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	user, err := userService.GetUserByID(req.ID)
	if err != nil || user == nil {
		logger.Errorf("user does not exist: %d", req.ID)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeUserUpdateNotFound)
		return
	}

	apps := user.Apps
	if len(req.Apps) > 0 {
		appsBytes, _ := json.Marshal(req.Apps)
		apps = string(appsBytes)
	}

	updates := map[string]interface{}{
		"username": req.Username,
		"avatar":   req.Avatar,
		"role":     req.Role,
		"active":   req.Active,
		"apps":     apps,
	}
	if req.Password != "" {
		hashed, hashErr := utils.HashPassword(req.Password)
		if hashErr != nil {
			logger.Errorf("hash password failed: %v", hashErr)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserPasswordHashFailed)
			return
		}
		updates["password"] = hashed
	}

	updatedUser, err := userService.UpdateUserByID(req.ID, updates)
	if err != nil {
		logger.Errorf("update user failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserUpdateFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"user": updatedUser})
	if req.Password != "" {
		RecordAudit(c, "user.password.reset", "user", req.Username, "success", "admin reset password")
	} else {
		RecordAudit(c, "user.update", "user", req.Username, "success", req.Role)
	}
}

// CreateUser 创建新用户。
func (uc *UserController) CreateUser(c *gin.Context) {
	var req struct {
		Username string   `json:"username" binding:"required"`
		Password string   `json:"password" binding:"required"`
		Avatar   string   `json:"avatar"`
		Role     string   `json:"role" binding:"required"`
		Active   bool     `json:"active"`
		Apps     []string `json:"apps"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("create user request parameter error: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserCreateInvalidParams)
		return
	}
	normalizedRole, ok := normalizeUserRole(req.Role)
	if !ok {
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserRoleInvalid)
		return
	}
	req.Role = normalizedRole

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	apps := "[]"
	if len(req.Apps) > 0 {
		appsBytes, _ := json.Marshal(req.Apps)
		apps = string(appsBytes)
	}

	user, err := userService.CreateUser(req.Username, req.Password, req.Avatar, req.Role, req.Active, apps)
	if err != nil {
		logger.Errorf("create user failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserCreateFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"user": user})
	RecordAudit(c, "user.create", "user", req.Username, "success", req.Role)
}

// SetUserStatus 更新当前登录用户在席状态（0/1）。
func (uc *UserController) SetUserStatus(c *gin.Context) {
	userName, exists := c.Get("userName")
	if !exists {
		logger.Errorf("failed get user name")
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}

	// 直接读取请求体
	var req struct {
		Status int `json:"status"`
	}

	body, err := c.GetRawData()
	if err != nil {
		logger.Errorf("failed to read request body: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserStatusInvalidParams)
		return
	}

	// 手动解析JSON
	if err := json.Unmarshal(body, &req); err != nil {
		logger.Errorf("failed to unmarshal JSON: %v", err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserStatusInvalidParams)
		return
	}

	// 验证参数
	if req.Status != 0 && req.Status != 1 {
		logger.Errorf("invalid status: %d", req.Status)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserStatusInvalidValue)
		return
	}

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	if err := userService.SetUserStatus(userName.(string), req.Status); err != nil {
		logger.Errorf("set user status failed: %v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserStatusUpdateFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"message": "set status successful"})
}

// GetUserStatus 获取当前登录用户在席状态。
func (uc *UserController) GetUserStatus(c *gin.Context) {
	userName, exists := c.Get("userName")
	if !exists {
		logger.Errorf("failed get user name")
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}

	userService := service.GetUserService()
	if userService == nil {
		logger.Errorf("user service not initialed")
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
		return
	}

	user, err := userService.GetUser(userName.(string))
	if err != nil || user == nil {
		logger.Errorf("get user info failed: %v", err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeUserStatusQueryFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"status": user.Status})
}

// UpdateProfile 更新当前登录用户个人资料字段。
func (uc *UserController) UpdateProfile(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		logger.Errorf("user profile auth context missing")
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}
	userID, ok := userIDVal.(uint)
	if !ok || userID == 0 {
		logger.Errorf("user profile user id invalid value=%v", userIDVal)
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeUserIDInvalid)
		return
	}

	var req struct {
		Avatar string `json:"avatar"`
		Email  string `json:"email"`
		Phone  string `json:"phone"`
		Bio    string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("user profile params invalid user_id=%d err=%v", userID, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserProfileInvalidParams)
		return
	}

	userService := service.GetUserService()
	updated, err := userService.UpdateUserByID(userID, map[string]interface{}{
		"avatar": req.Avatar,
		"email":  req.Email,
		"phone":  req.Phone,
		"bio":    req.Bio,
	})
	if err != nil {
		logger.Errorf("user profile update failed user_id=%d err=%v", userID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserProfileUpdateFailed)
		return
	}

	RecordAudit(c, "user.profile.update", "user", strconv.FormatUint(uint64(userID), 10), "success", "")
	response.ResponseSuccess(c, gin.H{"user": updated})
}

// ChangePassword 修改当前登录用户密码。
func (uc *UserController) ChangePassword(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		logger.Errorf("user password auth context missing")
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
		return
	}
	userNameVal, _ := c.Get("userName")
	userName, _ := userNameVal.(string)
	userID, ok := userIDVal.(uint)
	if !ok || userID == 0 || userName == "" {
		logger.Errorf("user password user id invalid user_id=%v user_name=%s", userIDVal, userName)
		response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeUserIDInvalid)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("user password params invalid user_id=%d err=%v", userID, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeUserPasswordInvalidParams)
		return
	}
	if len(req.NewPassword) < 8 {
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUserPasswordTooShort, "new password too short")
		return
	}

	userService := service.GetUserService()
	user, err := userService.GetUser(userName)
	if err != nil || user == nil {
		logger.Errorf("user password target not found user_name=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeUserNotFound)
		return
	}
	if !utils.VerifyPassword(user.Password, req.CurrentPassword) {
		logger.Errorf("user password current password invalid user_id=%d", userID)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUserPasswordCurrentInvalid, "current password invalid")
		return
	}

	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		logger.Errorf("user password hash failed user_id=%d err=%v", userID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserPasswordHashFailed)
		return
	}
	if _, err := userService.UpdateUserByID(userID, map[string]interface{}{"password": hashed}); err != nil {
		logger.Errorf("user password update failed user_id=%d err=%v", userID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserPasswordChangeFailed)
		return
	}

	RecordAudit(c, "user.password.change", "user", strconv.FormatUint(uint64(userID), 10), "success", "")
	response.ResponseSuccess(c, gin.H{"message": "password changed"})
}

func (uc *UserController) GetCaptcha(c *gin.Context) {
	cfg := getSystemSettingsCached()
	if !cfg.Captcha {
		response.ResponseSuccess(c, gin.H{"enabled": false})
		return
	}
	captchaID := fmt.Sprintf("%d", time.Now().UnixNano())
	code := storeCaptcha(captchaID)
	response.ResponseSuccess(c, gin.H{
		"enabled":      true,
		"captcha_id":   captchaID,
		"captcha_code": code,
	})
}

func isIPAllowed(remoteIP, whitelist string) bool {
	if remoteIP == "127.0.0.1" || remoteIP == "::1" || remoteIP == "::ffff:127.0.0.1" {
		return true
	}
	if whitelist == "" {
		return true
	}
	for _, item := range strings.Split(whitelist, ",") {
		ip := strings.TrimSpace(item)
		if ip == "" {
			continue
		}
		if ip == remoteIP {
			return true
		}
	}
	return false
}

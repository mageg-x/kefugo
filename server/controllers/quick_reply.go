package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// QuickReplyController 管理快捷回复的增删改查与使用计数。
type QuickReplyController struct{}

func seedDefaultQuickRepliesForUser(userName string) {
	userName = strings.TrimSpace(userName)
	if userName == "" || store.DB == nil {
		return
	}
	var count int64
	if err := store.DB.Model(&models.QuickReply{}).Where("owner = ?", userName).Count(&count).Error; err != nil {
		logger.Errorf("quick reply seed count failed user=%s err=%v", userName, err)
		return
	}
	if count > 0 {
		return
	}
	defaultRows := []models.QuickReply{
		{Owner: userName, Title: "欢迎接待", Category: "问候", Content: "您好，欢迎咨询，我是本次为您服务的客服，请问有什么可以帮您？"},
		{Owner: userName, Title: "请稍候", Category: "问候", Content: "收到，我正在为您核实处理中，请稍候 1-2 分钟。"},
		{Owner: userName, Title: "索取订单号", Category: "售后", Content: "为更快帮您处理，请提供订单号或手机号后四位。"},
		{Owner: userName, Title: "发票说明", Category: "售后", Content: "发票可在下单后申请开具，若需补开请提供订单信息与开票抬头。"},
		{Owner: userName, Title: "结束语", Category: "问候", Content: "本次咨询已为您处理完成，如有其他问题随时联系，祝您生活愉快。"},
	}
	if err := store.DB.Create(&defaultRows).Error; err != nil {
		logger.Errorf("quick reply seed create failed user=%s err=%v", userName, err)
		return
	}
	logger.Infof("quick reply seeded defaults user=%s size=%d", userName, len(defaultRows))
}

// List 返回当前登录用户的快捷回复列表。
func (qc *QuickReplyController) List(c *gin.Context) {
	userName, _ := getAuthUser(c)
	seedDefaultQuickRepliesForUser(userName)
	category := strings.TrimSpace(c.Query("category"))

	query := store.DB.Model(&models.QuickReply{}).Where("owner = ?", userName)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var rows []models.QuickReply
	if err := query.Order("updated_at DESC").Find(&rows).Error; err != nil {
		logger.Errorf("quick reply list query failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeQuickReplyListFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{"data": rows})
}

// Create 创建一条新的快捷回复。
func (qc *QuickReplyController) Create(c *gin.Context) {
	userName, _ := getAuthUser(c)
	var req struct {
		Title    string `json:"title" binding:"required"`
		Category string `json:"category" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("quick reply create params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeQuickReplyCreateInvalidParams)
		return
	}

	item := models.QuickReply{
		Owner:    userName,
		Title:    strings.TrimSpace(req.Title),
		Category: strings.TrimSpace(req.Category),
		Content:  strings.TrimSpace(req.Content),
	}
	if err := store.DB.Create(&item).Error; err != nil {
		logger.Errorf("quick reply create failed user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeQuickReplyCreateFailed)
		return
	}
	RecordAudit(c, "quick_reply.create", "quick_reply", strconv.FormatUint(uint64(item.ID), 10), "success", item.Title)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Update 更新指定快捷回复内容。
func (qc *QuickReplyController) Update(c *gin.Context) {
	userName, _ := getAuthUser(c)
	var req struct {
		ID       uint   `json:"id" binding:"required"`
		Title    string `json:"title" binding:"required"`
		Category string `json:"category" binding:"required"`
		Content  string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("quick reply update params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeQuickReplyUpdateInvalidParams)
		return
	}

	var item models.QuickReply
	if err := store.DB.Where("id = ? AND owner = ?", req.ID, userName).First(&item).Error; err != nil {
		logger.Errorf("quick reply not found for update user=%s id=%d err=%v", userName, req.ID, err)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeQuickReplyNotFound)
		return
	}

	item.Title = strings.TrimSpace(req.Title)
	item.Category = strings.TrimSpace(req.Category)
	item.Content = strings.TrimSpace(req.Content)
	if err := store.DB.Save(&item).Error; err != nil {
		logger.Errorf("quick reply update failed user=%s id=%d err=%v", userName, item.ID, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeQuickReplyUpdateFailed)
		return
	}
	RecordAudit(c, "quick_reply.update", "quick_reply", strconv.FormatUint(uint64(item.ID), 10), "success", item.Title)
	response.ResponseSuccess(c, gin.H{"item": item})
}

// Delete 删除指定快捷回复（硬删除）。
func (qc *QuickReplyController) Delete(c *gin.Context) {
	userName, _ := getAuthUser(c)
	idStr := c.Query("id")
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id64 == 0 {
		logger.Errorf("quick reply delete params invalid user=%s id=%s", userName, idStr)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeQuickReplyDeleteInvalidParams)
		return
	}

	result := store.DB.Unscoped().Where("id = ? AND owner = ?", uint(id64), userName).Delete(&models.QuickReply{})
	if result.Error != nil {
		logger.Errorf("quick reply delete failed user=%s id=%d err=%v", userName, id64, result.Error)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeQuickReplyDeleteFailed)
		return
	}
	if result.RowsAffected == 0 {
		logger.Errorf("quick reply delete target not found user=%s id=%d", userName, id64)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeQuickReplyNotFound)
		return
	}
	RecordAudit(c, "quick_reply.delete", "quick_reply", idStr, "success", "")
	response.ResponseSuccess(c, gin.H{"message": "delete successful"})
}

// Use 增加快捷回复使用次数。
func (qc *QuickReplyController) Use(c *gin.Context) {
	userName, _ := getAuthUser(c)
	var req struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("quick reply use params invalid user=%s err=%v", userName, err)
		response.ResponseError(c, http.StatusBadRequest, response.ErrCodeQuickReplyUseInvalidParams)
		return
	}

	result := store.DB.Model(&models.QuickReply{}).
		Where("id = ? AND owner = ?", req.ID, userName).
		Update("usage_count", gorm.Expr("usage_count + 1"))
	if result.Error != nil {
		logger.Errorf("quick reply use failed user=%s id=%d err=%v", userName, req.ID, result.Error)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeQuickReplyUseFailed)
		return
	}
	if result.RowsAffected == 0 {
		logger.Errorf("quick reply use target not found user=%s id=%d", userName, req.ID)
		response.ResponseError(c, http.StatusNotFound, response.ErrCodeQuickReplyNotFound)
		return
	}
	response.ResponseSuccess(c, gin.H{"message": "ok"})
}

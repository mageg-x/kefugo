package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/models"
	"kefu-server/store"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

type AuditController struct{}

// RecordAudit 记录关键操作审计日志。
// 失败时仅记录日志，不影响主业务流程。
func RecordAudit(c *gin.Context, action, targetType, targetID, result, detail string) {
	userName, role := getAuthUser(c)
	if userName == "" {
		userName = "system"
	}

	log := models.AuditLog{
		Operator:     userName,
		OperatorRole: role,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Result:       result,
		Detail:       detail,
	}
	if err := store.DB.Create(&log).Error; err != nil {
		logger.Errorf("audit log write failed action=%s target_type=%s target_id=%s err=%v", action, targetType, targetID, err)
	}
}

// List 按分页和条件筛选返回审计日志。
func (ac *AuditController) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	operator := c.Query("operator")
	action := c.Query("action")
	result := c.Query("result")
	startTimeStr := strings.TrimSpace(c.Query("start_time"))
	endTimeStr := strings.TrimSpace(c.Query("end_time"))

	query := store.DB.Model(&models.AuditLog{})
	if operator != "" {
		query = query.Where("operator LIKE ?", "%"+operator+"%")
	}
	if action != "" {
		query = query.Where("action LIKE ?", "%"+action+"%")
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	if startTimeStr != "" {
		if startTS, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil && startTS > 0 {
			query = query.Where("created_at >= ?", time.Unix(startTS, 0))
		}
	}
	if endTimeStr != "" {
		if endTS, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil && endTS > 0 {
			query = query.Where("created_at <= ?", time.Unix(endTS, 0))
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("audit count query failed err=%v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAuditCountFailed)
		return
	}

	var rows []models.AuditLog
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		logger.Errorf("audit list query failed err=%v", err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeAuditListFailed)
		return
	}

	response.ResponseSuccess(c, gin.H{
		"data":      rows,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

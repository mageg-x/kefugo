package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/config"
	"kefu-server/models"
	"kefu-server/service"
	"kefu-server/store"
	"kefu-server/utils"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// UploadController 提供访客/客服文件上传接口。
type UploadController struct{}

// CreatePublic 访客侧公开上传入口（需域名白名单校验）。
func (uc *UploadController) CreatePublic(c *gin.Context) {
	uc.create(c, false)
}

// Create 客服侧鉴权上传入口（需登录且具备角色权限）。
func (uc *UploadController) Create(c *gin.Context) {
	uc.create(c, true)
}

// create 执行上传主流程：
// 1) 鉴权/角色检查
// 2) 应用与域名检查
// 3) 文件合法性校验
// 4) 文件落盘并返回访问地址
func (uc *UploadController) create(c *gin.Context, needAuth bool) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	authRole := ""
	if needAuth {
		userName, role := getAuthUser(c)
		if userName == "" || role == "" {
			logger.Errorf("upload auth context missing")
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeUploadUnauthorized)
			return
		}
		authRole = role
		if authRole != "agent" && authRole != "admin" {
			logger.Errorf("upload role forbidden role=%s", strings.ToLower(strings.TrimSpace(authRole)))
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeUploadRoleForbidden)
			return
		}
	}

	if err := c.Request.ParseMultipartForm(models.MaxFileSizeBytes + (1 << 20)); err != nil {
		logger.Errorf("upload multipart parse failed: %v", err)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadMultipartInvalid, "invalid multipart body")
		return
	}

	appID := strings.TrimSpace(c.PostForm("app_id"))
	if appID == "" {
		appID = strings.TrimSpace(c.Query("app_id"))
	}
	if appID == "" {
		appID = strings.TrimSpace(c.Query("appid"))
	}
	if appID == "" {
		logger.Errorf("upload app_id missing")
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadAppIDRequired, "app_id required")
		return
	}
	var app models.App
	query := store.DB.Where("app_id = ?", appID)
	if !needAuth {
		query = query.Where("status = ?", 1)
	}
	if err := query.First(&app).Error; err != nil {
		logger.Errorf("upload app query failed app_id=%s err=%v", appID, err)
		response.ResponseError(c, http.StatusForbidden, response.ErrCodeUploadAppNotFoundDisabled)
		return
	}
	if needAuth && authRole == "agent" {
		userName, _ := getAuthUser(c)
		userService := service.GetUserService()
		if userService == nil {
			logger.Errorf("upload user service unavailable user=%s app_id=%s", userName, appID)
			response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUserServiceUnavailable)
			return
		}
		user, err := userService.GetUser(userName)
		if err != nil || user == nil {
			logger.Errorf("upload current user not found user=%s app_id=%s err=%v", userName, appID, err)
			response.ResponseError(c, http.StatusUnauthorized, response.ErrCodeAuthContextMissing)
			return
		}
		if !isAgentForApp(user.Apps, appID) {
			logger.Errorf("upload app access denied user=%s app_id=%s", userName, appID)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodePermissionAppAccessDenied)
			return
		}
	}

	if !needAuth {
		origin := c.GetHeader("Origin")
		referer := c.GetHeader("Referer")
		if !models.IsDomainAllowed(origin, referer, app.AllowDomain) {
			logger.Errorf("upload domain forbidden app_id=%s origin=%s referer=%s", appID, origin, referer)
			response.ResponseError(c, http.StatusForbidden, response.ErrCodeUploadDomainForbidden)
			return
		}
	}

	contentType := strings.TrimSpace(c.PostForm("content_type"))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		logger.Errorf("upload file missing: %v", err)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadFileRequired, "file required")
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		logger.Errorf("upload file open failed name=%s err=%v", fileHeader.Filename, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadOpenFileFailed)
		return
	}
	defer src.Close()
	select {
	case <-ctx.Done():
		logger.Errorf("upload request canceled before validate: %v", ctx.Err())
		response.ResponseErrorWithMsg(c, http.StatusRequestTimeout, response.ErrCodeRequestTimeout, "upload timeout")
		return
	default:
	}
	if contentType == "" {
		contentType = inferContentTypeByFile(fileHeader)
	}

	if _, err := validateUploadFile(fileHeader.Filename, fileHeader.Size, contentType); err != nil {
		logger.Errorf("upload file validate failed name=%s size=%d type=%s err=%v", fileHeader.Filename, fileHeader.Size, contentType, err)
		response.ResponseErrorWithMsg(c, http.StatusBadRequest, response.ErrCodeUploadValidateFailed, err.Error())
		return
	}

	cfg := config.GetConfig()
	uploadDir := "data/uploads"
	if cfg != nil && strings.TrimSpace(cfg.Admin.UploadDir) != "" {
		uploadDir = strings.TrimSpace(cfg.Admin.UploadDir)
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Errorf("upload mkdir failed dir=%s err=%v", uploadDir, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadMkdirFailed)
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	saveName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), utils.GenerateRandomString(8), ext)
	savePath := filepath.Join(uploadDir, saveName)
	dst, err := os.Create(savePath)
	if err != nil {
		logger.Errorf("upload create file failed path=%s err=%v", savePath, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadCreateFileFailed)
		return
	}
	copyErrCh := make(chan error, 1)
	go func() {
		_, err := io.Copy(dst, src)
		copyErrCh <- err
	}()
	select {
	case <-ctx.Done():
		_ = dst.Close()
		_ = os.Remove(savePath)
		logger.Errorf("upload write file timeout path=%s err=%v", savePath, ctx.Err())
		response.ResponseErrorWithMsg(c, http.StatusRequestTimeout, response.ErrCodeRequestTimeout, "upload timeout")
		return
	case err = <-copyErrCh:
	}
	if err != nil {
		_ = dst.Close()
		_ = os.Remove(savePath)
		logger.Errorf("upload write file failed path=%s err=%v", savePath, err)
		response.ResponseError(c, http.StatusInternalServerError, response.ErrCodeUploadWriteFileFailed)
		return
	}
	_ = dst.Close()

	urlPath := "/uploads/" + saveName
	resp := map[string]interface{}{
		"url":  urlPath,
		"name": fileHeader.Filename,
		"size": fileHeader.Size,
		"type": contentType,
	}
	respBytes, _ := json.Marshal(resp)
	RecordAudit(c, "upload.create", "upload", saveName, "success", string(respBytes))
	response.ResponseSuccess(c, resp)
}

func inferContentTypeByFile(fileHeader *multipart.FileHeader) string {
	if fileHeader == nil {
		return models.WSContentTypeFile
	}
	mimeType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileHeader.Filename)))
	if strings.HasPrefix(mimeType, "image/") || ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" || ext == ".bmp" || ext == ".svg" {
		return models.WSContentTypeImage
	}
	if strings.HasPrefix(mimeType, "audio/") || ext == ".mp3" || ext == ".wav" || ext == ".ogg" || ext == ".m4a" || ext == ".aac" || ext == ".webm" {
		return models.WSContentTypeAudio
	}
	return models.WSContentTypeFile
}

func validateUploadFile(fileName string, size int64, contentType string) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("empty file")
	}
	if size > models.MaxFileSizeBytes {
		return "", fmt.Errorf("file too large")
	}
	payload := &models.WSMessagePayload{
		ContentType: contentType,
		Name:        fileName,
		Size:        size,
		URL:         "https://localhost/upload-placeholder",
		Content:     "https://localhost/upload-placeholder",
	}
	if contentType == models.WSContentTypeAudio {
		payload.Duration = 0
	}
	if err := payload.Validate(); err != nil {
		return "", err
	}
	return fileName, nil
}

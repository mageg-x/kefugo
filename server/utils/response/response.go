package response

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"kefu-server/utils/logger"
)

// Response 是后端统一的 API 响应结构。
// Code 用于前端做稳定错误码判断，Msg 用于默认文案，Data 承载业务数据。
type Response struct {
	Code ErrorCode   `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// ResponseSuccess 返回标准成功响应。
func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  MessageForCode(ErrCodeSuccess),
		Data: data,
	})
}

// ResponseSuccessWithMsg 返回带自定义提示文案的成功响应。
func ResponseSuccessWithMsg(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: ErrCodeSuccess,
		Msg:  msg,
		Data: data,
	})
}

// ResponseError 返回标准错误响应，并统一记录英文小写错误日志。
// 通过统一出口记录日志，可以覆盖绝大多数控制器错误分支，降低漏打日志风险。
func ResponseError(c *gin.Context, httpStatus int, code ErrorCode) {
	msg := MessageForCode(code)
	logErrorResponse(c, httpStatus, code, msg)
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// ResponseErrorWithMsg 返回带自定义提示文案的错误响应，并统一记录英文小写错误日志。
func ResponseErrorWithMsg(c *gin.Context, httpStatus int, code ErrorCode, msg string) {
	logErrorResponse(c, httpStatus, code, msg)
	c.JSON(httpStatus, Response{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}

// logErrorResponse 记录统一错误响应日志。
// 日志字段尽量稳定，便于按 method/path/code/status 聚合排查问题。
func logErrorResponse(c *gin.Context, httpStatus int, code ErrorCode, msg string) {
	method := ""
	path := ""
	clientIP := ""
	if c != nil {
		method = strings.ToLower(strings.TrimSpace(c.Request.Method))
		path = strings.TrimSpace(c.Request.URL.Path)
		clientIP = strings.TrimSpace(c.ClientIP())
	}
	logger.Errorf("api error method=%s path=%s status=%d code=%d ip=%s msg=%s",
		method, path, httpStatus, code, clientIP, strings.TrimSpace(msg))
}

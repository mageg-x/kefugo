package response

// ErrorCode 定义错误码类型。
type ErrorCode int

const (
	// 0: 成功
	ErrCodeSuccess ErrorCode = 0

	// 10000-10999: 通用
	ErrCodeUnknown              ErrorCode = 10000
	ErrCodeInvalidParams        ErrorCode = 10001
	ErrCodeBadRequest           ErrorCode = 10002
	ErrCodeNotFound             ErrorCode = 10003
	ErrCodeConflict             ErrorCode = 10004
	ErrCodeInternalError        ErrorCode = 10005
	ErrCodeServiceUnavailable   ErrorCode = 10006
	ErrCodeMethodNotAllowed     ErrorCode = 10007
	ErrCodeRequestTimeout       ErrorCode = 10008
	ErrCodeUnsupportedMediaType ErrorCode = 10009
	ErrCodePayloadTooLarge      ErrorCode = 10010
	ErrCodeIllegalState         ErrorCode = 10011
	ErrCodeDataCorrupted        ErrorCode = 10012
	ErrCodeOperationFailed      ErrorCode = 10013
	ErrCodeInvalidCursor        ErrorCode = 10014
	ErrCodeInvalidSnapshot      ErrorCode = 10015

	// 11000-11999: 认证
	ErrCodeAuthUnauthorized        ErrorCode = 11000
	ErrCodeAuthTokenRequired       ErrorCode = 11001
	ErrCodeAuthTokenFormatInvalid  ErrorCode = 11002
	ErrCodeAuthTokenInvalid        ErrorCode = 11003
	ErrCodeAuthTokenExpired        ErrorCode = 11004
	ErrCodeAuthTokenRevoked        ErrorCode = 11005
	ErrCodeAuthTokenClaimsInvalid  ErrorCode = 11006
	ErrCodeAuthContextMissing      ErrorCode = 11007
	ErrCodeAuthInvalidCredentials  ErrorCode = 11008
	ErrCodeAuthLoginInvalidParams  ErrorCode = 11009
	ErrCodeAuthLoginUserNotFound   ErrorCode = 11010
	ErrCodeAuthLoginUserDisabled   ErrorCode = 11011
	ErrCodeAuthTokenGenerateFailed ErrorCode = 11012
	ErrCodeAuthPasswordTooWeak     ErrorCode = 11013
	ErrCodeAuthPasswordInvalid     ErrorCode = 11014
	ErrCodeAuthPasswordPolicy      ErrorCode = 11015

	// 12000-12999: 权限
	ErrCodePermissionForbidden          ErrorCode = 12000
	ErrCodePermissionRoleDenied         ErrorCode = 12001
	ErrCodePermissionAdminRequired      ErrorCode = 12002
	ErrCodePermissionAgentRequired      ErrorCode = 12003
	ErrCodePermissionSuperAdminRequired ErrorCode = 12004
	ErrCodePermissionAppAccessDenied    ErrorCode = 12005
	ErrCodePermissionSessionDenied      ErrorCode = 12006
	ErrCodePermissionOwnershipMismatch  ErrorCode = 12007
	ErrCodePermissionOperationDenied    ErrorCode = 12008
	ErrCodePermissionCrossAppDenied     ErrorCode = 12009

	// 13000-13999: 安全/风控
	ErrCodeSecurityTooManyRequests     ErrorCode = 13000
	ErrCodeSecurityRateLimitExceeded   ErrorCode = 13001
	ErrCodeSecurityCSRFInvalid         ErrorCode = 13002
	ErrCodeSecurityOriginNotAllowed    ErrorCode = 13003
	ErrCodeSecurityDomainNotAllowed    ErrorCode = 13004
	ErrCodeSecurityCaptchaRequired     ErrorCode = 13005
	ErrCodeSecurityCaptchaInvalid      ErrorCode = 13006
	ErrCodeSecuritySensitiveDetected   ErrorCode = 13007
	ErrCodeSecurityIPBlocked           ErrorCode = 13008
	ErrCodeSecurityRiskControlRejected ErrorCode = 13009

	// 20000-20999: 应用
	ErrCodeAppListFailed           ErrorCode = 20000
	ErrCodeAppCreateInvalidParams  ErrorCode = 20001
	ErrCodeAppIDDuplicated         ErrorCode = 20002
	ErrCodeAppCreateFailed         ErrorCode = 20003
	ErrCodeAppNotFound             ErrorCode = 20004
	ErrCodeAppUpdateInvalidParams  ErrorCode = 20005
	ErrCodeAppUpdateNoChanges      ErrorCode = 20006
	ErrCodeAppUpdateFailed         ErrorCode = 20007
	ErrCodeAppDeleteInvalidParams  ErrorCode = 20008
	ErrCodeAppDeleteFailed         ErrorCode = 20009
	ErrCodeAppDisabled             ErrorCode = 20010
	ErrCodeAppConfigInvalidParams  ErrorCode = 20011
	ErrCodeAppConfigSourceRequired ErrorCode = 20012
	ErrCodeAppConfigLoadFailed     ErrorCode = 20013
	ErrCodeAppDomainNotAllowed     ErrorCode = 20014
	ErrCodeAppStatusInvalid        ErrorCode = 20015
	ErrCodeAppQueryFailed          ErrorCode = 20016
	ErrCodeAppNotFoundOrDisabled   ErrorCode = 20017
	ErrCodeAppAccessForbidden      ErrorCode = 20018
	ErrCodeAppGenerateIDFailed     ErrorCode = 20019

	// 21000-21999: 用户
	ErrCodeUserServiceUnavailable       ErrorCode = 21000
	ErrCodeUserListFailed               ErrorCode = 21001
	ErrCodeUserListByIDsFailed          ErrorCode = 21002
	ErrCodeUserBatchActiveInvalidParams ErrorCode = 21003
	ErrCodeUserBatchActiveFailed        ErrorCode = 21004
	ErrCodeUserDeleteInvalidID          ErrorCode = 21005
	ErrCodeUserDeleteFailed             ErrorCode = 21006
	ErrCodeUserUpdateInvalidParams      ErrorCode = 21007
	ErrCodeUserUpdateNotFound           ErrorCode = 21008
	ErrCodeUserUpdateFailed             ErrorCode = 21009
	ErrCodeUserCreateInvalidParams      ErrorCode = 21010
	ErrCodeUserCreateFailed             ErrorCode = 21011
	ErrCodeUserStatusInvalidParams      ErrorCode = 21012
	ErrCodeUserStatusInvalidValue       ErrorCode = 21013
	ErrCodeUserStatusUpdateFailed       ErrorCode = 21014
	ErrCodeUserStatusQueryFailed        ErrorCode = 21015
	ErrCodeUserInfoNotFound             ErrorCode = 21016
	ErrCodeUserProfileInvalidParams     ErrorCode = 21017
	ErrCodeUserProfileUpdateFailed      ErrorCode = 21018
	ErrCodeUserPasswordInvalidParams    ErrorCode = 21019
	ErrCodeUserPasswordTooShort         ErrorCode = 21020
	ErrCodeUserPasswordCurrentInvalid   ErrorCode = 21021
	ErrCodeUserPasswordHashFailed       ErrorCode = 21022
	ErrCodeUserPasswordChangeFailed     ErrorCode = 21023
	ErrCodeUserIDInvalid                ErrorCode = 21024
	ErrCodeUserNotFound                 ErrorCode = 21025
	ErrCodeUserRoleInvalid              ErrorCode = 21026
	ErrCodeUserAppsInvalid              ErrorCode = 21027
	ErrCodeUserAlreadyExists            ErrorCode = 21028
	ErrCodeUserOperationNotAllowed      ErrorCode = 21029

	// 22000-22999: 会话
	ErrCodeSessionServiceUnavailable    ErrorCode = 22000
	ErrCodeSessionListFailed            ErrorCode = 22001
	ErrCodeSessionIDRequired            ErrorCode = 22002
	ErrCodeSessionNotFound              ErrorCode = 22003
	ErrCodeSessionAccessDenied          ErrorCode = 22004
	ErrCodeSessionAlreadyAssigned       ErrorCode = 22005
	ErrCodeSessionClosed                ErrorCode = 22006
	ErrCodeSessionAcceptInvalidParams   ErrorCode = 22007
	ErrCodeSessionAcceptFailed          ErrorCode = 22008
	ErrCodeSessionTransferInvalidParams ErrorCode = 22009
	ErrCodeSessionTransferTargetInvalid ErrorCode = 22010
	ErrCodeSessionTransferFailed        ErrorCode = 22011
	ErrCodeSessionCloseInvalidParams    ErrorCode = 22012
	ErrCodeSessionCloseFailed           ErrorCode = 22013
	ErrCodeSessionReadInvalidParams     ErrorCode = 22014
	ErrCodeSessionReadFailed            ErrorCode = 22015
	ErrCodeSessionRateInvalidParams     ErrorCode = 22016
	ErrCodeSessionRateInvalidScore      ErrorCode = 22017
	ErrCodeSessionRateFailed            ErrorCode = 22018
	ErrCodeSessionAgentsListFailed      ErrorCode = 22019
	ErrCodeSessionMessageQueryInvalid   ErrorCode = 22020
	ErrCodeSessionMessageQueryFailed    ErrorCode = 22021
	ErrCodeSessionAssignDenied          ErrorCode = 22022
	ErrCodeSessionOwnerMismatch         ErrorCode = 22023
	ErrCodeSessionForceCloseDenied      ErrorCode = 22024
	ErrCodeSessionFilterInvalid         ErrorCode = 22025
	ErrCodeSessionRateAlreadyDone       ErrorCode = 22026

	// 23000-23999: 消息
	ErrCodeMessageServiceUnavailable   ErrorCode = 23000
	ErrCodeMessageSaveFailed           ErrorCode = 23001
	ErrCodeMessageParsePayloadFailed   ErrorCode = 23002
	ErrCodeMessageContentTypeInvalid   ErrorCode = 23003
	ErrCodeMessageTextTooLong          ErrorCode = 23004
	ErrCodeMessageMediaURLInvalid      ErrorCode = 23005
	ErrCodeMessageFileTooLarge         ErrorCode = 23006
	ErrCodeMessageFileNameInvalid      ErrorCode = 23007
	ErrCodeMessageAudioDurationInvalid ErrorCode = 23008
	ErrCodeMessageFetchFailed          ErrorCode = 23009
	ErrCodeMessageCursorInvalid        ErrorCode = 23010
	ErrCodeMessageSnapshotInvalid      ErrorCode = 23011
	ErrCodeMessageBuildPayloadFailed   ErrorCode = 23012
	ErrCodeMessageTypeUnsupported      ErrorCode = 23013
	ErrCodeMessagePersistFailed        ErrorCode = 23014
	ErrCodeMessagePushFailed           ErrorCode = 23015

	// 24000-24999: 上传
	ErrCodeUploadUnauthorized        ErrorCode = 24000
	ErrCodeUploadRoleForbidden       ErrorCode = 24001
	ErrCodeUploadMultipartInvalid    ErrorCode = 24002
	ErrCodeUploadAppIDRequired       ErrorCode = 24003
	ErrCodeUploadAppNotFound         ErrorCode = 24004
	ErrCodeUploadDomainForbidden     ErrorCode = 24005
	ErrCodeUploadFileRequired        ErrorCode = 24006
	ErrCodeUploadOpenFileFailed      ErrorCode = 24007
	ErrCodeUploadValidateFailed      ErrorCode = 24008
	ErrCodeUploadMkdirFailed         ErrorCode = 24009
	ErrCodeUploadCreateFileFailed    ErrorCode = 24010
	ErrCodeUploadWriteFileFailed     ErrorCode = 24011
	ErrCodeUploadSaveFailed          ErrorCode = 24012
	ErrCodeUploadEmptyFile           ErrorCode = 24013
	ErrCodeUploadFileTooLarge        ErrorCode = 24014
	ErrCodeUploadFileExtForbidden    ErrorCode = 24015
	ErrCodeUploadMediaTypeInvalid    ErrorCode = 24016
	ErrCodeUploadContentTypeInvalid  ErrorCode = 24017
	ErrCodeUploadPathInvalid         ErrorCode = 24018
	ErrCodeUploadAppNotFoundDisabled ErrorCode = 24019

	// 25000-25999: 快捷回复
	ErrCodeQuickReplyListFailed          ErrorCode = 25000
	ErrCodeQuickReplyCreateInvalidParams ErrorCode = 25001
	ErrCodeQuickReplyCreateFailed        ErrorCode = 25002
	ErrCodeQuickReplyUpdateInvalidParams ErrorCode = 25003
	ErrCodeQuickReplyNotFound            ErrorCode = 25004
	ErrCodeQuickReplyUpdateFailed        ErrorCode = 25005
	ErrCodeQuickReplyDeleteInvalidParams ErrorCode = 25006
	ErrCodeQuickReplyDeleteFailed        ErrorCode = 25007
	ErrCodeQuickReplyUseInvalidParams    ErrorCode = 25008
	ErrCodeQuickReplyUseFailed           ErrorCode = 25009
	ErrCodeQuickReplyOwnerMismatch       ErrorCode = 25010
	ErrCodeQuickReplyCategoryInvalid     ErrorCode = 25011
	ErrCodeQuickReplyContentTooLong      ErrorCode = 25012
	ErrCodeQuickReplyTitleTooLong        ErrorCode = 25013

	// 26000-26999: 审计
	ErrCodeAuditListFailed         ErrorCode = 26000
	ErrCodeAuditCountFailed        ErrorCode = 26001
	ErrCodeAuditWriteFailed        ErrorCode = 26002
	ErrCodeAuditQueryInvalidParams ErrorCode = 26003
	ErrCodeAuditExportFailed       ErrorCode = 26004

	// 27000-27999: 管理后台
	ErrCodeAdminDashboardFailed            ErrorCode = 27000
	ErrCodeAdminVisitorsFailed             ErrorCode = 27001
	ErrCodeAdminUserStatsFailed            ErrorCode = 27002
	ErrCodeAdminSettingsInvalidParams      ErrorCode = 27003
	ErrCodeAdminSettingsSaveFailed         ErrorCode = 27004
	ErrCodeAdminProfileSummaryUnauthorized ErrorCode = 27005
	ErrCodeAdminProfileSummaryFailed       ErrorCode = 27006
	ErrCodeAdminExportFailed               ErrorCode = 27007
	ErrCodeAdminExportCSVFailed            ErrorCode = 27008
	ErrCodeAdminForbidden                  ErrorCode = 27009

	// 28000-28999: 系统设置
	ErrCodeSystemSettingsLoadFailed      ErrorCode = 28000
	ErrCodeSystemSettingsParseFailed     ErrorCode = 28001
	ErrCodeSystemSettingsNormalizeFailed ErrorCode = 28002
	ErrCodeSystemSettingsSaveFailed      ErrorCode = 28003
	ErrCodeSystemSettingsInvalid         ErrorCode = 28004

	// 29000-29999: WebSocket
	ErrCodeWSVisitorInvalidParams  ErrorCode = 29000
	ErrCodeWSVisitorOriginDenied   ErrorCode = 29001
	ErrCodeWSVisitorSessionFailed  ErrorCode = 29002
	ErrCodeWSVisitorAcceptFailed   ErrorCode = 29003
	ErrCodeWSAgentUnauthorized     ErrorCode = 29004
	ErrCodeWSAgentForbidden        ErrorCode = 29005
	ErrCodeWSAgentAcceptFailed     ErrorCode = 29006
	ErrCodeWSMessageInvalidPayload ErrorCode = 29007
	ErrCodeWSMessageHandleFailed   ErrorCode = 29008
	ErrCodeWSConnectionClosed      ErrorCode = 29009

	// 30000-30999: 存储/基础设施
	ErrCodeStorageInitFailed   ErrorCode = 30000
	ErrCodeDBInitFailed        ErrorCode = 30001
	ErrCodeDBQueryFailed       ErrorCode = 30002
	ErrCodeDBWriteFailed       ErrorCode = 30003
	ErrCodeDBTransactionFailed ErrorCode = 30004
	ErrCodeCacheReadFailed     ErrorCode = 30005
	ErrCodeCacheWriteFailed    ErrorCode = 30006
	ErrCodeBadgerReadFailed    ErrorCode = 30007
	ErrCodeBadgerWriteFailed   ErrorCode = 30008
	ErrCodeStorageCorrupted    ErrorCode = 30009

	// 31000-31999: 知识库工作区
	ErrCodeKnowledgeBaseListFailed         ErrorCode = 31000
	ErrCodeKnowledgeBaseCreateInvalid      ErrorCode = 31001
	ErrCodeKnowledgeBaseCreateFailed       ErrorCode = 31002
	ErrCodeKnowledgeBaseNotFound           ErrorCode = 31003
	ErrCodeKnowledgeBaseUpdateInvalid      ErrorCode = 31004
	ErrCodeKnowledgeBaseUpdateFailed       ErrorCode = 31005
	ErrCodeKnowledgeBaseDeleteInvalid      ErrorCode = 31006
	ErrCodeKnowledgeBaseDeleteFailed       ErrorCode = 31007
	ErrCodeKnowledgeBaseAppAccessDenied    ErrorCode = 31008
	ErrCodeKnowledgeVectorCollectionFailed ErrorCode = 31009

	ErrCodeKnowledgeDocumentListFailed    ErrorCode = 31100
	ErrCodeKnowledgeDocumentUploadInvalid ErrorCode = 31101
	ErrCodeKnowledgeDocumentSaveFailed    ErrorCode = 31102
	ErrCodeKnowledgeDocumentParseFailed   ErrorCode = 31103
	ErrCodeKnowledgeDocumentIndexFailed   ErrorCode = 31104
	ErrCodeKnowledgeDocumentNotFound      ErrorCode = 31105
	ErrCodeKnowledgeDocumentDeleteFailed  ErrorCode = 31106
	ErrCodeKnowledgeDocumentReindexFailed ErrorCode = 31107
	ErrCodeKnowledgeDocumentFormatInvalid ErrorCode = 31108

	ErrCodeKnowledgeChunkListFailed    ErrorCode = 31200
	ErrCodeKnowledgeChunkUpdateInvalid ErrorCode = 31201
	ErrCodeKnowledgeChunkUpdateFailed  ErrorCode = 31202
	ErrCodeKnowledgeChunkDeleteFailed  ErrorCode = 31203
	ErrCodeKnowledgeChunkNotFound      ErrorCode = 31204

	ErrCodeKnowledgeRetrieveInvalid ErrorCode = 31300
	ErrCodeKnowledgeRetrieveFailed  ErrorCode = 31301

	ErrCodeKnowledgeQAInvalid        ErrorCode = 31400
	ErrCodeKnowledgeQAFailed         ErrorCode = 31401
	ErrCodeKnowledgeFeedbackInvalid  ErrorCode = 31402
	ErrCodeKnowledgeFeedbackSaveFail ErrorCode = 31403

	// 31500-31599: 知识库模型配置与推理
	ErrCodeKnowledgeModelListFailed      ErrorCode = 31500
	ErrCodeKnowledgeModelSaveInvalid     ErrorCode = 31501
	ErrCodeKnowledgeModelSaveFailed      ErrorCode = 31502
	ErrCodeKnowledgeModelProfileNotFound ErrorCode = 31503
	ErrCodeKnowledgeModelProfileInvalid  ErrorCode = 31504
	ErrCodeKnowledgeModelSwitchFailed    ErrorCode = 31505
	ErrCodeKnowledgeModelInferFailed     ErrorCode = 31506
	ErrCodeKnowledgeModelBinaryNotFound  ErrorCode = 31507
	ErrCodeKnowledgeModelPathInvalid     ErrorCode = 31508
	ErrCodeKnowledgeModelProviderInvalid ErrorCode = 31509

	// 兼容旧错误码别名
	ErrCodeUnauthorized       = ErrCodeAuthUnauthorized
	ErrCodeForbidden          = ErrCodePermissionForbidden
	ErrCodeInvalidCredentials = ErrCodeAuthInvalidCredentials
	ErrCodeTokenExpired       = ErrCodeAuthTokenExpired
	ErrCodeTokenInvalid       = ErrCodeAuthTokenInvalid
	ErrCodeTooManyRequests    = ErrCodeSecurityTooManyRequests
)

var ErrorMessages = map[ErrorCode]string{
	ErrCodeSuccess:                         "success",
	ErrCodeUnknown:                         "unknown error",
	ErrCodeInvalidParams:                   "invalid parameters",
	ErrCodeBadRequest:                      "bad request",
	ErrCodeNotFound:                        "resource not found",
	ErrCodeConflict:                        "resource conflict",
	ErrCodeInternalError:                   "internal server error",
	ErrCodeServiceUnavailable:              "service unavailable",
	ErrCodeMethodNotAllowed:                "method not allowed",
	ErrCodeRequestTimeout:                  "request timeout",
	ErrCodeUnsupportedMediaType:            "unsupported media type",
	ErrCodePayloadTooLarge:                 "payload too large",
	ErrCodeIllegalState:                    "illegal state",
	ErrCodeDataCorrupted:                   "data corrupted",
	ErrCodeOperationFailed:                 "operation failed",
	ErrCodeInvalidCursor:                   "invalid cursor",
	ErrCodeInvalidSnapshot:                 "invalid snapshot",
	ErrCodeAuthUnauthorized:                "unauthorized",
	ErrCodeAuthTokenRequired:               "authorization token required",
	ErrCodeAuthTokenFormatInvalid:          "invalid authorization token format",
	ErrCodeAuthTokenInvalid:                "invalid token",
	ErrCodeAuthTokenExpired:                "token expired",
	ErrCodeAuthTokenRevoked:                "token revoked",
	ErrCodeAuthTokenClaimsInvalid:          "token claims invalid",
	ErrCodeAuthContextMissing:              "auth context missing",
	ErrCodeAuthInvalidCredentials:          "invalid username or password",
	ErrCodeAuthLoginInvalidParams:          "invalid login parameters",
	ErrCodeAuthLoginUserNotFound:           "login user not found",
	ErrCodeAuthLoginUserDisabled:           "login user disabled",
	ErrCodeAuthTokenGenerateFailed:         "generate token failed",
	ErrCodeAuthPasswordTooWeak:             "password too weak",
	ErrCodeAuthPasswordInvalid:             "password invalid",
	ErrCodeAuthPasswordPolicy:              "password does not meet policy",
	ErrCodePermissionForbidden:             "forbidden",
	ErrCodePermissionRoleDenied:            "role denied",
	ErrCodePermissionAdminRequired:         "admin role required",
	ErrCodePermissionAgentRequired:         "agent role required",
	ErrCodePermissionSuperAdminRequired:    "super admin role required",
	ErrCodePermissionAppAccessDenied:       "app access denied",
	ErrCodePermissionSessionDenied:         "session access denied",
	ErrCodePermissionOwnershipMismatch:     "resource ownership mismatch",
	ErrCodePermissionOperationDenied:       "operation denied",
	ErrCodePermissionCrossAppDenied:        "cross app access denied",
	ErrCodeSecurityTooManyRequests:         "too many requests",
	ErrCodeSecurityRateLimitExceeded:       "rate limit exceeded",
	ErrCodeSecurityCSRFInvalid:             "invalid csrf token",
	ErrCodeSecurityOriginNotAllowed:        "origin not allowed",
	ErrCodeSecurityDomainNotAllowed:        "domain not allowed",
	ErrCodeSecurityCaptchaRequired:         "captcha required",
	ErrCodeSecurityCaptchaInvalid:          "captcha invalid",
	ErrCodeSecuritySensitiveDetected:       "sensitive content detected",
	ErrCodeSecurityIPBlocked:               "ip blocked",
	ErrCodeSecurityRiskControlRejected:     "risk control rejected",
	ErrCodeAppListFailed:                   "list apps failed",
	ErrCodeAppCreateInvalidParams:          "invalid app create parameters",
	ErrCodeAppIDDuplicated:                 "app id duplicated",
	ErrCodeAppCreateFailed:                 "create app failed",
	ErrCodeAppNotFound:                     "app not found",
	ErrCodeAppUpdateInvalidParams:          "invalid app update parameters",
	ErrCodeAppUpdateNoChanges:              "no app fields to update",
	ErrCodeAppUpdateFailed:                 "update app failed",
	ErrCodeAppDeleteInvalidParams:          "invalid app delete parameters",
	ErrCodeAppDeleteFailed:                 "delete app failed",
	ErrCodeAppDisabled:                     "app is disabled",
	ErrCodeAppConfigInvalidParams:          "invalid app config parameters",
	ErrCodeAppConfigSourceRequired:         "origin or referer required",
	ErrCodeAppConfigLoadFailed:             "load app config failed",
	ErrCodeAppDomainNotAllowed:             "app domain not allowed",
	ErrCodeAppStatusInvalid:                "invalid app status",
	ErrCodeAppQueryFailed:                  "query app failed",
	ErrCodeAppNotFoundOrDisabled:           "app not found or disabled",
	ErrCodeAppAccessForbidden:              "app access forbidden",
	ErrCodeAppGenerateIDFailed:             "generate app id failed",
	ErrCodeUserServiceUnavailable:          "user service unavailable",
	ErrCodeUserListFailed:                  "list users failed",
	ErrCodeUserListByIDsFailed:             "list users by ids failed",
	ErrCodeUserBatchActiveInvalidParams:    "invalid batch active parameters",
	ErrCodeUserBatchActiveFailed:           "batch active update failed",
	ErrCodeUserDeleteInvalidID:             "invalid user id",
	ErrCodeUserDeleteFailed:                "delete user failed",
	ErrCodeUserUpdateInvalidParams:         "invalid user update parameters",
	ErrCodeUserUpdateNotFound:              "user not found for update",
	ErrCodeUserUpdateFailed:                "update user failed",
	ErrCodeUserCreateInvalidParams:         "invalid user create parameters",
	ErrCodeUserCreateFailed:                "create user failed",
	ErrCodeUserStatusInvalidParams:         "invalid user status parameters",
	ErrCodeUserStatusInvalidValue:          "invalid user status value",
	ErrCodeUserStatusUpdateFailed:          "update user status failed",
	ErrCodeUserStatusQueryFailed:           "query user status failed",
	ErrCodeUserInfoNotFound:                "user info not found",
	ErrCodeUserProfileInvalidParams:        "invalid profile parameters",
	ErrCodeUserProfileUpdateFailed:         "update profile failed",
	ErrCodeUserPasswordInvalidParams:       "invalid password parameters",
	ErrCodeUserPasswordTooShort:            "new password too short",
	ErrCodeUserPasswordCurrentInvalid:      "current password invalid",
	ErrCodeUserPasswordHashFailed:          "hash password failed",
	ErrCodeUserPasswordChangeFailed:        "change password failed",
	ErrCodeUserIDInvalid:                   "invalid user id",
	ErrCodeUserNotFound:                    "user not found",
	ErrCodeUserRoleInvalid:                 "invalid user role",
	ErrCodeUserAppsInvalid:                 "invalid user apps",
	ErrCodeUserAlreadyExists:               "user already exists",
	ErrCodeUserOperationNotAllowed:         "user operation not allowed",
	ErrCodeSessionServiceUnavailable:       "session service unavailable",
	ErrCodeSessionListFailed:               "list sessions failed",
	ErrCodeSessionIDRequired:               "session id required",
	ErrCodeSessionNotFound:                 "session not found",
	ErrCodeSessionAccessDenied:             "session access denied",
	ErrCodeSessionAlreadyAssigned:          "session already assigned",
	ErrCodeSessionClosed:                   "session closed",
	ErrCodeSessionAcceptInvalidParams:      "invalid session accept parameters",
	ErrCodeSessionAcceptFailed:             "accept session failed",
	ErrCodeSessionTransferInvalidParams:    "invalid session transfer parameters",
	ErrCodeSessionTransferTargetInvalid:    "invalid transfer target agent",
	ErrCodeSessionTransferFailed:           "transfer session failed",
	ErrCodeSessionCloseInvalidParams:       "invalid session close parameters",
	ErrCodeSessionCloseFailed:              "close session failed",
	ErrCodeSessionReadInvalidParams:        "invalid mark read parameters",
	ErrCodeSessionReadFailed:               "mark session read failed",
	ErrCodeSessionRateInvalidParams:        "invalid session rating parameters",
	ErrCodeSessionRateInvalidScore:         "invalid session rating score",
	ErrCodeSessionRateFailed:               "rate session failed",
	ErrCodeSessionAgentsListFailed:         "list agents failed",
	ErrCodeSessionMessageQueryInvalid:      "invalid message query parameters",
	ErrCodeSessionMessageQueryFailed:       "query session messages failed",
	ErrCodeSessionAssignDenied:             "session assign denied",
	ErrCodeSessionOwnerMismatch:            "session owner mismatch",
	ErrCodeSessionForceCloseDenied:         "session force close denied",
	ErrCodeSessionFilterInvalid:            "invalid session filter",
	ErrCodeSessionRateAlreadyDone:          "session already rated",
	ErrCodeMessageServiceUnavailable:       "message service unavailable",
	ErrCodeMessageSaveFailed:               "save message failed",
	ErrCodeMessageParsePayloadFailed:       "parse message payload failed",
	ErrCodeMessageContentTypeInvalid:       "invalid message content type",
	ErrCodeMessageTextTooLong:              "message text too long",
	ErrCodeMessageMediaURLInvalid:          "invalid message media url",
	ErrCodeMessageFileTooLarge:             "message file too large",
	ErrCodeMessageFileNameInvalid:          "invalid message file name",
	ErrCodeMessageAudioDurationInvalid:     "invalid message audio duration",
	ErrCodeMessageFetchFailed:              "fetch messages failed",
	ErrCodeMessageCursorInvalid:            "invalid message cursor",
	ErrCodeMessageSnapshotInvalid:          "invalid message snapshot",
	ErrCodeMessageBuildPayloadFailed:       "build message payload failed",
	ErrCodeMessageTypeUnsupported:          "unsupported message type",
	ErrCodeMessagePersistFailed:            "persist message failed",
	ErrCodeMessagePushFailed:               "push message failed",
	ErrCodeUploadUnauthorized:              "upload unauthorized",
	ErrCodeUploadRoleForbidden:             "upload role forbidden",
	ErrCodeUploadMultipartInvalid:          "invalid multipart body",
	ErrCodeUploadAppIDRequired:             "upload app_id required",
	ErrCodeUploadAppNotFound:               "upload app not found",
	ErrCodeUploadDomainForbidden:           "upload domain forbidden",
	ErrCodeUploadFileRequired:              "upload file required",
	ErrCodeUploadOpenFileFailed:            "open upload file failed",
	ErrCodeUploadValidateFailed:            "validate upload file failed",
	ErrCodeUploadMkdirFailed:               "create upload directory failed",
	ErrCodeUploadCreateFileFailed:          "create upload file failed",
	ErrCodeUploadWriteFileFailed:           "write upload file failed",
	ErrCodeUploadSaveFailed:                "save upload file failed",
	ErrCodeUploadEmptyFile:                 "empty upload file",
	ErrCodeUploadFileTooLarge:              "upload file too large",
	ErrCodeUploadFileExtForbidden:          "upload file extension forbidden",
	ErrCodeUploadMediaTypeInvalid:          "invalid upload media type",
	ErrCodeUploadContentTypeInvalid:        "invalid upload content type",
	ErrCodeUploadPathInvalid:               "invalid upload path",
	ErrCodeUploadAppNotFoundDisabled:       "upload app not found or disabled",
	ErrCodeQuickReplyListFailed:            "list quick replies failed",
	ErrCodeQuickReplyCreateInvalidParams:   "invalid quick reply create parameters",
	ErrCodeQuickReplyCreateFailed:          "create quick reply failed",
	ErrCodeQuickReplyUpdateInvalidParams:   "invalid quick reply update parameters",
	ErrCodeQuickReplyNotFound:              "quick reply not found",
	ErrCodeQuickReplyUpdateFailed:          "update quick reply failed",
	ErrCodeQuickReplyDeleteInvalidParams:   "invalid quick reply delete parameters",
	ErrCodeQuickReplyDeleteFailed:          "delete quick reply failed",
	ErrCodeQuickReplyUseInvalidParams:      "invalid quick reply use parameters",
	ErrCodeQuickReplyUseFailed:             "update quick reply usage failed",
	ErrCodeQuickReplyOwnerMismatch:         "quick reply owner mismatch",
	ErrCodeQuickReplyCategoryInvalid:       "invalid quick reply category",
	ErrCodeQuickReplyContentTooLong:        "quick reply content too long",
	ErrCodeQuickReplyTitleTooLong:          "quick reply title too long",
	ErrCodeAuditListFailed:                 "list audit logs failed",
	ErrCodeAuditCountFailed:                "count audit logs failed",
	ErrCodeAuditWriteFailed:                "write audit log failed",
	ErrCodeAuditQueryInvalidParams:         "invalid audit query parameters",
	ErrCodeAuditExportFailed:               "export audit logs failed",
	ErrCodeAdminDashboardFailed:            "load dashboard failed",
	ErrCodeAdminVisitorsFailed:             "load visitors failed",
	ErrCodeAdminUserStatsFailed:            "load user stats failed",
	ErrCodeAdminSettingsInvalidParams:      "invalid admin settings parameters",
	ErrCodeAdminSettingsSaveFailed:         "save admin settings failed",
	ErrCodeAdminProfileSummaryUnauthorized: "profile summary unauthorized",
	ErrCodeAdminProfileSummaryFailed:       "load profile summary failed",
	ErrCodeAdminExportFailed:               "export sessions failed",
	ErrCodeAdminExportCSVFailed:            "build csv export failed",
	ErrCodeAdminForbidden:                  "admin access forbidden",
	ErrCodeSystemSettingsLoadFailed:        "load system settings failed",
	ErrCodeSystemSettingsParseFailed:       "parse system settings failed",
	ErrCodeSystemSettingsNormalizeFailed:   "normalize system settings failed",
	ErrCodeSystemSettingsSaveFailed:        "save system settings failed",
	ErrCodeSystemSettingsInvalid:           "invalid system settings",
	ErrCodeWSVisitorInvalidParams:          "invalid visitor websocket parameters",
	ErrCodeWSVisitorOriginDenied:           "visitor websocket origin denied",
	ErrCodeWSVisitorSessionFailed:          "visitor websocket session failed",
	ErrCodeWSVisitorAcceptFailed:           "visitor websocket accept failed",
	ErrCodeWSAgentUnauthorized:             "agent websocket unauthorized",
	ErrCodeWSAgentForbidden:                "agent websocket forbidden",
	ErrCodeWSAgentAcceptFailed:             "agent websocket accept failed",
	ErrCodeWSMessageInvalidPayload:         "invalid websocket message payload",
	ErrCodeWSMessageHandleFailed:           "handle websocket message failed",
	ErrCodeWSConnectionClosed:              "websocket connection closed",
	ErrCodeStorageInitFailed:               "storage initialization failed",
	ErrCodeDBInitFailed:                    "database initialization failed",
	ErrCodeDBQueryFailed:                   "database query failed",
	ErrCodeDBWriteFailed:                   "database write failed",
	ErrCodeDBTransactionFailed:             "database transaction failed",
	ErrCodeCacheReadFailed:                 "cache read failed",
	ErrCodeCacheWriteFailed:                "cache write failed",
	ErrCodeBadgerReadFailed:                "badger read failed",
	ErrCodeBadgerWriteFailed:               "badger write failed",
	ErrCodeStorageCorrupted:                "storage data corrupted",
	ErrCodeKnowledgeBaseListFailed:         "list knowledge bases failed",
	ErrCodeKnowledgeBaseCreateInvalid:      "invalid knowledge base create parameters",
	ErrCodeKnowledgeBaseCreateFailed:       "create knowledge base failed",
	ErrCodeKnowledgeBaseNotFound:           "knowledge base not found",
	ErrCodeKnowledgeBaseUpdateInvalid:      "invalid knowledge base update parameters",
	ErrCodeKnowledgeBaseUpdateFailed:       "update knowledge base failed",
	ErrCodeKnowledgeBaseDeleteInvalid:      "invalid knowledge base delete parameters",
	ErrCodeKnowledgeBaseDeleteFailed:       "delete knowledge base failed",
	ErrCodeKnowledgeBaseAppAccessDenied:    "knowledge base app access denied",
	ErrCodeKnowledgeVectorCollectionFailed: "knowledge vector collection operation failed",
	ErrCodeKnowledgeDocumentListFailed:     "list knowledge documents failed",
	ErrCodeKnowledgeDocumentUploadInvalid:  "invalid knowledge document upload parameters",
	ErrCodeKnowledgeDocumentSaveFailed:     "save knowledge document failed",
	ErrCodeKnowledgeDocumentParseFailed:    "parse knowledge document failed",
	ErrCodeKnowledgeDocumentIndexFailed:    "index knowledge document failed",
	ErrCodeKnowledgeDocumentNotFound:       "knowledge document not found",
	ErrCodeKnowledgeDocumentDeleteFailed:   "delete knowledge document failed",
	ErrCodeKnowledgeDocumentReindexFailed:  "reindex knowledge document failed",
	ErrCodeKnowledgeDocumentFormatInvalid:  "knowledge document format not supported",
	ErrCodeKnowledgeChunkListFailed:        "list knowledge chunks failed",
	ErrCodeKnowledgeChunkUpdateInvalid:     "invalid knowledge chunk update parameters",
	ErrCodeKnowledgeChunkUpdateFailed:      "update knowledge chunk failed",
	ErrCodeKnowledgeChunkDeleteFailed:      "delete knowledge chunk failed",
	ErrCodeKnowledgeChunkNotFound:          "knowledge chunk not found",
	ErrCodeKnowledgeRetrieveInvalid:        "invalid knowledge retrieve parameters",
	ErrCodeKnowledgeRetrieveFailed:         "knowledge retrieve failed",
	ErrCodeKnowledgeQAInvalid:              "invalid knowledge qa parameters",
	ErrCodeKnowledgeQAFailed:               "knowledge qa failed",
	ErrCodeKnowledgeFeedbackInvalid:        "invalid knowledge feedback parameters",
	ErrCodeKnowledgeFeedbackSaveFail:       "save knowledge feedback failed",
	ErrCodeKnowledgeModelListFailed:        "list knowledge model settings failed",
	ErrCodeKnowledgeModelSaveInvalid:       "invalid knowledge model settings parameters",
	ErrCodeKnowledgeModelSaveFailed:        "save knowledge model settings failed",
	ErrCodeKnowledgeModelProfileNotFound:   "knowledge model profile not found",
	ErrCodeKnowledgeModelProfileInvalid:    "knowledge model profile invalid",
	ErrCodeKnowledgeModelSwitchFailed:      "switch knowledge model failed",
	ErrCodeKnowledgeModelInferFailed:       "knowledge model inference failed",
	ErrCodeKnowledgeModelBinaryNotFound:    "knowledge model runtime binary not found",
	ErrCodeKnowledgeModelPathInvalid:       "knowledge model path invalid",
	ErrCodeKnowledgeModelProviderInvalid:   "knowledge model provider invalid",
}

// MessageForCode returns human-readable default message for code.
func MessageForCode(code ErrorCode) string {
	if msg, ok := ErrorMessages[code]; ok && msg != "" {
		return msg
	}
	return ErrorMessages[ErrCodeUnknown]
}

// 审计领域服务：登录日志、操作日志、审计日志写入
package audit

import (
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// LogLogin 记录登录日志
func (s *Service) LogLogin(userID *int64, username string, success bool, failReason, ip, userAgent string) {
	result := int16(2)
	if success {
		result = 1
	}
	s.db.Exec(`
		INSERT INTO sys_login_log (user_id, username, login_result, fail_reason, ip_address, user_agent, login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, username, result, strOrNil(failReason), ip, strOrNil(userAgent), time.Now())
}

// LogAudit 记录敏感操作审计日志
func (s *Service) LogAudit(userID *int64, module, action, bizType string, bizID *int64, before, after, ip string) {
	s.db.Exec(`
		INSERT INTO audit_log (user_id, module, action, biz_type, biz_id, before_data, after_data, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, module, action, bizType, bizID, strOrNil(before), strOrNil(after), strOrNil(ip), time.Now())
}

// LogOperation 记录操作日志
func (s *Service) LogOperation(userID *int64, module, action, path, method, body string, respCode int, success bool) {
	s.db.Exec(`
		INSERT INTO sys_operation_log (user_id, module, action, request_path, request_method, request_body, response_code, success, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, module, action, path, method, strOrNil(body), respCode, success, time.Now())
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

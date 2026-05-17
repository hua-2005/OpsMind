// 认证领域服务：登录、令牌刷新、退出、密码校验、JWT 签发
package auth

import (
	"fmt"
	"time"

	"opsmind/server/internal/config"
	"opsmind/server/internal/model/entity"
	"opsmind/server/internal/pkg/errors"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewService(db *gorm.DB, cfg *config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

// Login 验证用户名密码，返回用户、角色和权限
func (s *Service) Login(username, password string) (*entity.SysUser, []string, []string, error) {
	var user entity.SysUser
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, nil, nil, errors.New(errors.CodeUnauthorized, "账号或密码错误")
	}

	if user.Status == 2 {
		return nil, nil, nil, errors.New(errors.CodeAccountFrozen, "账号已冻结，请联系管理员")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, nil, errors.New(errors.CodeUnauthorized, "账号或密码错误")
	}

	// 角色编码 — 只取启用状态的角色
	var roleCodes []string
	s.db.Raw(`
		SELECT r.code FROM sys_role r
		INNER JOIN sys_user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ? AND r.status = 1
	`, user.ID).Scan(&roleCodes)

	// 权限编码 — 管理员返回全部
	var permCodes []string
	if containsAdmin(roleCodes) {
		s.db.Raw(`SELECT code FROM sys_permission WHERE status = 1`).Scan(&permCodes)
	} else {
		s.db.Raw(`
			SELECT DISTINCT p.code FROM sys_permission p
			INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
			INNER JOIN sys_role r ON r.id = rp.role_id AND r.status = 1
			INNER JOIN sys_user_role ur ON ur.role_id = r.id
			WHERE ur.user_id = ? AND p.status = 1
		`, user.ID).Scan(&permCodes)
	}

	now := time.Now()
	s.db.Model(&user).Updates(map[string]interface{}{
		"last_login_at": now,
	})

	return &user, roleCodes, permCodes, nil
}

// GenerateToken 签发 JWT access_token
func (s *Service) GenerateToken(user *entity.SysUser) (string, int64, error) {
	expireHours := s.cfg.JWT.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	expiresAt := time.Now().Add(time.Duration(expireHours) * time.Hour)
	expiresIn := int64(time.Duration(expireHours) * time.Hour / time.Second)

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return tokenStr, expiresIn, nil
}

// ParseToken 解析并校验 JWT，返回 claims
func (s *Service) ParseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// IsAccountActive 校验账号是否正常（未被冻结或软删除），用于中间件校验已签发 token
func (s *Service) IsAccountActive(userID int64) (bool, error) {
	var count int64
	if err := s.db.Raw(
		`SELECT COUNT(*) FROM sys_user WHERE id = ? AND status = 1 AND deleted_at IS NULL`, userID,
	).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetProfile 获取当前用户资料（含角色）
func (s *Service) GetProfile(userID int64) (*entity.SysUser, []string, error) {
	var user entity.SysUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, nil, errors.New(errors.CodeNotFound, "用户不存在")
	}

	var roles []string
	s.db.Raw(`
		SELECT r.code FROM sys_role r
		INNER JOIN sys_user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ? AND r.status = 1
	`, userID).Scan(&roles)

	return &user, roles, nil
}

// GetPermissions 获取当前用户完整权限树 — 管理员返回全部
func (s *Service) GetPermissions(userID int64) (menus []entity.SysPermission, buttons []string, apis []string, err error) {
	admin := s.userIsAdmin(userID)

	if admin {
		s.db.Where("type = 1 AND status = 1 AND visible = true").Order("sort_order").Find(&menus)
		s.db.Model(&entity.SysPermission{}).Where("type = 2 AND status = 1").Pluck("code", &buttons)
		s.db.Model(&entity.SysPermission{}).Where("type = 3 AND status = 1").Pluck("code", &apis)
		return
	}

	s.db.Raw(`
		SELECT DISTINCT p.* FROM sys_permission p
		INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
		INNER JOIN sys_role r ON r.id = rp.role_id AND r.status = 1
		INNER JOIN sys_user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND p.type = 1 AND p.status = 1 AND p.visible = true
		ORDER BY p.sort_order
	`, userID).Scan(&menus)

	s.db.Raw(`
		SELECT DISTINCT p.code FROM sys_permission p
		INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
		INNER JOIN sys_role r ON r.id = rp.role_id AND r.status = 1
		INNER JOIN sys_user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND p.type = 2 AND p.status = 1
	`, userID).Scan(&buttons)

	s.db.Raw(`
		SELECT DISTINCT p.code FROM sys_permission p
		INNER JOIN sys_role_permission rp ON p.id = rp.permission_id
		INNER JOIN sys_role r ON r.id = rp.role_id AND r.status = 1
		INNER JOIN sys_user_role ur ON ur.role_id = r.id
		WHERE ur.user_id = ? AND p.type = 3 AND p.status = 1
	`, userID).Scan(&apis)

	return
}

func (s *Service) userIsAdmin(userID int64) bool {
	var count int64
	s.db.Raw(`
		SELECT COUNT(*) FROM sys_user_role ur
		INNER JOIN sys_role r ON r.id = ur.role_id
		WHERE ur.user_id = ? AND r.code = 'admin' AND r.status = 1
	`, userID).Scan(&count)
	return count > 0
}

func containsAdmin(roles []string) bool {
	for _, r := range roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

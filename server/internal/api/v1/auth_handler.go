// 认证 API：登录、刷新、退出、当前用户、权限
package v1

import (
	"opsmind/server/internal/domain/auth"
	"opsmind/server/internal/domain/audit"
	"opsmind/server/internal/model/entity"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc   *auth.Service
	audit *audit.Service
}

func NewAuthHandler(svc *auth.Service, auditSvc *audit.Service) *AuthHandler {
	return &AuthHandler{svc: svc, audit: auditSvc}
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeInvalidJSON, "请求参数格式错误"))
		return
	}
	if req.Username == "" || req.Password == "" {
		response.Error(c, errors.WithDetails(errors.CodeValidationError, "参数校验失败", []errors.FieldError{
			{Field: "username", Message: "不能为空", Code: "required"},
			{Field: "password", Message: "不能为空", Code: "required"},
		}))
		return
	}

	user, roles, perms, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		// 记录失败登录
		h.audit.LogLogin(nil, req.Username, false, err.Error(), clientIP(c), c.GetHeader("User-Agent"))
		response.Error(c, err.(*errors.AppError))
		return
	}

	token, expiresIn, err := h.svc.GenerateToken(user)
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "令牌签发失败")
		return
	}

	// 记录成功登录
	h.audit.LogLogin(&user.ID, user.Username, true, "", clientIP(c), c.GetHeader("User-Agent"))

	response.OK(c, gin.H{
		"access_token": token,
		"expires_in":   expiresIn,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"real_name": user.RealName,
			"status":    userStatus(user.Status),
		},
		"roles":       roles,
		"permissions": perms,
	})
}

// Refresh POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	userID := c.GetInt64("user_id")
	username := c.GetString("username")
	if userID == 0 {
		response.Error(c, errors.New(errors.CodeUnauthorized, "Token 无效"))
		return
	}
	token, expiresIn, err := h.svc.GenerateToken(&entity.SysUser{ID: userID, Username: username})
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "令牌刷新失败")
		return
	}
	response.OK(c, gin.H{
		"access_token": token,
		"expires_in":   expiresIn,
	})
}

// Logout POST /api/v1/auth/logout — 将当前 token 加入黑名单
func (h *AuthHandler) Logout(c *gin.Context) {
	token := extractToken(c)
	if token != "" {
		auth.BlacklistToken(token)
	}

	// 记录登出审计
	userID := c.GetInt64("user_id")
	h.audit.LogAudit(&userID, "auth", "logout", "sys_user", &userID, "", "", clientIP(c))

	response.NoContent(c)
}

// Profile GET /api/v1/auth/profile
func (h *AuthHandler) Profile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	user, roles, err := h.svc.GetProfile(userID)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	profile := gin.H{
		"id":        user.ID,
		"username":  user.Username,
		"real_name": user.RealName,
		"status":    userStatus(user.Status),
		"roles":     roles,
	}
	if user.Phone != nil {
		profile["phone"] = *user.Phone
	} else {
		profile["phone"] = nil
	}
	if user.Email != nil {
		profile["email"] = *user.Email
	} else {
		profile["email"] = nil
	}

	response.OK(c, profile)
}

// Permissions GET /api/v1/auth/permissions
func (h *AuthHandler) Permissions(c *gin.Context) {
	userID := c.GetInt64("user_id")
	menus, buttons, apis, err := h.svc.GetPermissions(userID)
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "获取权限失败")
		return
	}

	response.OK(c, gin.H{
		"menus":   buildMenuTree(menus),
		"buttons": buttons,
		"apis":    apis,
	})
}

func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || len(authHeader) < 8 {
		return ""
	}
	return authHeader[7:] // "Bearer " = 7 chars
}

func buildMenuTree(perms []entity.SysPermission) []gin.H {
	pmap := make(map[int64][]entity.SysPermission)
	var roots []entity.SysPermission
	for _, p := range perms {
		if p.ParentID == nil {
			roots = append(roots, p)
		} else {
			pmap[*p.ParentID] = append(pmap[*p.ParentID], p)
		}
	}

	var build func(list []entity.SysPermission) []gin.H
	build = func(list []entity.SysPermission) []gin.H {
		var result []gin.H
		for _, p := range list {
			item := gin.H{
				"name":     p.Name,
				"code":     p.Code,
				"path":     p.Path,
				"children": build(pmap[p.ID]),
			}
			result = append(result, item)
		}
		return result
	}
	return build(roots)
}

// 账号管理 API：列表（含角色）、创建、详情、更新、冻结、恢复
package v1

import (
	"strconv"

	"opsmind/server/internal/domain/audit"
	"opsmind/server/internal/domain/user"
	"opsmind/server/internal/model/entity"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"
	"opsmind/server/internal/repository"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	svc   *user.Service
	audit *audit.Service
}

func NewAccountHandler(svc *user.Service, auditSvc *audit.Service) *AccountHandler {
	return &AccountHandler{svc: svc, audit: auditSvc}
}

// List GET /api/v1/admin/accounts — 返回角色信息
func (h *AccountHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	q := c.Query("q")
	status := c.Query("status")
	roleCode := c.Query("role_code")

	users, total, err := h.svc.ListAccounts(page, perPage, q, status, roleCode)
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "查询账号列表失败")
		return
	}

	userIDs := make([]int64, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	roleMap := h.svc.GetUserRoles(userIDs)

	var items []gin.H
	for _, u := range users {
		item := gin.H{
			"id":         u.ID,
			"username":   u.Username,
			"real_name":  u.RealName,
			"status":     userStatus(u.Status),
			"created_at": u.CreatedAt,
		}
		if u.Phone != nil {
			item["phone"] = *u.Phone
		}
		if u.Email != nil {
			item["email"] = *u.Email
		}
		if u.LastLoginAt != nil {
			item["last_login_at"] = u.LastLoginAt
		}
		if roles, ok := roleMap[u.ID]; ok {
			rl := make([]gin.H, len(roles))
			for i, r := range roles {
				rl[i] = gin.H{"code": r.Code, "name": r.Name}
			}
			item["roles"] = rl
		}
		items = append(items, item)
	}

	meta := response.Meta{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: repository.TotalPages(total, perPage),
	}
	response.List(c, items, meta)
}

// Create POST /api/v1/admin/accounts
func (h *AccountHandler) Create(c *gin.Context) {
	var req struct {
		Username string  `json:"username"`
		Password string  `json:"password"`
		RealName string  `json:"real_name"`
		Phone    string  `json:"phone"`
		Email    string  `json:"email"`
		RoleIDs  []int64 `json:"role_ids"`
		Remark   string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeInvalidJSON, "请求参数格式错误"))
		return
	}
	if req.Username == "" || req.Password == "" || req.RealName == "" {
		response.Error(c, errors.WithDetails(errors.CodeValidationError, "参数校验失败", []errors.FieldError{
			{Field: "username", Message: "不能为空", Code: "required"},
			{Field: "password", Message: "不能为空", Code: "required"},
			{Field: "real_name", Message: "不能为空", Code: "required"},
		}))
		return
	}

	u, err := h.svc.CreateAccount(req.Username, req.Password, req.RealName, req.Phone, req.Email, req.RoleIDs, req.Remark)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	opID := c.GetInt64("user_id")
	h.audit.LogAudit(&opID, "account", "create", "sys_user", &u.ID, "", "", clientIP(c))

	response.Created(c, formatAccountItem(u, nil))
}

// Detail GET /api/v1/admin/accounts/:id
func (h *AccountHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeValidationError, "无效的账号 ID"))
		return
	}

	u, roles, err := h.svc.GetAccount(id)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	response.OK(c, formatAccountDetail(u, roles))
}

// Update PATCH /api/v1/admin/accounts/:id
func (h *AccountHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeValidationError, "无效的账号 ID"))
		return
	}

	var req struct {
		RealName string  `json:"real_name"`
		Phone    string  `json:"phone"`
		Email    string  `json:"email"`
		RoleIDs  []int64 `json:"role_ids"`
		Remark   string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeInvalidJSON, "请求参数格式错误"))
		return
	}

	u, err := h.svc.UpdateAccount(id, req.RealName, req.Phone, req.Email, req.Remark, req.RoleIDs)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	response.OK(c, gin.H{
		"id":         u.ID,
		"real_name":  u.RealName,
		"updated_at": u.UpdatedAt,
	})
}

// Freeze POST /api/v1/admin/accounts/:id/freeze — 含审计
func (h *AccountHandler) Freeze(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	operatorID := c.GetInt64("user_id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	if req.Reason == "" {
		response.Error(c, errors.WithDetails(errors.CodeValidationError, "冻结原因不能为空", []errors.FieldError{
			{Field: "reason", Message: "不能为空", Code: "required"},
		}))
		return
	}

	u, err := h.svc.FreezeAccount(id, operatorID, req.Reason)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	h.audit.LogAudit(&operatorID, "account", "freeze", "sys_user", &id, `"active"`, `"frozen"`, clientIP(c))

	response.OK(c, gin.H{
		"id":         u.ID,
		"status":     "frozen",
		"updated_at": u.UpdatedAt,
	})
}

// Restore POST /api/v1/admin/accounts/:id/restore — 含审计
func (h *AccountHandler) Restore(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	operatorID := c.GetInt64("user_id")

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	if req.Reason == "" {
		response.Error(c, errors.WithDetails(errors.CodeValidationError, "恢复原因不能为空", []errors.FieldError{
			{Field: "reason", Message: "不能为空", Code: "required"},
		}))
		return
	}

	u, err := h.svc.RestoreAccount(id, req.Reason)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	h.audit.LogAudit(&operatorID, "account", "restore", "sys_user", &id, `"frozen"`, `"active"`, clientIP(c))

	response.OK(c, gin.H{
		"id":         u.ID,
		"status":     "active",
		"updated_at": u.UpdatedAt,
	})
}

func formatAccountItem(u *entity.SysUser, roles []entity.SysRole) gin.H {
	item := gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"real_name":  u.RealName,
		"status":     userStatus(u.Status),
		"created_at": u.CreatedAt,
	}
	if u.Phone != nil {
		item["phone"] = *u.Phone
	}
	if u.Email != nil {
		item["email"] = *u.Email
	}
	if u.LastLoginAt != nil {
		item["last_login_at"] = u.LastLoginAt
	}
	if roles != nil {
		var roleItems []gin.H
		for _, r := range roles {
			roleItems = append(roleItems, gin.H{"code": r.Code, "name": r.Name})
		}
		item["roles"] = roleItems
	}
	return item
}

func formatAccountDetail(u *entity.SysUser, roles []entity.SysRole) gin.H {
	item := formatAccountItem(u, roles)
	if u.Remark != nil {
		item["remark"] = *u.Remark
	}
	if u.LastLoginAt != nil {
		item["last_login_at"] = u.LastLoginAt
	}
	item["roles"] = formatRoleItems(roles)
	return item
}

func formatRoleItems(roles []entity.SysRole) []gin.H {
	var items []gin.H
	for _, r := range roles {
		items = append(items, gin.H{"id": r.ID, "code": r.Code, "name": r.Name})
	}
	return items
}

// 角色权限 API：角色列表、权限树、角色权限更新
package v1

import (
	"strconv"

	"opsmind/server/internal/domain/user"
	"opsmind/server/internal/model/entity"
	"opsmind/server/internal/pkg/errors"
	"opsmind/server/internal/pkg/response"
	"opsmind/server/internal/repository"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	svc *user.Service
}

func NewRoleHandler(svc *user.Service) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// ListRoles GET /api/v1/admin/roles
func (h *RoleHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	q := c.Query("q")
	status := c.Query("status")

	roles, total, err := h.svc.ListRoles(page, perPage, q, status)
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "查询角色列表失败")
		return
	}

	var items []gin.H
	for _, r := range roles {
		items = append(items, gin.H{
			"id":     r.ID,
			"code":   r.Code,
			"name":   r.Name,
			"status": roleStatus(r.Status),
			"remark": r.Remark,
		})
	}

	meta := response.Meta{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: repository.TotalPages(total, perPage),
	}
	response.List(c, items, meta)
}

// GetPermissions GET /api/v1/admin/permissions
func (h *RoleHandler) GetPermissions(c *gin.Context) {
	permType := c.Query("type")
	status := c.Query("status")

	perms, err := h.svc.GetPermissionTree(permType, status)
	if err != nil {
		response.ErrorRaw(c, 500, errors.CodeInternalError, "查询权限失败")
		return
	}

	response.OK(c, buildPermissionTree(perms))
}

// UpdateRolePermissions PATCH /api/v1/admin/roles/:id/permissions
func (h *RoleHandler) UpdateRolePermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errors.New(errors.CodeValidationError, "无效的角色 ID"))
		return
	}

	var req struct {
		PermissionIDs []int64 `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errors.New(errors.CodeInvalidJSON, "请求参数格式错误"))
		return
	}

	count, err := h.svc.UpdateRolePermissions(id, req.PermissionIDs)
	if err != nil {
		response.Error(c, err.(*errors.AppError))
		return
	}

	response.OK(c, gin.H{
		"role_id":          id,
		"permission_count": count,
	})
}

func roleStatus(s int16) string {
	if s == 2 {
		return "disabled"
	}
	return "active"
}

// buildPermissionTree 将扁平权限列表转为树形结构（用于管理端权限配置）
func buildPermissionTree(perms []entity.SysPermission) []gin.H {
	pmap := make(map[int64][]entity.SysPermission)
	var roots []entity.SysPermission
	for _, p := range perms {
		if p.ParentID == nil {
			roots = append(roots, p)
		} else {
			pmap[*p.ParentID] = append(pmap[*p.ParentID], p)
		}
	}

	var build func([]entity.SysPermission) []gin.H
	build = func(list []entity.SysPermission) []gin.H {
		var result []gin.H
		for _, p := range list {
			item := gin.H{
				"id":        p.ID,
				"parent_id": p.ParentID,
				"type":      permTypeName(p.Type),
				"name":      p.Name,
				"code":      p.Code,
				"path":      p.Path,
				"method":    p.Method,
				"children":  build(pmap[p.ID]),
			}
			result = append(result, item)
		}
		return result
	}
	return build(roots)
}

func permTypeName(t int16) string {
	switch t {
	case 1:
		return "menu"
	case 2:
		return "button"
	case 3:
		return "api"
	default:
		return "unknown"
	}
}

// Package handler 实现 HTTP 请求处理。
//
// user.go 提供用户管理相关接口。
// Handler 层职责：参数解析、调用 Service、格式化响应。
// 不包含业务逻辑，所有校验和状态机在 Service 层完成。
package handler

import (
	"strconv"

	"opsmind/internal/service"
	"opsmind/pkg/errcode"
	"opsmind/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户管理接口。
type UserHandler struct {
	svc *service.UserService
}

// NewUserHandler 创建 UserHandler 实例。
func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// GetByID 获取用户详情。
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的用户 ID")
		return
	}

	user, svcErr := h.svc.GetByID(id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, user)
}

// List 用户列表（分页）。
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	users, total, err := h.svc.List(page, pageSize, keyword)
	if err != nil {
		response.Error(c, errcode.ErrUnknown, err.Error())
		return
	}

	response.SuccessWithPage(c, users, total, page, pageSize)
}

// Freeze 冻结用户。
func (h *UserHandler) Freeze(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的用户 ID")
		return
	}

	if svcErr := h.svc.Freeze(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

// Restore 恢复已冻结用户。
func (h *UserHandler) Restore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, errcode.ErrParam, "无效的用户 ID")
		return
	}

	if svcErr := h.svc.Restore(id); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	response.Success(c, nil)
}

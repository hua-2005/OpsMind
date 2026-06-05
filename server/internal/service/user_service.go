// Package service 实现用户管理业务逻辑。
//
// UserService 提供用户 CRUD、冻结/恢复功能。
// 为什么冻结/恢复放在 Service 而非 Repository：
// 冻结前需校验当前状态（已冻结不能重复冻结），这类业务规则属于 Service 层职责。
package service

import (
	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/gorm"
)

// UserService 用户管理服务。
type UserService struct {
	repo *repository.UserRepo
	db   *gorm.DB
}

// NewUserService 创建 UserService 实例。
func NewUserService(repo *repository.UserRepo, db *gorm.DB) *UserService {
	return &UserService{repo: repo, db: db}
}

// GetByID 根据 ID 获取用户。
func (s *UserService) GetByID(id int64) (*model.User, error) {
	user, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, AppError{Code: errcode.ErrNotFound, Message: "用户不存在"}
		}
		return nil, err
	}
	return user, nil
}

// List 查询用户列表（分页 + 关键词搜索）。
func (s *UserService) List(page, pageSize int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := s.db.Model(&model.User{})
	if keyword != "" {
		query = query.Where("username LIKE ? OR real_name LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Freeze 冻结用户。
//
// 冻结前校验当前状态：已冻结的用户不能重复冻结（返回 10006）。
func (s *UserService) Freeze(id int64) error {
	user, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "用户不存在"}
		}
		return err
	}

	if user.Status == 2 {
		return AppError{Code: errcode.ErrAlreadyFrozen, Message: "用户已被冻结"}
	}

	user.Status = 2
	return s.repo.Update(user)
}

// Restore 恢复已冻结用户。
//
// 恢复前校验当前状态：正常用户不能重复恢复（返回 10007）。
func (s *UserService) Restore(id int64) error {
	user, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppError{Code: errcode.ErrNotFound, Message: "用户不存在"}
		}
		return err
	}

	if user.Status == 1 {
		return AppError{Code: errcode.ErrAlreadyActive, Message: "用户已处于正常状态"}
	}

	user.Status = 1
	return s.repo.Update(user)
}

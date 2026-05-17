// 用户领域服务：账号管理、角色权限管理
package user

import (
	"opsmind/server/internal/model/entity"
	"opsmind/server/internal/pkg/errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ---------- 账号管理 ----------

// ListAccounts 分页查询账号列表
func (s *Service) ListAccounts(page, perPage int, q, status, roleCode string) ([]entity.SysUser, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	query := s.db.Model(&entity.SysUser{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("username ILIKE ? OR real_name ILIKE ? OR phone ILIKE ?", like, like, like)
	}
	if status == "active" {
		query = query.Where("status = 1")
	} else if status == "frozen" {
		query = query.Where("status = 2")
	}
	if roleCode != "" {
		query = query.Where(`id IN (SELECT user_id FROM sys_user_role ur
			INNER JOIN sys_role r ON r.id = ur.role_id WHERE r.code = ?)`, roleCode)
	}

	var total int64
	query.Count(&total)

	var users []entity.SysUser
	offset := (page - 1) * perPage
	if err := query.Order("id DESC").Offset(offset).Limit(perPage).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CreateAccount 创建本地模拟运维账号
func (s *Service) CreateAccount(username, password, realName, phone, email string, roleIDs []int64, remark string) (*entity.SysUser, error) {
	// 唯一性检查
	var count int64
	s.db.Model(&entity.SysUser{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New(errors.CodeConflict, "登录账号已存在")
	}

	// 密码强度：最小 8 位
	if len(password) < 8 {
		return nil, errors.WithDetails(errors.CodeValidationError, "密码强度不足", []errors.FieldError{
			{Field: "password", Message: "密码至少 8 位", Code: "weak_password"},
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := entity.SysUser{
		Username:     username,
		PasswordHash: string(hash),
		RealName:     realName,
		Status:       1,
	}
	if phone != "" {
		user.Phone = &phone
	}
	if email != "" {
		user.Email = &email
	}
	if remark != "" {
		user.Remark = &remark
	}

	return &user, s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		// 绑定角色
		for _, rid := range roleIDs {
			if err := tx.Create(&entity.SysUserRole{UserID: user.ID, RoleID: rid}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAccount 获取账号详情（含角色）
func (s *Service) GetAccount(id int64) (*entity.SysUser, []entity.SysRole, error) {
	var user entity.SysUser
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, nil, errors.New(errors.CodeNotFound, "账号不存在")
	}

	var roles []entity.SysRole
	s.db.Raw(`
		SELECT r.* FROM sys_role r
		INNER JOIN sys_user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = ?
	`, id).Scan(&roles)

	return &user, roles, nil
}

// UpdateAccount 更新账号资料和角色
func (s *Service) UpdateAccount(id int64, realName, phone, email, remark string, roleIDs []int64) (*entity.SysUser, error) {
	var user entity.SysUser
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, errors.New(errors.CodeNotFound, "账号不存在")
	}

	updates := map[string]interface{}{}
	if realName != "" {
		updates["real_name"] = realName
	}
	if phone != "" {
		updates["phone"] = phone
	}
	if email != "" {
		updates["email"] = email
	}
	if remark != "" {
		updates["remark"] = remark
	}

	return &user, s.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		// 更新角色绑定：先删后插
		if len(roleIDs) > 0 {
			tx.Where("user_id = ?", id).Delete(&entity.SysUserRole{})
			for _, rid := range roleIDs {
				tx.Create(&entity.SysUserRole{UserID: id, RoleID: rid})
			}
		}
		return nil
	})
}

// FreezeAccount 冻结账号
func (s *Service) FreezeAccount(id, operatorID int64, reason string) (*entity.SysUser, error) {
	if id == operatorID {
		return nil, errors.New(errors.CodeValidationError, "不允许冻结当前登录账号")
	}

	var user entity.SysUser
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, errors.New(errors.CodeNotFound, "账号不存在")
	}
	if user.Status == 2 {
		return nil, errors.New(errors.CodeConflict, "账号已是冻结状态")
	}

	remark := "冻结原因: " + reason
	s.db.Model(&user).Updates(map[string]interface{}{
		"status": 2,
		"remark": remark,
	})
	user.Status = 2
	return &user, nil
}

// RestoreAccount 恢复账号
func (s *Service) RestoreAccount(id int64, reason string) (*entity.SysUser, error) {
	var user entity.SysUser
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, errors.New(errors.CodeNotFound, "账号不存在")
	}
	if user.Status != 2 {
		return nil, errors.New(errors.CodeConflict, "账号非冻结状态，无需恢复")
	}

	remark := "恢复原因: " + reason
	s.db.Model(&user).Updates(map[string]interface{}{
		"status": 1,
		"remark": remark,
	})
	user.Status = 1
	return &user, nil
}

// ---------- 角色权限 ----------

// ListRoles 角色列表
func (s *Service) ListRoles(page, perPage int, q, status string) ([]entity.SysRole, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	query := s.db.Model(&entity.SysRole{})
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("code ILIKE ? OR name ILIKE ?", like, like)
	}
	if status == "active" {
		query = query.Where("status = 1")
	} else if status == "disabled" {
		query = query.Where("status = 2")
	}

	var total int64
	query.Count(&total)

	var roles []entity.SysRole
	offset := (page - 1) * perPage
	query.Order("id ASC").Offset(offset).Limit(perPage).Find(&roles)
	return roles, total, nil
}

// GetPermissionTree 权限树（菜单+按钮+接口）
func (s *Service) GetPermissionTree(permType, status string) ([]entity.SysPermission, error) {
	query := s.db.Model(&entity.SysPermission{}).Order("sort_order ASC")
	if permType == "menu" {
		query = query.Where("type = 1")
	} else if permType == "button" {
		query = query.Where("type = 2")
	} else if permType == "api" {
		query = query.Where("type = 3")
	}
	if status == "active" {
		query = query.Where("status = 1")
	} else if status == "disabled" {
		query = query.Where("status = 2")
	}

	var perms []entity.SysPermission
	if err := query.Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

type GinRole struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// GetUserRoles 批量获取用户角色，用于列表展示
func (s *Service) GetUserRoles(userIDs []int64) map[int64][]GinRole {
	if len(userIDs) == 0 {
		return nil
	}
	type row struct {
		UserID int64  `gorm:"column:user_id"`
		Code   string `gorm:"column:code"`
		Name   string `gorm:"column:name"`
	}
	var rows []row
	s.db.Raw(`
		SELECT ur.user_id, r.code, r.name FROM sys_role r
		INNER JOIN sys_user_role ur ON r.id = ur.role_id
		WHERE ur.user_id IN ? AND r.status = 1
	`, userIDs).Scan(&rows)

	result := make(map[int64][]GinRole)
	for _, r := range rows {
		result[r.UserID] = append(result[r.UserID], GinRole{Code: r.Code, Name: r.Name})
	}
	return result
}

// UpdateRolePermissions 更新角色权限绑定
func (s *Service) UpdateRolePermissions(roleID int64, permIDs []int64) (int, error) {
	var role entity.SysRole
	if err := s.db.First(&role, roleID).Error; err != nil {
		return 0, errors.New(errors.CodeNotFound, "角色不存在")
	}

	count := 0
	err := s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("role_id = ?", roleID).Delete(&entity.SysRolePermission{})
		for _, pid := range permIDs {
			tx.Create(&entity.SysRolePermission{RoleID: roleID, PermissionID: pid})
			count++
		}
		return nil
	})
	return count, err
}

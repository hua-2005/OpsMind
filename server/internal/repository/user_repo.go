// Package repository 提供用户的数据访问层。
//
// UserRepo 封装 users 表的查询和创建操作，供 UserService 调用。
// 为什么独立于 ConfigRepo：User 表的查询模式更丰富（按 ID/用户名/手机号），
// 且后续会扩展分页查询、角色关联等功能，独立 Repo 更利于演进。
package repository

import (
	"opsmind/internal/model"

	"gorm.io/gorm"
)

// UserRepo 用户数据访问
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建 UserRepo 实例
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// GetByID 按 ID 查询用户。
//
// ID 是主键，查询不到时返回 gorm.ErrRecordNotFound。
func (r *UserRepo) GetByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 按用户名查询用户。
//
// username 具有唯一索引，用于登录验证。
func (r *UserRepo) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByPhone 按手机号查询用户。
//
// phone 用于报障人身份识别和注册校验。
func (r *UserRepo) GetByPhone(phone string) (*model.User, error) {
	var user model.User
	err := r.db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsByPhone 检查手机号是否已注册。
//
// 为什么单独封装而非复用 GetByPhone：
// 语义更清晰（布尔返回值 vs 结构体），且后续可优化为 SELECT 1 提升性能。
func (r *UserRepo) ExistsByPhone(phone string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("phone = ?", phone).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create 新增用户。
//
// 创建后 user.ID 会被 GORM 自动填充（autoIncrement）。
// 用户名唯一约束由数据库保证，重复时返回 PostgreSQL 错误。
func (r *UserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// Update 更新用户全部字段。
//
// 为什么用 Save 而非 Updates：Save 会更新所有字段（包括零值），
// 适用于修改密码等需要更新 password_hash、first_login、updated_at 的场景。
func (r *UserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

// List 分页查询用户列表，支持关键词搜索。
func (r *UserRepo) List(page, pageSize int, keyword string) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := r.db.Model(&model.User{})
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

// UpdateStatus 更新用户状态（冻结/恢复）。
//
// 为什么单独封装而非复用 Update：仅更新 status 字段，避免意外覆盖其他字段。
func (r *UserRepo) UpdateStatus(id int64, status int) error {
	return r.db.Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

// ExistsByUsername 检查用户名是否已存在。
func (r *UserRepo) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// --- 角色/菜单/权限关联查询 ---

// GetUserRoles 查询用户关联的角色列表。
func (r *UserRepo) GetUserRoles(userID int64) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	return roles, err
}

// AssignRoles 分配用户角色（先删后插，保证幂等）。
func (r *UserRepo) AssignRoles(userID int64, roleIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
			return err
		}
		for _, roleID := range roleIDs {
			if err := tx.Create(&model.UserRole{UserID: userID, RoleID: roleID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListMenus 查询全部菜单（按排序字段升序）。
func (r *UserRepo) ListMenus() ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Order("sort_order ASC, id ASC").Find(&menus).Error
	return menus, err
}

// GetRoleMenus 查询角色关联的菜单列表。
func (r *UserRepo) GetRoleMenus(roleID int64) ([]model.Menu, error) {
	var menus []model.Menu
	err := r.db.Joins("JOIN role_menus ON role_menus.menu_id = menus.id").
		Where("role_menus.role_id = ?", roleID).
		Order("menus.sort_order ASC, menus.id ASC").
		Find(&menus).Error
	return menus, err
}

// UpdateRoleMenus 更新角色菜单关联（先删后插）。
func (r *UserRepo) UpdateRoleMenus(roleID int64, menuIDs []int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RoleMenu{}).Error; err != nil {
			return err
		}
		for _, menuID := range menuIDs {
			if err := tx.Create(&model.RoleMenu{RoleID: roleID, MenuID: menuID}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUserPermissions 聚合用户所有角色的权限列表（去重）。
func (r *UserRepo) GetUserPermissions(userID int64) ([]string, error) {
	roles, err := r.GetUserRoles(userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, role := range roles {
		// Permissions 是 datatypes.JSON，需要从 Go 侧解析
		// 这里直接返回角色名列表，由 Service 层根据角色名映射权限
		_ = role
	}

	// 查询角色名列表
	var roleNames []string
	err = r.db.Model(&model.Role{}).
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Pluck("roles.name", &roleNames).Error
	if err != nil {
		return nil, err
	}

	// 根据角色名映射权限（与 middleware/rolePermissions 保持一致）
	permMap := map[string][]string{
		"系统管理员":   {"ticket:read", "ticket:write", "ticket:assign", "knowledge:read", "knowledge:write", "knowledge:review", "system:config", "user:manage", "audit:read"},
		"运维人员":    {"ticket:read", "ticket:write", "knowledge:read", "knowledge:write"},
		"知识库管理员": {"knowledge:read", "knowledge:write", "knowledge:review"},
		"报障人":     {},
	}

	for _, name := range roleNames {
		for _, perm := range permMap[name] {
			seen[perm] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for perm := range seen {
		result = append(result, perm)
	}
	return result, nil
}

// 种子数据：替换迁移中的占位密码为真实 bcrypt 哈希
package bootstrap

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdminPassword 检查并替换 admin 用户的占位密码哈希
func SeedAdminPassword(db *gorm.DB) error {
	var result struct {
		ID   int64
		Hash string
	}
	if err := db.Raw("SELECT id, password_hash FROM sys_user WHERE username = 'admin'").Scan(&result).Error; err != nil {
		return err
	}

	// 检查是否需要替换占位密码（首次运行或密码仍是占位符）
	if result.ID > 0 && len(result.Hash) < 60 {
		realHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := db.Exec("UPDATE sys_user SET password_hash = ? WHERE id = ?", string(realHash), result.ID).Error; err != nil {
			return err
		}
		log.Println("admin password seeded (admin/admin123)")
	}

	return nil
}

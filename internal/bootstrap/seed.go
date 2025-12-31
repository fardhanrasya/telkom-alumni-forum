package bootstrap

import (
	"log"

	"anoa.com/telkomalumiforum/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Role{},
		&model.User{},
		&model.Profile{},
		&model.Category{},
		&model.Thread{},
		&model.Post{},
		&model.Attachment{},
		&model.Notification{},
		&model.Menfess{},
		&model.PointLog{},
		&model.UserStats{},
		&model.Reaction{},
	)
}

func SeedRoles(db *gorm.DB) error {
	defaultRoles := []model.Role{
		{Name: "admin", Description: "Super administrator"},
		{Name: "guru", Description: "Guru"},
		{Name: "siswa", Description: "Siswa"},
	}

	for _, role := range defaultRoles {
		var count int64
		if err := db.Model(&model.Role{}).
			Where("name = ?", role.Name).
			Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			if err := db.Create(&role).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func SeedAdminUser(db *gorm.DB) error {
	var adminRole model.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&model.User{}).
		Where("email = ?", "admin@telkom.com").
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		log.Println("Admin user already exists, skipping seed")
		return nil
	}

	password := "admin123"
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := model.User{
		Username:     "admin",
		Email:        "admin@telkom.com",
		PasswordHash: string(hashedPasswordBytes),
		RoleID:       &adminRole.ID,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	adminProfile := model.Profile{
		UserID:   adminUser.ID,
		FullName: "Administrator",
		Bio:      stringPtr("System Administrator"),
	}

	if err := db.Create(&adminProfile).Error; err != nil {
		return err
	}

	log.Println("✅ Admin user seeded successfully")
	log.Println("   Email: admin@telkom.com")
	log.Println("   Password: admin123")

	return nil
}

func SeedBotUser(db *gorm.DB) error {
	var siswaRole model.Role
	if err := db.Where("name = ?", "siswa").First(&siswaRole).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&model.User{}).
		Where("username = ?", "Mading_Bot").
		Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	password := "bot12345" // Random password, not intended for login
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	botUser := model.User{
		Username:     "Mading_Bot",
		Email:        "bot@telkom.com",
		PasswordHash: string(hashedPasswordBytes),
		RoleID:       &siswaRole.ID,
		AvatarURL:    stringPtr("https://ui-avatars.com/api/?name=Mading+Bot&background=random"),
	}

	if err := db.Create(&botUser).Error; err != nil {
		return err
	}

	botProfile := model.Profile{
		UserID:   botUser.ID,
		FullName: "Mading Bot",
		Bio:      stringPtr("🤖 Bot Mading Sekolah - Memberikan informasi terkini seputar teknologi dan berita sekolah."),
	}

	if err := db.Create(&botProfile).Error; err != nil {
		return err
	}

	log.Println("✅ Bot user seeded successfully")
	return nil
}

func stringPtr(s string) *string {
	return &s
}

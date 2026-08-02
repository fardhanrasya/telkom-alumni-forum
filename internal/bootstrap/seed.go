package bootstrap

import (
	"log"

	"anoa.com/telkomalumiforum/internal/entity"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&entity.Role{},
		&entity.User{},
		&entity.Profile{},
		&entity.Category{},
		&entity.Thread{},
		&entity.Post{},
		&entity.Attachment{},
		&entity.Notification{},
		&entity.Menfess{},
		&entity.PointLog{},
		&entity.UserStats{},
		&entity.Reaction{},
		&entity.UserFollow{},
		&entity.ThreadFeedView{},
		&entity.Wallet{},
		&entity.CoinTransaction{},
		&entity.Cosmetic{},
		&entity.CosmeticOwnership{},
		&entity.UserEquip{},
		&entity.MissionDefinition{},
		&entity.MissionProgress{},
	)
}

// SeedMissionDefinitions inserts the initial mission catalog described in
// docs/SPEC-cosmetics-economy.md §4.1. view_thread and login_streak are
// seeded inactive: they need per-user-per-day view tracking and a
// last_active_date/streak column that don't exist yet — separate tickets.
func SeedMissionDefinitions(db *gorm.DB) error {
	missions := []entity.MissionDefinition{
		{ActionType: "create_thread", Kind: "daily", Name: "Buat 1 thread", Target: 1, Reward: 10, IsActive: true},
		{ActionType: "like_received", Kind: "daily", Name: "Dapat 3 like", Target: 3, Reward: 10, IsActive: true},
		{ActionType: "comment_received", Kind: "daily", Name: "Dapat 3 komentar", Target: 3, Reward: 10, IsActive: true},
		{ActionType: "view_thread", Kind: "daily", Name: "Baca 5 thread berbeda", Target: 5, Reward: 5, IsActive: false},
		{ActionType: "login_streak", Kind: "daily", Name: "Login hari ini", Target: 1, Reward: 5, IsActive: false},
		{ActionType: "follow", Kind: "achievement", Name: "Follow 10 pengguna", Target: 10, Reward: 50, IsActive: true},
	}

	for _, m := range missions {
		var count int64
		if err := db.Model(&entity.MissionDefinition{}).
			Where("action_type = ? AND kind = ?", m.ActionType, m.Kind).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Create(&m).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func SeedRoles(db *gorm.DB) error {
	defaultRoles := []entity.Role{
		{Name: "admin", Description: "Super administrator"},
		{Name: "guru", Description: "Guru"},
		{Name: "siswa", Description: "Siswa"},
	}

	for _, role := range defaultRoles {
		var count int64
		if err := db.Model(&entity.Role{}).
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
	var adminRole entity.Role
	if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&entity.User{}).
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

	adminUser := entity.User{
		Username:     "admin",
		Email:        "admin@telkom.com",
		PasswordHash: string(hashedPasswordBytes),
		RoleID:       &adminRole.ID,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return err
	}

	adminProfile := entity.Profile{
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
	var siswaRole entity.Role
	if err := db.Where("name = ?", "siswa").First(&siswaRole).Error; err != nil {
		return err
	}

	var count int64
	if err := db.Model(&entity.User{}).
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

	botUser := entity.User{
		Username:     "Mading_Bot",
		Email:        "bot@telkom.com",
		PasswordHash: string(hashedPasswordBytes),
		RoleID:       &siswaRole.ID,
		AvatarURL:    stringPtr("https://ui-avatars.com/api/?name=Mading+Bot&background=random"),
	}

	if err := db.Create(&botUser).Error; err != nil {
		return err
	}

	botProfile := entity.Profile{
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

package repository

import (
	"context"

	"anoa.com/telkomalumiforum/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User, profile *model.Profile) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindRoleByName(ctx context.Context, name string) (*model.Role, error)
	Update(ctx context.Context, user *model.User, profile *model.Profile) error
	FindAll(ctx context.Context) ([]*model.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *userRepository) Create(ctx context.Context, user *model.User, profile *model.Profile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		if profile != nil {
			profile.UserID = user.ID
			if err := tx.Create(profile).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Profile").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Profile").
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Profile").
		Where("id = ?", id).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}

	return &role, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User, profile *model.Profile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}

		if profile != nil {
			if err := tx.Save(profile).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *userRepository) FindAll(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	if err := r.db.WithContext(ctx).
		Preload("Role").
		Preload("Profile").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}

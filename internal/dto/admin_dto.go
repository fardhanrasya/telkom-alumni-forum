package dto

import "anoa.com/telkomalumiforum/internal/model"

type CreateUserInput struct {
	Username       string  `json:"username" form:"username" binding:"required,min=3,max=50"`
	Email          string  `json:"email" form:"email" binding:"required,email"`
	Password       string  `json:"password" form:"password" binding:"required,min=8"`
	Role           string  `json:"role" form:"role" binding:"required"`
	FullName       string  `json:"full_name" form:"full_name" binding:"required"`
	IdentityNumber *string `json:"identity_number" form:"identity_number"`
	ClassGrade     *string `json:"class_grade" form:"class_grade"`
	Bio            *string `json:"bio" form:"bio"`
}

type CreateUserResponse struct {
	User    *model.User    `json:"user"`
	Role    *model.Role    `json:"role"`
	Profile *model.Profile `json:"profile"`
}

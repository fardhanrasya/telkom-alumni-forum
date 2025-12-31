package service

import (
	"context"

	"anoa.com/telkomalumiforum/internal/modules/user/repository"
)

type StatService interface {
	GetTotalUsers(ctx context.Context) (int64, error)
}

type statService struct {
	userRepo repository.UserRepository
}

func NewStatService(userRepo repository.UserRepository) StatService {
	return &statService{
		userRepo: userRepo,
	}
}

func (s *statService) GetTotalUsers(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

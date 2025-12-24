package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"anoa.com/telkomalumiforum/internal/model"
	"anoa.com/telkomalumiforum/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type LikeService interface {
	LikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	UnlikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error
	LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error
	GetThreadLikes(ctx context.Context, threadID uuid.UUID) (int64, error)
	GetPostLikes(ctx context.Context, postID uuid.UUID) (int64, error)
	CheckUserLikedThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) (bool, error)
	CheckUserLikedPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)
	StartWorker(ctx context.Context)
}

type likeService struct {
	redisClient         *redis.Client
	likeRepo            repository.LikeRepository
	threadRepo          repository.ThreadRepository
	postRepo            repository.PostRepository
	notificationService NotificationService
	leaderboardService  LeaderboardService
}

func NewLikeService(redisClient *redis.Client, likeRepo repository.LikeRepository, threadRepo repository.ThreadRepository, postRepo repository.PostRepository, notificationService NotificationService, leaderboardService LeaderboardService) LikeService {
	return &likeService{
		redisClient:         redisClient,
		likeRepo:            likeRepo,
		threadRepo:          threadRepo,
		postRepo:            postRepo,
		notificationService: notificationService,
		leaderboardService:  leaderboardService,
	}
}

const (
	LikeQueueKey = "like_queue"
)

type LikeTask struct {
	Type     string `json:"type"`   // "thread" or "post"
	Action   string `json:"action"` // "like" or "unlike"
	UserID   string `json:"user_id"`
	TargetID string `json:"target_id"`
}

func (s *likeService) LikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	key := fmt.Sprintf("thread_likes:%s", threadID.String())

	// 1. Check if user already liked in Redis
	isMember, err := s.redisClient.SIsMember(ctx, key, userID.String()).Result()
	if err != nil {
		return err
	}
	if isMember {
		return fmt.Errorf("already liked")
	}

	// 2. Add to Redis
	if err := s.redisClient.SAdd(ctx, key, userID.String()).Err(); err != nil {
		return err
	}

	// 3. Push to Worker Queue
	task := LikeTask{
		Type:     "thread",
		Action:   "like",
		UserID:   userID.String(),
		TargetID: threadID.String(),
	}
	return s.pushTask(ctx, task)
}

func (s *likeService) UnlikeThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) error {
	key := fmt.Sprintf("thread_likes:%s", threadID.String())

	// 1. Remove from Redis
	// We don't necessarily need to check if exists, just remove.
	// But if strict:
	// isMember, _ := s.redisClient.SIsMember(...)

	if err := s.redisClient.SRem(ctx, key, userID.String()).Err(); err != nil {
		return err
	}

	// 2. Push to Worker Queue
	task := LikeTask{
		Type:     "thread",
		Action:   "unlike",
		UserID:   userID.String(),
		TargetID: threadID.String(),
	}
	return s.pushTask(ctx, task)
}

func (s *likeService) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	key := fmt.Sprintf("post_likes:%s", postID.String())

	isMember, err := s.redisClient.SIsMember(ctx, key, userID.String()).Result()
	if err != nil {
		return err
	}
	if isMember {
		return fmt.Errorf("already liked")
	}

	if err := s.redisClient.SAdd(ctx, key, userID.String()).Err(); err != nil {
		return err
	}

	task := LikeTask{
		Type:     "post",
		Action:   "like",
		UserID:   userID.String(),
		TargetID: postID.String(),
	}
	return s.pushTask(ctx, task)
}

func (s *likeService) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) error {
	key := fmt.Sprintf("post_likes:%s", postID.String())

	if err := s.redisClient.SRem(ctx, key, userID.String()).Err(); err != nil {
		return err
	}

	task := LikeTask{
		Type:     "post",
		Action:   "unlike",
		UserID:   userID.String(),
		TargetID: postID.String(),
	}
	return s.pushTask(ctx, task)
}

func (s *likeService) GetThreadLikes(ctx context.Context, threadID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("thread_likes:%s", threadID.String())
	return s.redisClient.SCard(ctx, key).Result()
}

func (s *likeService) GetPostLikes(ctx context.Context, postID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("post_likes:%s", postID.String())
	return s.redisClient.SCard(ctx, key).Result()
}

func (s *likeService) CheckUserLikedThread(ctx context.Context, userID uuid.UUID, threadID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("thread_likes:%s", threadID.String())
	isMember, err := s.redisClient.SIsMember(ctx, key, userID.String()).Result()
	if err == nil && isMember {
		return true, nil
	}

	return s.likeRepo.IsThreadLiked(ctx, userID, threadID)
}

func (s *likeService) CheckUserLikedPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("post_likes:%s", postID.String())
	isMember, err := s.redisClient.SIsMember(ctx, key, userID.String()).Result()
	if err == nil && isMember {
		return true, nil
	}
	return s.likeRepo.IsPostLiked(ctx, userID, postID)
}

func (s *likeService) pushTask(ctx context.Context, task LikeTask) error {
	bytes, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return s.redisClient.RPush(ctx, LikeQueueKey, bytes).Err()
}

func (s *likeService) StartWorker(ctx context.Context) {
	log.Println("❤️ Like Worker Started...")
	for {
		// BLPOP blocks until item available
		// 0 timeout means block indefinitely
		res, err := s.redisClient.BLPop(ctx, 0, LikeQueueKey).Result()
		if err != nil {
			// If context cancelled or connection error
			if ctx.Err() != nil {
				return
			}
			log.Printf("Redis BLPOP error: %v, retrying in 1s...", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// res[0] is key, res[1] is value
		if len(res) < 2 {
			continue
		}

		var task LikeTask
		if err := json.Unmarshal([]byte(res[1]), &task); err != nil {
			log.Printf("Invalid like task json: %v", err)
			continue
		}

		s.processTask(ctx, task)
	}
}

func (s *likeService) processTask(ctx context.Context, task LikeTask) {
	userID, err := uuid.Parse(task.UserID)
	if err != nil {
		log.Printf("Invalid UserID in like task: %v", err)
		return
	}
	targetID, err := uuid.Parse(task.TargetID)
	if err != nil {
		log.Printf("Invalid TargetID in like task: %v", err)
		return
	}

	var opErr error
	switch task.Type {
	case "thread":
		if task.Action == "like" {
			opErr = s.likeRepo.LikeThread(ctx, userID, targetID)
			if opErr == nil {
				// Notify Thread Author
				thread, err := s.threadRepo.FindByID(ctx, targetID)
				if err == nil && thread.UserID != userID {
					notif := &model.Notification{
						UserID:     thread.UserID,
						ActorID:    userID,
						EntityID:   thread.ID,
						EntitySlug: thread.Slug,
						EntityType: "thread",
						Type:       "like_thread",
						Message:    "Someone liked your thread",
					}
					_ = s.notificationService.CreateNotification(ctx, notif)

					// Gamification: Give points to thread author
					if s.leaderboardService != nil {
						s.leaderboardService.AddGamificationPointsAsync(thread.UserID, ActionLikeReceived, thread.ID.String(), "threads")
					}
				}
			}
		} else {
			opErr = s.likeRepo.UnlikeThread(ctx, userID, targetID)
		}
	case "post":
		if task.Action == "like" {
			opErr = s.likeRepo.LikePost(ctx, userID, targetID)
			if opErr == nil {
				// Notify Post Author
				post, err := s.postRepo.FindByID(ctx, targetID)
				if err == nil && post.UserID != userID {
					// Need thread for slug
					// post doesn't usually preload thread unless FindByID does.
					// Safest is to fetch thread.
					thread, errThread := s.threadRepo.FindByID(ctx, post.ThreadID)
					var slug string
					if errThread == nil {
						slug = thread.Slug
					}

					notif := &model.Notification{
						UserID:     post.UserID,
						ActorID:    userID,
						EntityID:   post.ID,
						EntitySlug: slug,
						EntityType: "post",
						Type:       "like_post",
						Message:    "Someone liked your post",
					}
					_ = s.notificationService.CreateNotification(ctx, notif)

					// Gamification: Give points to post author
					if s.leaderboardService != nil {
						s.leaderboardService.AddGamificationPointsAsync(post.UserID, ActionLikeReceived, post.ID.String(), "posts")
					}
				}
			}
		} else {
			opErr = s.likeRepo.UnlikePost(ctx, userID, targetID)
		}
	}

	if opErr != nil {
		// Log error, maybe retry?
		// For duplicates on 'like', we might ignore.
		log.Printf("Failed to process like task %+v: %v", task, opErr)
	}
}

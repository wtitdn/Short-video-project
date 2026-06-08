package usecase

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/wtitdn/renew_video/internal/controller/apierror"
	"github.com/wtitdn/renew_video/internal/entity"
	llm2 "github.com/wtitdn/renew_video/internal/middleware/llm"
	"github.com/wtitdn/renew_video/internal/middleware/rabbitmq/producer"
	"github.com/wtitdn/renew_video/internal/repo"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
	"gorm.io/gorm"
)

type CommentService struct {
	ar              *repo.AccountRepository
	repo            *repo.CommentRepository
	VideoRepository *repo.VideoRepository
	cache           *rediscache.Client
	commentMQ       *producer.CommentMQ
	popularityMQ    *producer.PopularityMQ
	llm             *llm2.LLMClient
}

func NewCommentService(ar *repo.AccountRepository, repo *repo.CommentRepository, videoRepo *repo.VideoRepository, cache *rediscache.Client, commentMQ *producer.CommentMQ, popularityMQ *producer.PopularityMQ, llm *llm2.LLMClient) *CommentService {
	return &CommentService{ar: ar, repo: repo, VideoRepository: videoRepo, cache: cache, commentMQ: commentMQ, popularityMQ: popularityMQ, llm: llm}
}

func (s *CommentService) Publish(ctx context.Context, comment *entity.Comment) error {
	if comment == nil {
		return errors.New("comment is nil")
	}
	comment.Username = strings.TrimSpace(comment.Username)
	comment.Content = strings.TrimSpace(comment.Content)
	if comment.VideoID == 0 || comment.AuthorID == 0 {
		return errors.New("video_id and author_id are required")
	}
	if comment.Content == "" {
		return errors.New("content is required")
	}

	exists, err := s.VideoRepository.IsExist(ctx, comment.VideoID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("video not found")
	}

	mysqlEnqueued := false
	redisEnqueued := false
	if s.commentMQ != nil {
		if err := s.commentMQ.Publish(ctx, comment.Username, comment.VideoID, comment.AuthorID, comment.Content); err == nil {
			mysqlEnqueued = true
		}
	}
	if s.popularityMQ != nil {
		if err := s.popularityMQ.Update(ctx, comment.VideoID, 1); err == nil {
			redisEnqueued = true
		}
	}
	if mysqlEnqueued && redisEnqueued {
		s.notifyMentions(ctx, comment)
		return nil
	}

	// Fallback: direct MySQL write when comment MQ publish fails.
	if !mysqlEnqueued {
		if err := s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
			if err := tx.Select("id").First(&entity.Video{}, comment.VideoID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("video not found")
				}
				return err
			}
			if err := tx.Create(comment).Error; err != nil {
				return err
			}
			return tx.Model(&entity.Video{}).Where("id = ?", comment.VideoID).
				UpdateColumn("popularity", gorm.Expr("popularity + 1")).Error
		}); err != nil {
			return err
		}
	}

	// Fallback: direct Redis update when popularity MQ publish fails.
	if !redisEnqueued {
		UpdatePopularityCache(ctx, s.cache, comment.VideoID, 1)
	}
	s.notifyMentions(ctx, comment)
	return nil
}

func (s *CommentService) Delete(ctx context.Context, commentID uint, accountID uint) error {
	comment, err := s.repo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment == nil {
		return errors.New("comment not found")
	}
	if comment.AuthorID != accountID {
		return apierror.ErrUnauthorized
	}
	if s.commentMQ != nil {
		if err := s.commentMQ.Delete(ctx, commentID); err == nil {
			return nil
		}
	}
	return s.repo.DeleteComment(ctx, comment)
}

func (s *CommentService) GetAll(ctx context.Context, videoID uint) ([]entity.Comment, error) {
	exists, err := s.VideoRepository.IsExist(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("video not found")
	}
	return s.repo.GetAllComments(ctx, videoID)
}

var mentionRegex = regexp.MustCompile(`@(\w+)`)

func (s *CommentService) notifyMentions(ctx context.Context, comment *entity.Comment) {
	matches := mentionRegex.FindAllStringSubmatch(comment.Content, -1)
	if len(matches) == 0 {
		return
	}
	seen := make(map[string]bool)
	for _, m := range matches {
		username := m[1]
		if seen[username] || username == comment.Username {
			continue
		}
		seen[username] = true

		account, err := s.ar.FindByUsername(ctx, username)
		if err != nil || account == nil || account.ID == 0 {
			continue
		}

		notif := &entity.Notification{
			RecipientID: account.ID,
			SenderID:    comment.AuthorID,
			Type:        "mention",
			TargetID:    comment.VideoID,
			Content:     comment.Username + " 在评论中提到了你",
		}
		err = s.repo.CreateNotification(ctx, notif)
	}
}

func (s *CommentService) AiSummary(ctx context.Context, videoID uint, comments []entity.Comment) (string, error) {
	if s.llm == nil {
		return "", errors.New("llm client is nil")
	}

	var key string
	if s.cache != nil {
		key = s.cache.Key("ai_summary:%d", videoID)
		result, err := s.cache.GetBytes(ctx, key)
		if err == nil {
			return string(result), nil
		}
	}
	//没有key存在，也就是尚未进行summary
	//30s作为超时
	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res, err := s.llm.LLMSummary(llmCtx, comments)

	if err != nil {
		return "", err
	}

	if s.cache != nil {
		_ = s.cache.SetBytes(ctx, key, []byte(res), 1*time.Hour)
	}

	return res, nil
}

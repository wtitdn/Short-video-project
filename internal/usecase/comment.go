package usecase

import (
	"github.com/wtitdn/renew_video/internal/repo"
	rediscache "github.com/wtitdn/renew_video/pkg/redis"
)

type CommentService struct {
	repo            *repo.CommentRepository
	VideoRepository *repo.VideoRepository
	cache           *rediscache.Client
	commentMQ       *rabbitmq.CommentMQ
	popularityMQ    *rabbitmq.PopularityMQ
}

func NewCommentService(repo *repo.CommentRepository, videoRepo *repo.VideoRepository, cache *rediscache.Client, commentMQ *rabbitmq.CommentMQ, popularityMQ *rabbitmq.PopularityMQ) *CommentService {
	return &CommentService{repo: repo, VideoRepository: videoRepo, cache: cache, commentMQ: commentMQ, popularityMQ: popularityMQ}
}

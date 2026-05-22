package repo

import (
	"context"

	"github.com/wtitdn/renew_video/internal/entity"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) CreateComment(ctx context.Context, comment *entity.Comment) error {

}
func (r *CommentRepository) DeleteComment(ctx context.Context, comment *entity.Comment) error {

}
func (r *CommentRepository) GetAllComments(ctx context.Context, videoID uint) ([]entity.Comment, error) {

}
func (r *CommentRepository) IsExist(ctx context.Context, id uint) (bool, error) {

}
func (r *CommentRepository) GetByID(ctx context.Context, id uint) (*entity.Comment, error) {

}

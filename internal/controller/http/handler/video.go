package handler

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wtitdn/renew_video/internal/controller/apierror"
	"github.com/wtitdn/renew_video/internal/entity"
	"github.com/wtitdn/renew_video/internal/repo"
	"github.com/wtitdn/renew_video/internal/usecase"
	"github.com/wtitdn/renew_video/pkg/jwt"
)

const videoBucket = "videosys"
const minioOperationTimeout = 5 * time.Second

type VideoHandler struct {
	service        *usecase.VideoService
	accountService *usecase.AccountService
	minioRepo      *repo.MinioRepository
}

func NewVideoHandler(service *usecase.VideoService, accountService *usecase.AccountService, minioRepo *repo.MinioRepository) *VideoHandler {
	return &VideoHandler{service: service, accountService: accountService, minioRepo: minioRepo}
}

func (vh *VideoHandler) PublishVideo(c *gin.Context) {
	var req entity.PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	username, err := jwt.GetUsername(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	video := &entity.Video{
		AuthorID:    authorId,
		Username:    username,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CoverURL:    req.CoverURL,
		CreateTime:  time.Now(),
	}
	if err := vh.service.Publish(c.Request.Context(), video); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}

func (vh *VideoHandler) UploadVideo(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	const maxSize = 200 << 20
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}

	ext := strings.ToLower(filepath.Ext(f.Filename))
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 is allowed"})
		return
	}

	date := time.Now().Format("20060102")
	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	filename = filename + ext
	objectKey := path.Join("videos", fmt.Sprintf("%d", authorId), date, filename)

	src, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	contentType := f.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}
	minioCtx, cancel := context.WithTimeout(c.Request.Context(), minioOperationTimeout)
	defer cancel()

	if err := vh.minioRepo.EnsureBucket(minioCtx, videoBucket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	info, err := vh.minioRepo.UploadObjectInfo(c.Request.Context(), videoBucket, objectKey, contentType, src, f.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	playURL := info.Location
	if playURL == "" {
		playURL, err = vh.minioRepo.PresignedGetURL(c.Request.Context(), videoBucket, objectKey, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"bucket":     videoBucket,
		"object_key": objectKey,
		"url":        playURL,
		"play_url":   playURL,
	})
}

func (vh *VideoHandler) UploadCover(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	const maxSize = 10 << 20
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file size"})
		return
	}

	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .jpg/.jpeg/.png/.webp is allowed"})
		return
	}

	date := time.Now().Format("20060102")
	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	filename = filename + ext
	objectKey := path.Join("covers", fmt.Sprintf("%d", authorId), date, filename)

	src, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()

	contentType := f.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := vh.minioRepo.EnsureBucket(c.Request.Context(), videoBucket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	info, err := vh.minioRepo.UploadObjectInfo(c.Request.Context(), videoBucket, objectKey, contentType, src, f.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	coverURL := info.Location
	if coverURL == "" {
		coverURL, err = vh.minioRepo.PresignedGetURL(c.Request.Context(), videoBucket, objectKey, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"bucket":     videoBucket,
		"object_key": objectKey,
		"url":        coverURL,
		"cover_url":  coverURL,
	})
}

func (vh *VideoHandler) InitChunkUpload(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	authorID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req entity.InitChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if ext != ".mp4" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .mp4 is allowed"})
		return
	}
	if req.ContentType == "" {
		req.ContentType = "video/mp4"
	}

	if err := vh.minioRepo.EnsureBucket(c.Request.Context(), videoBucket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	name, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	objectKey := path.Join("videos", fmt.Sprintf("%d", authorID), time.Now().Format("20060102"), name+ext)
	uploadID, err := vh.minioRepo.InitUpload(c.Request.Context(), videoBucket, objectKey, req.ContentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entity.InitChunkUploadResponse{
		Bucket:    videoBucket,
		ObjectKey: objectKey,
		UploadID:  uploadID,
	})
}

func (vh *VideoHandler) CreateChunkPartURL(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	var req entity.ChunkPartURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	u, err := vh.minioRepo.CreatePartURL(c.Request.Context(), videoBucket, req.ObjectKey, req.UploadID, req.PartNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entity.ChunkPartURLResponse{PartNumber: req.PartNumber, URL: u})
}

func (vh *VideoHandler) CompleteChunkUpload(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	var req entity.CompleteChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if len(req.Parts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parts are required"})
		return
	}

	parts := make([]repo.CompletePart, 0, len(req.Parts))
	for _, p := range req.Parts {
		if p.PartNumber <= 0 || strings.TrimSpace(p.ETag) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid part"})
			return
		}
		parts = append(parts, repo.CompletePart{PartNumber: p.PartNumber, ETag: p.ETag})
	}

	info, err := vh.minioRepo.CompleteUpload(c.Request.Context(), videoBucket, req.ObjectKey, req.UploadID, parts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	playURL := info.Location
	if playURL == "" {
		playURL, _ = vh.minioRepo.PresignedGetURL(c.Request.Context(), videoBucket, req.ObjectKey, 24*time.Hour)
	}
	c.JSON(http.StatusOK, gin.H{
		"bucket":     info.Bucket,
		"object_key": info.Key,
		"etag":       info.ETag,
		"location":   info.Location,
		"url":        playURL,
		"play_url":   playURL,
	})
}

func (vh *VideoHandler) AbortChunkUpload(c *gin.Context) {
	if vh.minioRepo == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "minio is not available"})
		return
	}
	var req entity.AbortChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.minioRepo.AbortUpload(c.Request.Context(), videoBucket, req.ObjectKey, req.UploadID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "aborted"})
}

func (vh *VideoHandler) DeleteVideo(c *gin.Context) {
	var req entity.DeleteVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.Delete(c.Request.Context(), req.ID, authorId); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "video deleted"})
}

func (vh *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req entity.ListByAuthorIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	videos, err := vh.service.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []entity.Video{}
	}
	c.JSON(200, videos)
}

func (vh *VideoHandler) GetDetail(c *gin.Context) {
	var req entity.GetDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	video, err := vh.service.GetDetail(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}

func (vh *VideoHandler) UpdateLikesCount(c *gin.Context) {
	var req entity.UpdateLikesCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.UpdateLikesCount(c.Request.Context(), req.ID, req.LikesCount); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "likes count updated"})
}

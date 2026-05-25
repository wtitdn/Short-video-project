package repo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	miniosdk "github.com/minio/minio-go/v7"
	storage "github.com/wtitdn/renew_video/pkg/minio"
)

type MinioRepository struct {
	storage *storage.Minio
}

type CompletePart struct {
	PartNumber int
	ETag       string
}

func NewMinioRepository(storage *storage.Minio) *MinioRepository {
	return &MinioRepository{storage: storage}
}
func (r *MinioRepository) InitUpload(ctx context.Context, bucket, objectKey, contentType string) (string, error) {
	if r == nil || r.storage == nil || r.storage.MinioCore == nil {
		return "", errors.New("minio repository is not initialized")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return r.storage.MinioCore.NewMultipartUpload(
		ctx,
		bucket,
		objectKey,
		miniosdk.PutObjectOptions{
			ContentType: contentType,
		},
	)
}
func (r *MinioRepository) EnsureBucket(ctx context.Context, bucket string) error {
	if r == nil || r.storage == nil || r.storage.MinioClient == nil {
		return errors.New("minio repository is not initialized")
	}

	exists, err := r.storage.MinioClient.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return r.storage.MinioClient.MakeBucket(ctx, bucket, miniosdk.MakeBucketOptions{})
}

func (r *MinioRepository) CreatePartURL(ctx context.Context, bucket, objectKey, uploadID string, partNumber int) (string, error) {
	if r == nil || r.storage == nil || r.storage.MinioClient == nil {
		return "", errors.New("minio repository is not initialized")
	}
	if bucket == "" || objectKey == "" || uploadID == "" || partNumber <= 0 {
		return "", errors.New("invalid multipart upload params")
	}

	query := url.Values{}
	query.Set("partNumber", strconv.Itoa(partNumber))
	query.Set("uploadId", uploadID)

	u, err := r.storage.MinioClient.Presign(
		ctx,
		http.MethodPut,
		bucket,
		objectKey,
		24*time.Hour,
		query,
	)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (r *MinioRepository) CompleteUpload(ctx context.Context, bucket, objectKey, uploadID string, parts []CompletePart) (*miniosdk.UploadInfo, error) {
	if r == nil || r.storage == nil || r.storage.MinioCore == nil {
		return nil, errors.New("minio repository is not initialized")
	}
	if bucket == "" || objectKey == "" || uploadID == "" || len(parts) == 0 {
		return nil, errors.New("invalid complete upload params")
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	minioParts := make([]miniosdk.CompletePart, 0, len(parts))
	for _, p := range parts {
		minioParts = append(minioParts, miniosdk.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       strings.Trim(p.ETag, `"`),
		})
	}

	info, err := r.storage.MinioCore.CompleteMultipartUpload(
		ctx,
		bucket,
		objectKey,
		uploadID,
		minioParts,
		miniosdk.PutObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
func (r *MinioRepository) AbortUpload(ctx context.Context, bucket, objectKey, uploadID string) error {
	if r == nil || r.storage == nil || r.storage.MinioCore == nil {
		return errors.New("minio repository is not initialized")
	}
	return r.storage.MinioCore.AbortMultipartUpload(ctx, bucket, objectKey, uploadID)
}

func (r *MinioRepository) RemoveObject(ctx context.Context, bucket, objectKey string) error {
	if r == nil || r.storage == nil || r.storage.MinioClient == nil {
		return errors.New("minio repository is not initialized")
	}
	if bucket == "" || objectKey == "" {
		return errors.New("bucket and objectKey are required")
	}

	return r.storage.MinioClient.RemoveObject(
		ctx,
		bucket,
		objectKey,
		miniosdk.RemoveObjectOptions{},
	)
}

// 用户头像逻辑
func (r *MinioRepository) UploadObject(ctx context.Context, bucket, objectKey, contentType string, reader io.Reader, size int64) error {
	if r == nil || r.storage == nil || r.storage.MinioClient == nil {
		return errors.New("minio repository is not initialized")
	}
	if bucket == "" || objectKey == "" {
		return errors.New("bucket and objectKey are required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := r.storage.MinioClient.PutObject(
		ctx,
		bucket,
		objectKey,
		reader,
		size,
		miniosdk.PutObjectOptions{
			ContentType: contentType,
		},
	)
	return err
}

// 预签名
func (r *MinioRepository) PresignedGetURL(ctx context.Context, bucket, objectKey string, expiry time.Duration) (string, error) {
	if r == nil || r.storage == nil || r.storage.MinioClient == nil {
		return "", errors.New("minio repository is not initialized")
	}

	u, err := r.storage.MinioClient.PresignedGetObject(ctx, bucket, objectKey, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

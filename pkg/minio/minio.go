package minio

import (
	"errors"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/wtitdn/renew_video/internal/config"
)

type Minio struct {
	MinioClient *minio.Client
	MinioCore   *minio.Core
}

func NewMinio(cfg *config.MinioConfig) (*Minio, error) {
	if cfg == nil {
		return nil, errors.New("minio config is nil")
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	}
	minioClient, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, err
	}
	minioCore, err := minio.NewCore(cfg.Endpoint, opts)
	if err != nil {
		return nil, err
	}
	a := Minio{minioClient, minioCore}
	return &a, nil
}

func (w *Minio) Close() error {
	return nil
}

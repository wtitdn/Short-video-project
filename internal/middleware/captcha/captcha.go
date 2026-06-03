package captcha

import (
	"context"
	"strings"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/wtitdn/renew_video/pkg/redis"
)

// CaptchaClient stores captcha answers in Redis.
type CaptchaClient struct {
	*base64Captcha.DriverDigit
	redisClient *redis.Client
}

func NewCaptchaRedis(height, width int, redisClient *redis.Client) *CaptchaClient {
	driver := base64Captcha.NewDriverDigit(
		height,
		width,
		4,   // captcha length
		0.7, // max skew/distortion
		80,  // dot count
	)

	return &CaptchaClient{
		DriverDigit: driver,
		redisClient: redisClient,
	}
}

const (
	captchaPrefix = "captcha:"
	expireSeconds = 60
	redisTimeout  = 1 * time.Second
)

// GenerateIdAndImage generates a captcha image and stores its answer.
func (c *CaptchaClient) GenerateIdAndImage(ip string) (id, b64s, ans string, err error) {
	id, content, answer := c.GenerateIdQuestionAnswer()

	item, err := c.DrawCaptcha(content)
	if err != nil {
		return "", "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()

	key := captchaPrefix + ip
	if err := c.redisClient.SetBytes(ctx, key, []byte(answer), expireSeconds*time.Second); err != nil {
		return "", "", "", err
	}

	return id, item.EncodeB64string(), answer, nil
}

// Verify checks whether the submitted answer matches the stored answer.
func (c *CaptchaClient) Verify(ip string, answer string) (match bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()

	key := captchaPrefix + ip
	valBytes, err := c.redisClient.GetBytes(ctx, key)
	if err != nil {
		return false, err
	}
	val := string(valBytes)
	return val == strings.TrimSpace(answer), nil
}

package captcha

import "github.com/mojocn/base64Captcha"

type Config struct {
	Height   int
	Width    int
	Length   int
	MaxSkew  float64
	DotCount int
}

// Generator 验证码生成器接口。
type Generator interface {
	Generate() (id string, b64s string, answer string, err error)
}

// Base64Generator 基于 base64Captcha 的验证码生成器。
type Base64Generator struct{ cfg Config }

// NewBase64Generator 创建 Base64Generator。
func NewBase64Generator(cfg Config) *Base64Generator { return &Base64Generator{cfg: cfg} }

// noopStore base64Captcha.Store 的空实现。
type noopStore struct{}

func (noopStore) Set(string, string) error         { return nil }
func (noopStore) Get(string, bool) string          { return "" }
func (noopStore) Verify(string, string, bool) bool { return false }

// Generate 生成验证码。
func (g *Base64Generator) Generate() (string, string, string, error) {
	driver := base64Captcha.NewDriverDigit(g.cfg.Height, g.cfg.Width, g.cfg.Length, g.cfg.MaxSkew, g.cfg.DotCount)
	c := base64Captcha.NewCaptcha(driver, noopStore{})
	return c.Generate()
}

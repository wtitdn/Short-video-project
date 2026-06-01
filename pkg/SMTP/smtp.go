package smtp

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/wtitdn/renew_video/internal/config"
	"gopkg.in/gomail.v2"
)

type SmtpClient struct {
	dialer *gomail.Dialer
	from   string
}

func NewSMTPClient(cfg config.SMTPConfig) *SmtpClient {
	return &SmtpClient{
		dialer: gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password),
		from:   cfg.From,
	}
}
func (c *SmtpClient) Close() error {
	return nil
}
func (c *SmtpClient) SendCode(to string, code string) error {
	message := gomail.NewMessage()
	message.SetHeader("From", c.from)
	message.SetHeader("To", to)
	message.SetHeader("Subject", "验证码")
	message.SetBody("text/html", fmt.Sprintf(`
        <div style="font-size:16px;">
            <p>您好，您的验证码是：</p>
            <h2 style="color:#1677ff">%s</h2>
            <p>5 分钟内有效，请不要泄露给他人。</p>
        </div>
    `, code))

	return c.dialer.DialAndSend(message)
}

// 生成 6 位数字验证码（最常用、最安全）
func (c *SmtpClient) GenerateCode() string {
	// 验证码字符集：纯数字
	const charset = "0123456789"
	// 验证码长度 6
	length := 6

	code := make([]byte, length)
	for i := range code {
		// 安全随机数
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[num.Int64()]
	}
	return string(code)
}

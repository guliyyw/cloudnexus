package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/logger"
	"go.uber.org/zap"
)

// Sender sends emails via SMTP. Falls back to logging if not configured.
type Sender struct {
	cfg     config.AppConfig
	useSMTP bool
}

func NewSender(cfg config.AppConfig) *Sender {
	useSMTP := cfg.SMTP.Host != "" && cfg.SMTP.Port != 0
	return &Sender{cfg: cfg, useSMTP: useSMTP}
}

func (s *Sender) Send(to, subject, body string) error {
	if !s.useSMTP {
		logger.Log.Info("邮件已记录(未配置SMTP)",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.String("body", body),
		)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTP.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.SMTP.Host, s.cfg.SMTP.Port)
	auth := smtp.PlainAuth("", s.cfg.SMTP.Username, s.cfg.SMTP.Password, s.cfg.SMTP.Host)

	// SECURITY: 强制使用 TLS
	// 端口 465 使用隐式 TLS，端口 587 使用 STARTTLS
	if s.cfg.SMTP.Port == 465 {
		// 隐式 TLS (SMTPS)
		tlsConfig := &tls.Config{
			ServerName: s.cfg.SMTP.Host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS 连接失败: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, s.cfg.SMTP.Host)
		if err != nil {
			return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
		}
		defer client.Close()
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
		if err := client.Mail(s.cfg.SMTP.From); err != nil {
			return fmt.Errorf("设置发件人失败: %w", err)
		}
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
		wc, err := client.Data()
		if err != nil {
			return fmt.Errorf("准备邮件数据失败: %w", err)
		}
		_, err = wc.Write([]byte(msg))
		if err != nil {
			return fmt.Errorf("写入邮件内容失败: %w", err)
		}
		if err := wc.Close(); err != nil {
			return fmt.Errorf("关闭邮件数据失败: %w", err)
		}
		return client.Quit()
	}

	// STARTTLS (端口 587 或其他)
	return smtp.SendMail(addr, auth, s.cfg.SMTP.From, []string{to}, []byte(msg))
}

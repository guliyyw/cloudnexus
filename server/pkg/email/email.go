package email

import (
	"fmt"
	"net/smtp"

	"github.com/cloudnexus/server/pkg/config"
	"github.com/cloudnexus/server/pkg/logger"
	"go.uber.org/zap"
)

// Sender sends emails via SMTP. Falls back to logging if not configured.
type Sender struct {
	cfg    config.AppConfig
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
	return smtp.SendMail(addr, auth, s.cfg.SMTP.From, []string{to}, []byte(msg))
}

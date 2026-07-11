package email

import (
	"dockvol-backend/internal/config"
	"dockvol-backend/internal/util/logger"
)

var (
	env = config.GetEnv()
	log = logger.GetLogger()
)

var emailSMTPSender = &EmailSMTPSender{
	log,
	env.SMTPHost,
	env.SMTPPort,
	env.SMTPUser,
	env.SMTPPassword,
	env.SMTPFrom,
	env.SMTPInsecureSkipVerify,
	env.SMTPHost != "" && env.SMTPPort != 0,
}

func GetEmailSMTPSender() *EmailSMTPSender {
	return emailSMTPSender
}

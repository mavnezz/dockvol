package cloudflare_turnstile

import (
	"dockvol-backend/internal/config"
)

var cloudflareTurnstileService = &CloudflareTurnstileService{
	config.GetEnv().CloudflareTurnstileSecretKey,
	config.GetEnv().CloudflareTurnstileSiteKey,
}

func GetCloudflareTurnstileService() *CloudflareTurnstileService {
	return cloudflareTurnstileService
}

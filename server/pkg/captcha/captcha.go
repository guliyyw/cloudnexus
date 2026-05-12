package captcha

import (
	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

// Manager wraps base64Captcha for easy DI.
type Manager struct {
	captcha *base64Captcha.Captcha
	store   *RedisStore
}

func NewManager(redisClient *redis.Client) *Manager {
	store := NewRedisStore(redisClient)
	driver := base64Captcha.NewDriverString(
		48,  // height
		140, // width
		5,   // noise count (干扰线数量)
		2,  // showLineOptions (OptionShowHollowLine)
		4,   // length (验证码字符数)
		"0123456789abcdefghjkmnpqrstuvwxyz", // 排除易混淆字符
		nil,
		[]string{},
	)
	return &Manager{
		captcha: base64Captcha.NewCaptcha(driver, store),
		store:   store,
	}
}

// Generate creates a captcha and returns the id, base64 image, and answer.
func (m *Manager) Generate() (id string, b64s string, err error) {
	return m.captcha.Generate()
}

// Verify checks the captcha answer.
func (m *Manager) Verify(id, answer string) bool {
	return m.store.Verify(id, answer, true)
}

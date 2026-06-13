package jobs

const (
	TypeDemoLog             = "demo:log"
	TypeCleanupRefreshToken = "auth:cleanup-refresh-tokens"
)

type DemoLog struct {
	Message string `json:"message"`
}

func (DemoLog) Type() string   { return TypeDemoLog }
func (j DemoLog) Payload() any { return j }

type CleanupRefreshTokens struct{}

func (CleanupRefreshTokens) Type() string   { return TypeCleanupRefreshToken }
func (j CleanupRefreshTokens) Payload() any { return j }

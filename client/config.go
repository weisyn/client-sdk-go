package client

// Config 客户端配置
type Config struct {
	// Endpoint 节点端点地址
	Endpoint string
	
	// Protocol 协议类型
	Protocol Protocol
	
	// Timeout 超时时间（秒）
	Timeout int
	
	// TLS 配置
	TLS *TLSConfig
	
	// 调试模式
	Debug bool
	
	// 日志器（可选）
	Logger Logger
}

// Protocol 协议类型
type Protocol string

const (
	ProtocolHTTP      Protocol = "http"
	ProtocolGRPC      Protocol = "grpc"
	ProtocolWebSocket Protocol = "websocket"
)

// TLSConfig TLS 配置
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Insecure bool // 跳过 TLS 验证（仅用于开发）
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoint: "http://localhost:8545",
		Protocol: ProtocolHTTP,
		Timeout:  30,
		Debug:    false,
	}
}


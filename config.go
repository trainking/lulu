package lulu

import (
	"os"

	"gopkg.in/yaml.v3"
)

type (
	// Config gamex的基础配置内容
	Config struct {
		Version          string   `yaml:"Version"`                    // 服务的版本号
		Address          string   `yaml:"Address"`                    // 监听的地址
		NetWork          string   `yaml:"Network"`                    // 传输层协议，tcp, kcp，websocket
		WebsocketPath    string   `yaml:"WebsocketPath,omitempty"`    // websocket时使用升级路径
		KcpMode          string   `yaml:"KcpMode,omitempty"`          // kcp模式，nomarl 普通模式 fast 极速模式；默认极速模式
		ConnReadTimeout  int      `yaml:"ConnReadTimeout,omitempty"`  // 每个连接的读超时(等于客户端心跳的超时)，秒为单位， 默认10秒
		ConnWriteTimeout int      `yaml:"ConnWriteTimeout,omitempty"` // 每个连接的写超时，秒为单位，默认5秒
		ConnMax          int      `yaml:"ConnMax,omitempty"`          // 最大连接数， 默认10000
		ValidTimeout     int      `yaml:"ValidTimeout,omitempty"`     // 有效链接超时；连接成功后，多久未验证身份，则断开，秒为单位, 默认10秒
		HeartLimit       int      `yaml:"HeartLimit,omitempty"`       // 心跳包限制数量, 每分钟不能超过的数量，默认100
		Password         string   `yaml:"Password,omitempty"`         // 密码
		OutUrl           string   `yaml:"OutUrl,omitempty"`           // 外部访问的URL
		TLS              *TLSConf `yaml:"TLS,omitempty"`              // TLS配置
	}

	// TLSConf TLS配置结构体
	TLSConf struct {
		CertFile string `yaml:"CertFile"`
		KeyFile  string `yaml:"KeyFile"`
	}
)

// LoadDefaultAppConfig 读取默认路径下的配置， 路径是项目路径下 configs/lulu.yaml
func LoadDefaultAppConfig() *Config {
	c, err := ReadConfigYaml("configs/lulu.yaml")
	if err != nil {
		panic(err)
	}
	return c
}

// ReadConfigYaml 读取yaml配置文件
func ReadConfigYaml(path string) (*Config, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return nil, err
	}

	config.defaultValue()

	return &config, nil
}

// defaultValue 设置参数的默认值
func (c *Config) defaultValue() {
	if c.WebsocketPath == "" {
		c.WebsocketPath = "/ws"
	}

	if c.KcpMode == "" {
		c.KcpMode = "fast"
	}

	if c.ConnReadTimeout == 0 {
		c.ConnReadTimeout = 10
	}

	if c.ConnWriteTimeout == 0 {
		c.ConnWriteTimeout = 5
	}

	if c.ConnMax == 0 {
		c.ConnMax = 10000
	}

	if c.ValidTimeout == 0 {
		c.ValidTimeout = 10
	}
}

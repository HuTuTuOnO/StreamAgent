package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// 单次任务的硬超时时间，固定写死，不暴露给配置。
	DefaultTimeout = 10 * time.Minute
	// 调度器默认间隔，具体以配置文件里的 cron 表达式为准。
	DefaultInterval = 30 * time.Minute
	// UnlockTests 的默认语言，当前先固定中文输出。
	DefaultLanguage = "zh"
)

// 这里只保留真正需要外部传入的字段，其余保护性参数在代码里固定处理。
type Config struct {
	Debug     bool     `yaml:"debug"`
	Mode      string   `yaml:"mode"`
	API       string   `yaml:"api"`
	Token     string   `yaml:"token"`
	Node      int      `yaml:"node"`
	Stack     string   `yaml:"stack"`
	Scheduler string   `yaml:"scheduler"`
	Exclude   []string `yaml:"exclude"`
}

func Load(path string) (*Config, error) {
	// 读取配置文件并反序列化为 Config。
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	cfg.Stack = strings.ToLower(strings.TrimSpace(cfg.Stack))
	cfg.API = strings.TrimSpace(cfg.API)
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.Scheduler = strings.TrimSpace(cfg.Scheduler)

	for i := range cfg.Exclude {
		cfg.Exclude[i] = strings.TrimSpace(cfg.Exclude[i])
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	// mode 只有两种：client 或 server。
	if c.Mode != "client" && c.Mode != "server" {
		return errors.New("mode must be client or server")
	}
	// API 和 token 是两个模式都需要的公共参数。
	if c.API == "" {
		return errors.New("api is required")
	}
	if c.Token == "" {
		return errors.New("token is required")
	}
	// server 模式下需要明确节点编号，用于上报。
	if c.Mode == "server" && c.Node <= 0 {
		return errors.New("node must be greater than 0 in server mode")
	}
	// stack 只影响 client 模式下节点探测时的地址选择。
	if c.Stack == "" {
		c.Stack = "default"
	}
	switch c.Stack {
	case "ipv4", "ipv6", "default":
	default:
		return errors.New("stack must be ipv4, ipv6, or default")
	}
	// scheduler 使用 6 段 cron 表达式，包含秒字段。
	if c.Scheduler == "" {
		return errors.New("scheduler is required")
	}
	return nil
}

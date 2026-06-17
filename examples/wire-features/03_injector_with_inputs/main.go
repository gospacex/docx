// 03_injector_with_inputs - Injector 带输入参数示例
package main

import (
	"fmt"
)

// =============================================================================
// 定义服务
// =============================================================================

// Config 配置结构
type Config struct {
	ServiceName string
	Port        int
}

// MessageService 上游服务接口
type MessageService interface {
	GetMessage() string
	GetServiceName() string
	GetPort() int
}

type messageService struct {
	cfg Config
}

func NewMessageService(cfg Config) MessageService {
	return &messageService{cfg: cfg}
}

func (s *messageService) GetMessage() string {
	return fmt.Sprintf("Hello from %s on port %d!", s.cfg.ServiceName, s.cfg.Port)
}

func (s *messageService) GetServiceName() string {
	return s.cfg.ServiceName
}

func (s *messageService) GetPort() int {
	return s.cfg.Port
}

// =============================================================================
// 手动 DI 版本（不依赖 Wire）
// =============================================================================

type ManualInjector struct {
	MessageService MessageService
}

func NewManualInjector(cfg Config) *ManualInjector {
	return &ManualInjector{
		MessageService: NewMessageService(cfg),
	}
}

// =============================================================================
// 主函数
// =============================================================================

func main() {
	fmt.Println("=== Wire Feature 03: Injector with Inputs ===\n")

	// 手动 DI - 直接传递配置参数
	cfg := Config{
		ServiceName: "MyService",
		Port:        8080,
	}
	inj := NewManualInjector(cfg)
	fmt.Printf("✅ Manual DI: %s\n", inj.MessageService.GetMessage())

	// Wire DI 说明
	fmt.Println("\n使用 Wire DI (带输入参数):")
	fmt.Println("1. 创建 wire.go (//go:build wireinject)")
	fmt.Println("2. 定义 Provider 函数 (NewMessageService 接收 Config 参数)")
	fmt.Println("3. Injector 注入器函数也接收 Config 参数")
	fmt.Println("4. Injector 内部通过 wire.Build 注入依赖")
	fmt.Println("5. 运行: wire ./...")
}

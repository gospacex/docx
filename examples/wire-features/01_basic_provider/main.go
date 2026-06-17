// 01_basic_provider - 基础 Provider 示例
package main

import (
	"fmt"
)

// =============================================================================
// 定义服务
// =============================================================================

type MessageService interface {
	GetMessage() string
}

type messageService struct {
	msg string
}

func NewMessageService() MessageService {
	return &messageService{msg: "Hello, World!"}
}

func (s *messageService) GetMessage() string {
	return s.msg
}

// =============================================================================
// 手动 DI 版本（不依赖 Wire）
// =============================================================================

type ManualInjector struct {
	MessageService MessageService
}

func NewManualInjector() *ManualInjector {
	return &ManualInjector{
		MessageService: NewMessageService(),
	}
}

// =============================================================================
// 主函数
// =============================================================================

func main() {
	fmt.Println("=== Wire Feature 01: Basic Provider ===\n")

	// 手动 DI
	inj := NewManualInjector()
	fmt.Printf("✅ Manual DI: %s\n", inj.MessageService.GetMessage())

	// Wire DI 说明
	fmt.Println("\n使用 Wire DI:")
	fmt.Println("1. 创建 wire.go (//go:build wireinject)")
	fmt.Println("2. 定义 Provider 函数 (NewMessageService)")
	fmt.Println("3. wire.Build(NewMessageService, wire.Struct(...))")
	fmt.Println("4. 运行: wire ./...")
}

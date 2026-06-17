// QuickStart - Wire DI 快速入门示例
// 演示如何通过 Wire 注入单例服务
package main

import (
	"fmt"
)

// =============================================================================
// 服务定义
// =============================================================================

// Greeter 问候服务接口
type Greeter interface {
	Greet() string
}

// greeter 单例实例
type greeter struct {
	name string
}

// Greet 返回问候语
func (g *greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.name)
}

// NewGreeter 创建 Greeter 单例
func NewGreeter() Greeter {
	return &greeter{name: "World"}
}

// =============================================================================
// Injector 定义（Wire DI 生成的代码会填充这里）
// =============================================================================

// Injector 所有 provider 的统一注入器
//
//go:generate go run -mod=mod github.com/google/wire/cmd/wire
var injector *Injector

// =============================================================================
// Manual DI 版本（不依赖 Wire）
// =============================================================================

// ManualInjector 手动依赖注入版本
type ManualInjector struct {
	Greeter Greeter
}

// NewManualInjector 手动创建 Injector
func NewManualInjector() *ManualInjector {
	return &ManualInjector{
		Greeter: NewGreeter(),
	}
}

// =============================================================================
// 主函数
// =============================================================================

func main() {
	fmt.Println("=== Hubx QuickStart: Wire DI 单例 ===\n")

	// -------------------------------------------------------------------
	// 方式一: 手动 DI（无需 Wire）
	// -------------------------------------------------------------------
	fmt.Println("[方式一] 手动依赖注入（无需 Wire）")
	manualInjector := NewManualInjector()
	fmt.Printf("  Greeter: %s\n", manualInjector.Greeter.Greet())
	fmt.Println()

	// -------------------------------------------------------------------
	// 方式二: Wire DI（需要 wire 工具）
	// -------------------------------------------------------------------
	fmt.Println("[方式二] Wire DI（需要 wire ./... 生成 wire_gen.go）")
	fmt.Println("  injector := InitializeInjector()")
	fmt.Println("  injector.Greeter.Greet()")
	fmt.Println()

	// -------------------------------------------------------------------
	// 使用 Wire 生成的注入器
	// -------------------------------------------------------------------
	fmt.Println("[使用 Wire 注入器]")
	injector, err := InitializeInjector()
	if err != nil {
		fmt.Printf("  ❌ 初始化失败: %v\n", err)
		return
	}
	fmt.Printf("  Greeter>: %s\n", injector.Greeter.Greet())
	fmt.Println()

	// -------------------------------------------------------------------
	// Wire DI 模式说明
	// -------------------------------------------------------------------
	fmt.Println("=== Wire DI 模式说明 ===")
	fmt.Println()
	fmt.Println("1. 定义服务接口和实现:")
	fmt.Println("   type Greeter interface { Greet() string }")
	fmt.Println("   type greeter struct { name string }")
	fmt.Println("   func NewGreeter() Greeter { return &greeter{name: \"World\"} }")
	fmt.Println()
	fmt.Println("2. 创建 wire.go:")
	fmt.Println("   //go:build wireinject")
	fmt.Println("   func InitializeInjector() (*Injector, error) {")
	fmt.Println("       wire.Build(NewGreeter, wire.Struct(new(Injector), \"*\"))")
	fmt.Println("       return nil, nil")
	fmt.Println("   }")
	fmt.Println()
	fmt.Println("3. 运行 wire 工具:")
	fmt.Println("   wire ./...")
	fmt.Println()
	fmt.Println("4. 使用生成的注入器:")
	fmt.Println("   injector := InitializeInjector()")
	fmt.Println("   injector.Greeter.Greet()")
	fmt.Println()
	fmt.Println("=== 完成 ===")
}

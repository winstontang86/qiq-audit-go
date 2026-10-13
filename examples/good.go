// Package examples 存放 qiq-audit-go 的自我回归夹具。
// good.go 是符合《Golang 代码规范》的示范：预期 0 个【必须】违规、0 个【推荐】违规。
package examples

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// TaxRate 增值税率
const TaxRate = 0.05

// DefaultPageSize 默认分页大小
const DefaultPageSize = 20

// Order 订单基础信息
type Order struct {
	ID        string
	URL       string
	Amount    float64
	CreatedAt time.Time
}

// Fetcher 按订单号拉取订单
type Fetcher interface {
	// Fetch 按订单号返回订单，未找到时返回 error
	Fetch(id string) (*Order, error)
}

// Loader 从订单文件加载数据
type Loader struct {
	path string
}

// NewLoader 创建订单加载器
func NewLoader(path string) *Loader {
	return &Loader{path: path}
}

// Fetch 读取文件并返回指定订单
func (l *Loader) Fetch(id string) (*Order, error) {
	file, err := os.Open(l.path)
	if err != nil {
		return nil, fmt.Errorf("open order file: %w", err)
	}
	defer file.Close()

	orders := make([]Order, 0)
	if err := json.NewDecoder(file).Decode(&orders); err != nil {
		return nil, fmt.Errorf("decode order file: %w", err)
	}
	for i := range orders {
		if orders[i].ID == id {
			return &orders[i], nil
		}
	}

	return nil, errors.New("order not found")
}

// TotalPrice 返回含税金额
func TotalPrice(amount float64) float64 {
	return (1.0 + TaxRate) * amount
}

// Status 订单状态
type Status string

// 订单状态枚举
const (
	// StatusPaid 已支付
	StatusPaid Status = "paid"
	// StatusCanceled 已取消
	StatusCanceled Status = "canceled"
)

// Describe 返回订单状态的可读描述
func Describe(status Status, page int, size int) string {
	if size <= 0 {
		size = DefaultPageSize
	}
	if page < 0 {
		page = 0
	}

	switch status {
	case StatusPaid:
		return fmt.Sprintf("paid page=%d size=%d", page, size)
	case StatusCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

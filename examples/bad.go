// 本文件是 qiq-audit-go 的自我回归夹具：故意写成不符合《Golang 代码规范》的样子。
// 预期命中多条【必须】/【推荐】违规；行尾 “// ←” 标记规则编号，编号含义见 references/checklist.md。
// 请勿格式化本文件：其中的格式问题用于验证 GO-2.1-FORMAT 与 GO-2.3-SPACE。
//
// 规则对应关系：
//   GO-2.1-FORMAT / GO-2.3-SPACE  OrderData 字段未对齐、attempt 赋值多余空格
//   GO-2.2-LINE-WIDTH             Query 中的 longLine 语句
//   GO-2.4-IMPORT                 第三方包未单独分组、包名可用却加别名
//   GO-2.5.1-ERROR-HANDLE         os.Remove 的返回值未处理
//   GO-2.5.1-ERROR-LAST           Query 返回 (error, *OrderData)
//   GO-2.5.1-ERROR-TEXT           "bad service client is nil."
//   GO-2.5.1-ERROR-FLOW           Query 中的 if/else 正常流
//   GO-2.5.1-ERROR-COND           readErr != nil || content == nil
//   GO-2.5.1-ERROR-NEW            fmt.Errorf 无格式化参数
//   GO-2.5.1-ERROR-WRAP           fmt.Errorf 用 %v 丢错误链
//   GO-2.5.2-PANIC                用 panic 处理 size 用户输入
//   GO-2.5.3-RECOVER              recover 不在 defer 中
//   GO-3.2-STRUCT-DOC             OrderData / FetchData 缺文档
//   GO-3.3-FUNC-DOC               FetchData.Fetch 缺文档
//   GO-3.4-VALUE-DOC              App_Version / Full_Name 缺文档
//   GO-4.3-STRUCT-NAME            OrderData 使用宽泛词 Data
//   GO-4.3-STRUCT-LAYOUT          &OrderData{...} 单行初始化
//   GO-4.4-INTERFACE-NAME         FetchData 单方法接口未以 er 结尾
//   GO-4.5-VAR-NAME               Id / Url / Full_Name
//   GO-4.6-CONST-NAME             App_Version
//   GO-5.1-IF-COMPARE             nil != this.client
//   GO-5.1-IF-BOOL                allow == true
//   GO-5.3-RANGE                  只用第二项却保留 idx
//   GO-5.4-SWITCH                 status switch 缺 default
//   GO-5.6-GOTO                   业务代码使用 goto retry
//   GO-6.1-PARAM-COUNT            Query 有 6 个参数
//   GO-6.1-PARAM-NAME             Set 的参数 Name 大写开头
//   GO-6.1-PARAM-VALUE            updateIndex 传递 map 指针
//   GO-6.1-RESULT-NAME            Count 单个结果使用命名返回
//   GO-6.2-DEFER-LOOP             循环内 defer rows.Close()
//   GO-6.2-DEFER-RESOURCE         Copy 未判断错误就 defer
//   GO-6.3-RECEIVER-BANNED        Query 接收器 this
//   GO-6.5-NESTING                DeepNest 嵌套 5 层
//   GO-6.7-MAGIC                  1.05 税率魔法数字
//
// 包注释由 good.go 提供，故本文件不演示 GO-3.1-PACKAGE-DOC；本目录 package 名与目录名一致，
// 也不演示 GO-4.1-PACKAGE-NAME；项目级规则（GO-7.1-MODULES、GO-7.2-MODULE-NAME、GO-7.2-VENDOR、
// GO-7.2-GO-SUM、GO-8.1-README、GO-8.2-API-TEST）无法由单文件夹具覆盖。
// 单测相关违规见 bad_test.go。
package examples

import (
	"encoding/json"
	"errors"
	"fmt"
	proto "github.com/golang/protobuf/proto" // ← GO-2.4-IMPORT
	"os"
	"strings"
	"time"
)

type OrderData struct { // ← GO-3.2-STRUCT-DOC GO-4.3-STRUCT-NAME
	Id       string  // ← GO-4.5-VAR-NAME
	Url      string  // ← GO-4.5-VAR-NAME
	Amount  float64
	Data     []byte
	CreateAt time.Time
}

const App_Version = "1.0.0" // ← GO-4.6-CONST-NAME GO-3.4-VALUE-DOC

var Full_Name = "qiq" // ← GO-4.5-VAR-NAME GO-3.4-VALUE-DOC

type FetchData interface { // ← GO-3.2-STRUCT-DOC GO-4.4-INTERFACE-NAME
	Fetch(id string) ([]byte, error) // ← GO-3.3-FUNC-DOC
}

// BadService 演示接收器命名、错误处理与控制结构问题
type BadService struct {
	client *proto.Buffer
	index  map[string]string
}

// Query 故意违反多条必须规则
func (this *BadService) Query(uid string, orderID string, status string, page int, size int, extra string) (error, *OrderData) { // ← GO-6.3-RECEIVER-BANNED GO-2.5.1-ERROR-LAST GO-6.1-PARAM-COUNT
	var order *OrderData
	var err error

	if nil != this.client { // ← GO-5.1-IF-COMPARE
		return errors.New("bad service client is nil."), nil // ← GO-2.5.1-ERROR-TEXT
	}

	price := 19.9

	file, openErr := os.Open("/tmp/order.csv")
	if openErr != nil {
		fmt.Fprintln(os.Stderr, openErr)
	} else { // ← GO-2.5.1-ERROR-FLOW
		content, readErr := os.ReadFile(file.Name())
		if readErr != nil || content == nil { // ← GO-2.5.1-ERROR-COND
			return readErr, nil
		}
		order = &OrderData{Id: uid, Url: extra, Amount: 1.05 * price} // ← GO-4.3-STRUCT-LAYOUT GO-6.7-MAGIC
		fmt.Fprintln(os.Stderr, string(content))
	}

	os.Remove("/tmp/order.tmp") // ← GO-2.5.1-ERROR-HANDLE

	for _, item := range strings.Split(extra, ",") {
		rows, queryErr := os.Open(item)
		if queryErr != nil {
			continue
		}
		defer rows.Close() // ← GO-6.2-DEFER-LOOP
		fmt.Fprintln(os.Stderr, rows.Name())
	}

	switch status {
	case "paid":
		order.Data = []byte(status)
	case "canceled":
		order.Data = nil
	} // ← GO-5.4-SWITCH

	sum := 0
	for idx, item := range this.index {
		_ = idx
		sum += len(item) // ← GO-5.3-RANGE
	}

	allow := sum > 0
	if allow == true { // ← GO-5.1-IF-BOOL
		order.Amount = 1.05 * price // ← GO-6.7-MAGIC
	}

	if size > 100 {
		panic(fmt.Sprintf("invalid page size: %d", size)) // ← GO-2.5.2-PANIC
	}

	if recovered := recover(); recovered != nil { // ← GO-2.5.3-RECOVER
		fmt.Fprintln(os.Stderr, recovered)
	}

	attempt  := 0 // ← GO-2.3-SPACE
retry:
	attempt++
	if attempt < 3 {
		goto retry // ← GO-5.6-GOTO
	}

	longLine := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s|%s", uid, orderID, status, extra, uid, orderID, status, extra, uid, orderID) // ← GO-2.2-LINE-WIDTH
	fmt.Fprintln(os.Stderr, longLine)

	return err, order
}

// DeepNest 演示嵌套层级超过 4 层
func (s *BadService) DeepNest(items []string) int {
	total := 0
	for _, item := range items { // 第 1 层
		if item != "" { // 第 2 层
			for _, char := range item { // 第 3 层
				if char != ' ' { // 第 4 层
					for _, other := range items { // 第 5 层 ← GO-6.5-NESTING
						if other == item {
							total++
						}
					}
				}
			}
		}
	}
	return total
}

// Copy 演示 defer 与错误检查的顺序问题
func (s *BadService) Copy(path string) error {
	file, err := os.Open(path)
	defer file.Close() // ← GO-6.2-DEFER-RESOURCE
	if err != nil {
		return err
	}
	_, _ = file.Stat()
	return nil
}

// Load 演示错误构造与包装问题
func (s *BadService) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config: %v", err) // ← GO-2.5.1-ERROR-WRAP
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&s.index); err != nil {
		return fmt.Errorf("decode config failed") // ← GO-2.5.1-ERROR-NEW
	}
	return nil
}

// Count 演示命名返回问题
func (s *BadService) Count() (total int) { // ← GO-6.1-RESULT-NAME
	return len(s.index)
}

// Set 演示参数命名问题
func (s *BadService) Set(Name string) { // ← GO-6.1-PARAM-NAME
	s.index[Name] = Name
}

func updateIndex(index *map[string]string, key string, value string) { // ← GO-6.1-PARAM-VALUE
	(*index)[key] = value
}

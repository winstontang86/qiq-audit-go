// 本文件是 qiq-audit-go 的自我回归夹具：演示单测相关违规
//（GO-2.6-TEST-NAME、GO-2.6-TEST-STRUCT）。
// 单测文件行数上限为 1600 行、单测函数体上限为 160 行，本项目夹具不演示超长场景。
package examples

import "testing"

// TestOrderFixture 单测结构体尽量不要导出
type TestOrderFixture struct { // ← GO-2.6-TEST-STRUCT：单测结构体定义尽量不导出
	ID string
}

// TestBad_Query 下划线位置不符合规范：合法写法为 TestBadService_Query
func TestBad_Query(t *testing.T) { // ← GO-2.6-TEST-NAME：下划线只能用于 Test_Foo / TestBar_Foo
	fixture := TestOrderFixture{ID: "1"}
	if fixture.ID == "" {
		t.Fatal("empty id")
	}
}

// TestBadService_Query 对应 func (s *BadService) Query 的合法命名
func TestBadService_Query(t *testing.T) {
	fixture := TestOrderFixture{ID: "1"}
	if fixture.ID != "1" {
		t.Fatalf("unexpected id %q", fixture.ID)
	}
}

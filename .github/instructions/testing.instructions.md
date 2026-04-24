# 单元测试指南

## 概述

本项目使用 Go 标准库的 `testing` 包进行单元测试。测试文件以 `_test.go` 结尾，与被测试的源文件放在同一目录中。

## 运行测试

### 运行所有测试

```bash
make test
# 或
go test -v ./...
```

### 运行特定包的测试

```bash
go test -v ./pkg/storage
go test -v ./pkg/db
```

### 运行特定测试函数

```bash
go test -v -run TestGetSkillsDir ./pkg/storage
```

### CLI 回归验证

当命令结构发生调整时，除了单元测试，建议补一轮基础 smoke test：

```bash
go run . add -h
go run . list --repo
go run . install -r demo-repo demo-skill -h
go run . install sync-repo -h
go run . install default-repo -h
go run . remove --repo demo-repo
go run . search -r demo-repo demo-skill -h
go run . search --global demo-skill -h
go run . status downloads -h
go run . update demo-skill -r demo-repo -h
```

如果要验证旧入口已经移除，可以额外执行：

```bash
go run . repo -h
```

预期结果是返回 `unknown command "repo"`。

### 生成测试覆盖率报告

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 测试文件结构

```
pkg/
├── storage/
│   ├── storage.go          # 源代码
│   └── storage_test.go     # 测试代码
├── db/
│   ├── database.go         # 源代码
│   └── db_test.go          # 测试代码
└── repository/
    ├── repository.go       # 源代码
    └── repository_test.go  # 测试代码
```

## 测试命名规范

- 测试文件：`filename_test.go`
- 测试函数：`TestFunctionName`（首字母大写）
- 子测试：`t.Run("SubtestName", func(t *testing.T) { ... })`

## 测试示例

### 基本测试

```go
func TestGetSkillsDir(t *testing.T) {
    dir, err := GetSkillsDir()
    if err != nil {
        t.Fatalf("GetSkillsDir() error = %v", err)
    }

    if dir == "" {
        t.Error("GetSkillsDir() returned empty string")
    }
}
```

### 使用临时文件

```go
func TestAddSkill(t *testing.T) {
    // 创建临时目录
    tempDir, err := os.MkdirTemp("", "skill-test-*")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir) // 测试结束后清理

    // 执行测试逻辑
    // ...
}
```

### 使用临时环境变量

```go
func TestEnsureSkillsDir(t *testing.T) {
    // 保存原始环境变量
    oldHome := os.Getenv("HOME")
    defer os.Setenv("HOME", oldHome)

    // 设置临时环境变量
    tempDir := "/tmp/test-dir"
    os.Setenv("HOME", tempDir)

    // 执行测试
    err := EnsureSkillsDir()
    if err != nil {
        t.Fatalf("EnsureSkillsDir() error = %v", err)
    }
}
```

### 表驱动测试

```go
func TestParseSkillName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid skill name",
            input:    "my-skill",
            expected: "my-skill",
            wantErr:  false,
        },
        {
            name:     "empty skill name",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ParseSkillName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSkillName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("ParseSkillName() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## 测试最佳实践

### 1. 使用 `defer` 清理资源

```go
func TestDatabaseConnection(t *testing.T) {
    tempFile, err := os.CreateTemp("", "test-*.db")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tempFile.Name()) // 确保测试后清理

    // 测试逻辑
}
```

### 2. 使用子测试组织相关测试

```go
func TestSkillOperations(t *testing.T) {
    t.Run("AddSkill", func(t *testing.T) {
        // 测试添加技能
    })

    t.Run("GetSkill", func(t *testing.T) {
        // 测试获取技能
    })

    t.Run("RemoveSkill", func(t *testing.T) {
        // 测试删除技能
    })
}
```

### 3. 测试错误情况

```go
func TestGetSkill_NotFound(t *testing.T) {
    _, err := GetSkill("non-existent-skill")
    if err == nil {
        t.Error("GetSkill() should return error for non-existent skill")
    }
}
```

### 4. 使用 `t.Fatal` vs `t.Error`

- `t.Fatal()`：测试立即失败，停止当前测试函数
- `t.Error()`：记录错误但继续执行测试

```go
func TestExample(t *testing.T) {
    // 关键失败，立即停止
    if err != nil {
        t.Fatalf("Setup failed: %v", err)
    }

    // 非关键错误，记录后继续
    if result != expected {
        t.Errorf("Result mismatch: got %v, want %v", result, expected)
    }
}
```

## 模拟和依赖注入

对于需要外部依赖的测试，可以考虑使用接口和模拟：

```go
// 定义接口
type Storage interface {
    Save(name string, data []byte) error
    Load(name string) ([]byte, error)
}

// 创建模拟实现
type MockStorage struct {
    SaveFunc func(name string, data []byte) error
    LoadFunc func(name string) ([]byte, error)
}

func (m *MockStorage) Save(name string, data []byte) error {
    if m.SaveFunc != nil {
        return m.SaveFunc(name, data)
    }
    return nil
}

func (m *MockStorage) Load(name string) ([]byte, error) {
    if m.LoadFunc != nil {
        return m.LoadFunc(name)
    }
    return nil, nil
}

// 在测试中使用
func TestWithMock(t *testing.T) {
    mock := &MockStorage{
        SaveFunc: func(name string, data []byte) error {
            t.Logf("Saving %s", name)
            return nil
        },
    }

    // 将 mock 传递给被测试的函数
    err := ProcessData(mock, "test", []byte("data"))
    if err != nil {
        t.Errorf("ProcessData() error = %v", err)
    }
}
```

## 持续集成

在 CI/CD 流程中运行测试：

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.22
      - name: Run tests
        run: make test
      - name: Upload coverage
        uses: codecov/codecov-action@v2
```

## 参考资源

- [Go 官方测试文档](https://golang.org/pkg/testing/)
- [Go Testing Examples](https://golang.org/doc/code.html#Testing)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

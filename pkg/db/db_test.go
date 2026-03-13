package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
)

func TestOpenDB(t *testing.T) {
	// 临时修改环境变量以使用临时数据库
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", os.TempDir())
	defer os.Setenv("APPDATA", oldAppData)

	// 创建临时目录
	testDir := os.TempDir() + "/skill"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 测试创建数据库连接
	db, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Fatal("OpenDB() returned nil")
	}
}

func TestNewManager(t *testing.T) {
	// 临时修改环境变量以使用临时数据库
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", os.TempDir())
	defer os.Setenv("APPDATA", oldAppData)

	// 创建临时目录
	testDir := os.TempDir() + "/skill"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 测试创建管理器
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	queries := manager.GetQueries()
	if queries == nil {
		t.Fatal("GetQueries() returned nil")
	}
}

func TestGetCurrentVersion(t *testing.T) {
	// 临时修改环境变量以使用临时数据库
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", os.TempDir())
	defer os.Setenv("APPDATA", oldAppData)

	// 创建临时目录
	testDir := os.TempDir() + "/skill"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建数据库连接
	sqlDB, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer sqlDB.Close()

	// 初始化数据库 schema
	_, err = sqlDB.Exec(`
		CREATE TABLE db_version (
			version INTEGER NOT NULL PRIMARY KEY,
			applied_at TEXT NOT NULL,
			description TEXT
		);
		INSERT INTO db_version (version, applied_at, description) 
		VALUES (1, datetime('now'), 'Initial schema');
	`)
	if err != nil {
		t.Fatalf("Failed to create db_version table: %v", err)
	}

	// 创建管理器
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试获取当前版本
	version, err := manager.GetQueries().GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion() error = %v", err)
	}

	if version != 1 {
		t.Errorf("GetCurrentVersion() = %d, expected 1", version)
	}
}

func TestInsertVersion(t *testing.T) {
	// 临时修改环境变量以使用临时数据库
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", os.TempDir())
	defer os.Setenv("APPDATA", oldAppData)

	// 创建临时目录
	testDir := os.TempDir() + "/skill"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建数据库连接
	sqlDB, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer sqlDB.Close()

	// 初始化数据库 schema
	_, err = sqlDB.Exec(`
		CREATE TABLE db_version (
			version INTEGER NOT NULL PRIMARY KEY,
			applied_at TEXT NOT NULL,
			description TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create db_version table: %v", err)
	}

	// 创建管理器
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试插入版本
	params := InsertVersionParams{
		Version:     2,
		AppliedAt:   "2024-01-01T00:00:00Z",
		Description: sql.NullString{String: "Test migration", Valid: true},
	}

	err = manager.GetQueries().InsertVersion(ctx, params)
	if err != nil {
		t.Fatalf("InsertVersion() error = %v", err)
	}

	// 验证版本是否插入成功
	version, err := manager.GetQueries().GetCurrentVersion(ctx)
	if err != nil {
		t.Fatalf("GetCurrentVersion() error = %v", err)
	}

	if version != 2 {
		t.Errorf("GetCurrentVersion() = %d, expected 2", version)
	}
}

func TestListAllVersions(t *testing.T) {
	// 临时修改环境变量以使用临时数据库
	oldAppData := os.Getenv("APPDATA")
	os.Setenv("APPDATA", os.TempDir())
	defer os.Setenv("APPDATA", oldAppData)

	// 创建临时目录
	testDir := os.TempDir() + "/skill"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建数据库连接
	sqlDB, err := OpenDB()
	if err != nil {
		t.Fatalf("OpenDB() error = %v", err)
	}
	defer sqlDB.Close()

	// 初始化数据库 schema
	_, err = sqlDB.Exec(`
		CREATE TABLE db_version (
			version INTEGER NOT NULL PRIMARY KEY,
			applied_at TEXT NOT NULL,
			description TEXT
		);
		INSERT INTO db_version (version, applied_at, description) 
		VALUES (1, datetime('now'), 'Initial schema'),
		       (2, datetime('now'), 'Add new feature');
	`)
	if err != nil {
		t.Fatalf("Failed to create db_version table: %v", err)
	}

	// 创建管理器
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// 测试列出所有版本
	versions, err := manager.GetQueries().ListAllVersions(ctx)
	if err != nil {
		t.Fatalf("ListAllVersions() error = %v", err)
	}

	if len(versions) != 2 {
		t.Errorf("ListAllVersions() returned %d versions, expected 2", len(versions))
	}

	if versions[0].Version != 1 {
		t.Errorf("ListAllVersions()[0].Version = %d, expected 1", versions[0].Version)
	}

	if versions[1].Version != 2 {
		t.Errorf("ListAllVersions()[1].Version = %d, expected 2", versions[1].Version)
	}
}
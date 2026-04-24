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

func TestEnsureDBExistsCreatesCoreTables(t *testing.T) {
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

	if err := EnsureDBExists(); err != nil {
		t.Fatalf("EnsureDBExists() error = %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	var count int
	err = manager.GetDB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='repositories'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check repositories table: %v", err)
	}
	if count != 1 {
		t.Fatalf("repositories table missing")
	}

	err = manager.GetDB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='skill_downloads'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check skill_downloads table: %v", err)
	}
	if count != 1 {
		t.Fatalf("skill_downloads table missing")
	}
}

func TestEnsureDBExistsDropsLegacyVersionTable(t *testing.T) {
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

	if err := EnsureDBExists(); err != nil {
		t.Fatalf("EnsureDBExists() error = %v", err)
	}

	var count int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='db_version'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check db_version table: %v", err)
	}
	if count != 0 {
		t.Fatalf("db_version table should be dropped")
	}
}

func TestRecordSkillDownload(t *testing.T) {
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

	if err := EnsureDBExists(); err != nil {
		t.Fatalf("EnsureDBExists() error = %v", err)
	}

	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	err = manager.RecordSkillDownload(ctx, CreateSkillDownloadParams{
		SkillName:      "pdf",
		InstalledAs:    "pdf",
		SkillVersion:   sqlNullStringForTest("1.0.0"),
		SourceType:     "indexed-repository",
		SourceName:     sqlNullStringForTest("official"),
		SourceUrl:      sqlNullStringForTest("https://example.com/repo.git"),
		Downloader:     "tester",
		DownloaderHost: sqlNullStringForTest("ci-host"),
		DownloadedAt:   "2026-03-26T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("RecordSkillDownload() error = %v", err)
	}

	records, err := manager.GetQueries().ListSkillDownloads(ctx)
	if err != nil {
		t.Fatalf("ListSkillDownloads() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("ListSkillDownloads() returned %d records, expected 1", len(records))
	}

	if records[0].SkillName != "pdf" {
		t.Errorf("SkillName = %s, expected pdf", records[0].SkillName)
	}

	if records[0].Downloader != "tester" {
		t.Errorf("Downloader = %s, expected tester", records[0].Downloader)
	}
}

func sqlNullStringForTest(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
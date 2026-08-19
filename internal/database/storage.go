// Storage：连接管理、建库建表、生命周期(Close)
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosRestful"
	"github.com/zhuofeng1023/mqbot/internal/config"
)

// StatusPoint 一条待落盘的状态数据点
type StatusPoint struct {
	RobotID string
	TS      int64 // 时间戳
	State   string
	Battery float64
	X, Y    float64
	Speed   float64
}

type Storage struct {
	db        *sql.DB
	writer    *Writer
	Dropped   atomic.Int64 // 因 channel 满被丢弃的总数
	cfg       config.DatabaseConfig
}

// 创建 Storage
func NewStorage(cfg config.DatabaseConfig) (*Storage, error) {
	dsn := fmt.Sprintf("%s:%s@http(%s:%d)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	// 打开数据库连接
	db, err := sql.Open("taosRestful", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接错误: %w", err)
	}

	s := &Storage{db: db, cfg: cfg}
	if err := s.initDatabase(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	s.writer = NewWriter(db, &cfg.Write)
	go s.writer.Run()

	return s, nil
}

// 初始化数据库： 创建数据库 & 创建超级表
func (s *Storage) initDatabase(ctx context.Context) error {
	dbName := "MQBOT"

	// 创建数据库 (如果不存在)
	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s KEEP 3650 DAYS 10 BLOCKS 16;", dbName)
	if _, err := s.db.ExecContext(ctx, createDB); err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}

	// 创建超级表 (如果不存在)
	createSTable := fmt.Sprintf(`CREATE STABLE IF NOT EXISTS %s.robot_status (
		ts TIMESTAMP, state BINARY(20), battery FLOAT, x DOUBLE, y DOUBLE, speed FLOAT
	) TAGS (robot_id BINARY(50));`, dbName)
	
	if _, err := s.db.ExecContext(ctx, createSTable); err != nil {
		return fmt.Errorf("创建超级表失败: %w", err)
	}

	return nil
}

// 向数据库写入数据
func (s *Storage) Write(p StatusPoint) {
	if s.writer != nil {
		s.writer.Write(p)
	}
}

// 关闭数据库实例
func (s *Storage) Close() error {
	if s.writer != nil {
		s.writer.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

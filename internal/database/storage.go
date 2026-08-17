// Storage：连接管理、建库建表、生命周期(Close)
package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	_ "github.com/taosdata/driver-go/v3/taosRestful" // 注册驱动，匿名导入
	"github.com/zhuofeng1023/mqbot/internal/config"
)

// StatusPoint 一条待落盘的状态数据点
type StatusPoint struct {
	RobotID string
	TS      int64 // 毫秒时间戳，handStatus 收到消息时取 time.Now().UnixMilli()
	State   string
	Battery float64
	X, Y    float64
	Speed   float64
}

type Storage struct {
	db        *sql.DB
	writer    *Writer
	Dropped   atomic.Int64 // 因 channel 满被丢弃的总数
	WriteErrs atomic.Int64 // 刷写失败次数
	Written   atomic.Int64 // 成功落盘条数
	cfg       config.DatabaseConfig
}

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

func (s *Storage) initDatabase(ctx context.Context) error {
	dbName := "MQBOT"

	createDB := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s KEEP 3650 DAYS 10 BLOCKS 16;", dbName)
	if _, err := s.db.ExecContext(ctx, createDB); err != nil {
		return err
	}

	useDB := fmt.Sprintf("USE %s;", dbName)
	if _, err := s.db.ExecContext(ctx, useDB); err != nil {
		return err
	}

	createSTable := `CREATE STABLE IF NOT EXISTS robot_status (
		ts TIMESTAMP, state BINARY(20), battery FLOAT, x DOUBLE, y DOUBLE, speed FLOAT
	) TAGS (robot_id BINARY(50));`
	_, err := s.db.ExecContext(ctx, createSTable)
	return err
}

func (s *Storage) Write(p StatusPoint) {
	if s.writer != nil {
		s.writer.Write(p)
	}
}

func (s *Storage) Close() error {
	if s.writer != nil {
		s.writer.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

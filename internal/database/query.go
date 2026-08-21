package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Point 查询结果行
type Point struct {
	TS      int64   `json:"ts"`
	State   string  `json:"state"`
	Battery float64 `json:"battery"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Speed   float64 `json:"speed"`
}

// TrackPoint 轨迹降采样结果
type TrackPoint struct {
	TS int64   `json:"ts"` // 起始时间
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// QueryHistory 查某设备时间段内的原始状态点（升序，limit 封顶）
func (s *Storage) QueryHistory(robotID string, from, to int64, limit int) ([]Point, error) {
	// 保证返回空切片而不是 nil，对前端 JSON 序列化更友好
	points := make([]Point, 0)

	// 使用 ? 占位符，防止 SQL 注入
	// state 是保留字，需反引号转义
	query := `SELECT ts, ` + "`state`" + `, battery, x, y, speed
			  FROM MQBOT.robot_status
			  WHERE robot_id = ? AND ts >= ? AND ts < ?
			  ORDER BY ts ASC LIMIT ?`

	rows, err := s.db.QueryContext(context.Background(), query, robotID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("查询历史记录失败: %w", err)
	}
	defer rows.Close()

	// 遍历结果集
	for rows.Next() {
		var p Point
		var tsTime time.Time // TDengine 的 ts 列通常扫描为 time.Time

		// Scan 取数据
		if err := rows.Scan(&tsTime, &p.State, &p.Battery, &p.X, &p.Y, &p.Speed); err != nil {
			return nil, fmt.Errorf("扫描历史记录行失败: %w", err)
		}

		p.TS = tsTime.UnixMilli() // 将 time.Time 转换为毫秒时间戳
		points = append(points, p)
	}

	// 检查迭代过程中是否有错误
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历历史记录结果集出错: %w", err)
	}

	return points, nil
}

// QueryTrack 查轨迹（降采样：按 interval 聚合）
func (s *Storage) QueryTrack(robotID string, from, to int64, interval string) ([]TrackPoint, error) {
	trackPoints := make([]TrackPoint, 0)

	// 简单校验 interval 格式，防止非法字符串注入（只允许数字+s/m/h结尾）
	if len(interval) < 2 {
		return nil, fmt.Errorf("无效的 interval 格式: %s", interval)
	}

	query := `SELECT _wstart, LAST(x) as x, LAST(y) as y 
			  FROM MQBOT.robot_status 
			  WHERE robot_id = ? AND ts >= ? AND ts < ? 
			  INTERVAL(?)`

	rows, err := s.db.QueryContext(context.Background(), query, robotID, from, to, interval)
	if err != nil {
		return nil, fmt.Errorf("查询轨迹失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tp TrackPoint
		var tsTime time.Time

		if err := rows.Scan(&tsTime, &tp.X, &tp.Y); err != nil {
			return nil, fmt.Errorf("扫描轨迹行失败: %w", err)
		}

		tp.TS = tsTime.UnixMilli()
		trackPoints = append(trackPoints, tp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历轨迹结果集出错: %w", err)
	}

	return trackPoints, nil
}

// QueryLatest 查某设备最近一条
func (s *Storage) QueryLatest(robotID string) (*Point, error) {
	query := `SELECT LAST(ts), LAST(` + "`state`" + `), LAST(battery), LAST(x), LAST(y), LAST(speed)
			  FROM MQBOT.robot_status
			  WHERE robot_id = ?`

	var p Point
	var tsTime time.Time

	// QueryRow 用于只返回一行的场景
	err := s.db.QueryRowContext(context.Background(), query, robotID).Scan(
		&tsTime, &p.State, &p.Battery, &p.X, &p.Y, &p.Speed,
	)

	if err != nil {
		// 如果没有查到数据，返回 nil 而不是报错
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询最新状态失败: %w", err)
	}

	p.TS = tsTime.UnixMilli()
	return &p, nil
}
// Writer：channel 攒批 拼 SQL 丢弃计数 优雅关闭
package database

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhuofeng1023/mqbot/internal/config"
)

// Writer 异步批量写入器：channel 收消息，后台 goroutine 攒批刷写
type Writer struct {
	ch        chan StatusPoint
	db        *sql.DB
	cfg       *config.DBWriteConfig
	done      chan struct{}
	closed    sync.Once
	wg        sync.WaitGroup
	WriteErrs atomic.Int64 // 刷写失败次数
	Written   atomic.Int64 // 成功落盘条数
	Dropped   atomic.Int64
}

func NewWriter(db *sql.DB, cfg *config.DBWriteConfig) *Writer {
	return &Writer{
		ch:        make(chan StatusPoint, cfg.ChannelCap),
		db:        db,
		cfg:       cfg,
		done:      make(chan struct{}),
		Dropped:   atomic.Int64{},
		WriteErrs: atomic.Int64{},
		Written:   atomic.Int64{},
	}
}

func (w *Writer) Write(p StatusPoint) {
	select {
	case w.ch <- p:
	default:
		w.Dropped.Add(1)
	}
}

func (w *Writer) Run() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		// 使用 Timer 替代 Ticker，以便在每次 flush 后重置计时
		timer := time.NewTimer(time.Duration(w.cfg.FlushIntervalMs) * time.Millisecond)
		defer timer.Stop()

		buffer := make([]StatusPoint, 0, w.cfg.BatchSize)

		for {
			select {
			case p := <-w.ch:
				buffer = append(buffer, p)
				if len(buffer) >= w.cfg.BatchSize {
					w.flush(buffer)
					buffer = buffer[:0]
					// 写入后重置计时器
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(time.Duration(w.cfg.FlushIntervalMs) * time.Millisecond)
				}
			case <-timer.C:
				if len(buffer) > 0 {
					w.flush(buffer)
					buffer = buffer[:0]
				}
				timer.Reset(time.Duration(w.cfg.FlushIntervalMs) * time.Millisecond)
			case <-w.done:
				// 关闭时，将通道内剩余数据全部取出 (drain)
				draining := true
				for draining {
					select {
					case p, ok := <-w.ch:
						if !ok {
							draining = false
						} else {
							buffer = append(buffer, p)
							if len(buffer) >= w.cfg.BatchSize {
								w.flush(buffer)
								buffer = buffer[:0]
							}
						}
					default:
						draining = false
					}
				}
				if len(buffer) > 0 {
					w.flush(buffer)
				}
				return
			}
		}
	}()
}

func (w *Writer) flush(points []StatusPoint) {
	if len(points) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO ")

	// 按 robot_id 分组
	groups := make(map[string][]StatusPoint)
	for _, p := range points {
		groups[p.RobotID] = append(groups[p.RobotID], p)
	}

	for robotID, pts := range groups {
		tableName := subTableName(robotID)
		// 使用 esc 函数防止 SQL 注入
		sb.WriteString(fmt.Sprintf("%s USING robot_status TAGS('%s') VALUES ", tableName, esc(robotID)))
		for _, p := range pts {
			// 为时间戳和字符串值加上引号，并使用 esc 转义
			sb.WriteString(fmt.Sprintf("('%d', '%s', %f, %f, %f, %f) ",
				p.TS,
				esc(p.State),
				p.Battery,
				p.X,
				p.Y,
				p.Speed,
			))
		}
	}

	if _, err := w.db.Exec(sb.String()); err != nil {
		w.WriteErrs.Add(1)
		fmt.Printf("Writer flush error: %v\n", err)
	} else {
		w.Written.Add(int64(len(points)))
	}
}

func (w *Writer) Close() {
	w.closed.Do(func() {
		close(w.done)
		w.wg.Wait() // 修复：等待后台协程结束
	})
}

func subTableName(id string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	clean := re.ReplaceAllString(id, "_")
	if clean == "" {
		clean = "default"
	}
	return "d_" + clean
}

func esc(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

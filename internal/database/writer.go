// Writer：channel 攒批 拼 SQL 丢弃计数 优雅关闭
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhuofeng1023/mqbot/internal/config"
)

// Writer 异步批量写入器：channel 收消息，后台 goroutine 攒批刷写
type Writer struct {
	ch      chan StatusPoint
	db      *sql.DB
	cfg     *config.DBWriteConfig
	done    chan struct{}
	closed  sync.Once
	Dropped atomic.Int64
}

func NewWriter(db *sql.DB, cfg *config.DBWriteConfig) *Writer {
	return &Writer{
		ch:      make(chan StatusPoint, cfg.ChannelCap),
		db:      db,
		cfg:     cfg,
		done:    make(chan struct{}),
		Dropped: atomic.Int64{},
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
	ticker := time.NewTicker(time.Duration(w.cfg.FlushIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	buffer := make([]StatusPoint, 0, w.cfg.BatchSize)

	for {
		select {
		case p := <-w.ch:
			buffer = append(buffer, p)
			if len(buffer) >= w.cfg.BatchSize {
				w.flush(buffer)
				buffer = buffer[:0]
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				w.flush(buffer)
				buffer = buffer[:0]
			}
		case <-w.done:
			if len(buffer) > 0 {
				w.flush(buffer)
			}
			return
		}
	}
}

func (w *Writer) flush(points []StatusPoint) {
	if len(points) == 0 {
		return
	}
	var sb strings.Builder
	sb.WriteString("INSERT INTO ")
	for _, p := range points {
		sb.WriteString(fmt.Sprintf("d_%s USING robot_status TAGS('%s') VALUES('%d','%s',%f,%f,%f,%f) ",
			p.RobotID, p.RobotID, p.TS, p.State, p.Battery, p.X, p.Y, p.Speed))
	}
	if _, err := w.db.Exec(sb.String()); err != nil {
		fmt.Printf("Writer flush error: %v\n", err)
	}
}

func (w *Writer) Close() {
	w.closed.Do(func() { close(w.done) })
}

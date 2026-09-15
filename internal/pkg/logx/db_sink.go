package logx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"agent-desk/internal/models"

	"gorm.io/gorm"
)

const (
	dbSinkChannelCapacity = 4096
	dbSinkBatchSize       = 200
	dbSinkFlushInterval   = 1 * time.Second
)

// dbSink 包装底层 stdout handler，并将 INFO/WARN/ERROR 级别日志异步写入数据库。
type dbSink struct {
	inner slog.Handler

	// DB 引用，通过 atomic 实现 lazy 注入。nil 时跳过 DB 写入。
	db atomic.Pointer[gorm.DB]

	// 日志 channel，worker goroutine 消费。
	ch chan models.SystemLog

	// 控制 worker 生命周期。
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
}

// dbSinkDropped 记录因 channel 满而丢弃的日志条数（进程级累计）。
var dbSinkDropped atomic.Uint64

// dbSinkLastDropReportSec 记录上次输出丢弃告警的秒级时间戳，用于限频，避免 INFO 高频时刷屏 stderr。
var dbSinkLastDropReportSec atomic.Int64

var sink *dbSink

// AttachDB 注入 DB 并启动 worker goroutine。调用幂等。
func AttachDB(db *gorm.DB) {
	if sink == nil {
		return
	}
	if sink.db.Load() != nil {
		return
	}
	sink.db.Store(db)
	go sink.worker()
}

// Close 触发 worker 最终 flush 并退出。在 server shutdown 时调用。
func Close() error {
	if sink == nil {
		return nil
	}
	sink.stopOnce.Do(func() {
		close(sink.stop)
	})
	select {
	case <-sink.done:
	case <-time.After(3 * time.Second):
	}
	return nil
}

func (h *dbSink) Handle(ctx context.Context, r slog.Record) error {
	// 先走 stdout
	if err := h.inner.Handle(ctx, r.Clone()); err != nil {
		return err
	}

	// 只持久化 INFO 及以上级别（DEBUG 不写库）
	if r.Level < slog.LevelInfo {
		return nil
	}

	// DB 未注入时跳过
	db := h.db.Load()
	if db == nil {
		return nil
	}

	// 构造 SystemLog
	logEntry := buildSystemLog(r)

	// 非阻塞写入 channel，满时丢弃（每秒最多输出一次 stderr 告警，避免高频日志刷屏）
	select {
	case h.ch <- logEntry:
	default:
		dropped := dbSinkDropped.Add(1)
		nowSec := time.Now().Unix()
		if dbSinkLastDropReportSec.Swap(nowSec) != nowSec {
			fmt.Fprintf(os.Stderr, "logx: db sink channel full, dropped %d log entries\n", dropped)
		}
	}

	return nil
}

func (h *dbSink) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *dbSink) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dbSink{
		inner: h.inner.WithAttrs(attrs),
		db:    h.db,
		ch:    h.ch,
		stop:  h.stop,
		done:  h.done,
	}
}

func (h *dbSink) WithGroup(name string) slog.Handler {
	return &dbSink{
		inner: h.inner.WithGroup(name),
		db:    h.db,
		ch:    h.ch,
		stop:  h.stop,
		done:  h.done,
	}
}

func (h *dbSink) worker() {
	defer close(h.done)

	batch := make([]models.SystemLog, 0, dbSinkBatchSize)
	ticker := time.NewTicker(dbSinkFlushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		db := h.db.Load()
		if db == nil {
			batch = batch[:0]
			return
		}
		if err := db.CreateInBatches(batch, dbSinkBatchSize).Error; err != nil {
			// 写入失败不影响主流程，输出到 stderr
			fmt.Fprintf(os.Stderr, "logx: batch insert system logs failed: %v\n", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-h.stop:
			// drain 剩余日志
			for {
				select {
				case entry := <-h.ch:
					batch = append(batch, entry)
					if len(batch) >= dbSinkBatchSize {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case entry := <-h.ch:
			batch = append(batch, entry)
			if len(batch) >= dbSinkBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func buildSystemLog(r slog.Record) models.SystemLog {
	// 提取 source（file:line）
	source := ""
	if r.PC != 0 {
		frames := runtimeCallerFrames(r.PC)
		if frames != nil {
			source = fmt.Sprintf("%s:%d", frames.File, frames.Line)
		}
	}

	// 提取 attrs
	attrs := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	attrsJSON := ""
	if len(attrs) > 0 {
		if b, err := json.Marshal(attrs); err == nil {
			attrsJSON = string(b)
		}
	}

	return models.SystemLog{
		Level:      r.Level.String(),
		Message:    r.Message,
		Source:     source,
		LoggerName: "default",
		Attrs:      attrsJSON,
		CreatedAt:  r.Time,
	}
}

func runtimeCallerFrames(pc uintptr) *runtime.Frame {
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if frame.File == "" {
		return nil
	}
	return &frame
}

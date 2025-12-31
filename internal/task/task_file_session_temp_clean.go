package task

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/haierkeys/fast-note-sync-service/global"
	"go.uber.org/zap"
)

// FileSessionTempCleanTask 临时文件清理任务
type FileSessionTempCleanTask struct {
	firstRun bool
}

// Name 任务名称
func (t *FileSessionTempCleanTask) Name() string {
	return "FileSessionTempClean"
}

// LoopInterval 执行间隔 (0 表示不进行周期性执行)
func (t *FileSessionTempCleanTask) LoopInterval() time.Duration {
	return time.Hour
}

// IsStartupRun 是否立即执行一次
func (t *FileSessionTempCleanTask) IsStartupRun() bool {
	return true
}

// Run 执行清理任务
func (t *FileSessionTempCleanTask) Run(ctx context.Context) error {
	tempDir := global.Config.App.TempPath
	if tempDir == "" {
		tempDir = "storage/temp"
	}

	var err error

	// 检查目录是否存在
	if _, err = os.Stat(tempDir); os.IsNotExist(err) {
		global.Logger.Error("task log",
			zap.String("task", t.Name()),
			zap.String("type", "run"),
			zap.String("path", tempDir),
			zap.String("reason", "temp directory does not exist"),
			zap.String("msg", "failed"))
		return err
	}

	// 首次运行时，删除整个目录并重新创建
	if t.firstRun {
		t.firstRun = false

		// 删除整个目录
		if err = os.RemoveAll(tempDir); err != nil {
			global.Logger.Error("task log",
				zap.String("task", t.Name()),
				zap.String("type", "startupRun"),
				zap.String("path", tempDir),
				zap.String("msg", "failed"),
				zap.Error(err))
			return err
		}

		// 重新创建目录
		if err = os.MkdirAll(tempDir, 0754); err != nil {
			global.Logger.Error("task log",
				zap.String("task", t.Name()),
				zap.String("type", "startupRun"),
				zap.String("path", tempDir),
				zap.String("msg", "failed"),
				zap.Error(err))
			return err
		}

		global.Logger.Info("task log",
			zap.String("task", t.Name()),
			zap.String("type", "startupRun"),
			zap.String("msg", "success"))

		return nil
	}

	// 周期性运行：删除超过 2 小时未修改的文件
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	deletedCount := 0
	errorCount := 0

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			global.Logger.Warn("task log",
				zap.String("task", t.Name()),
				zap.String("type", "loopRun"),
				zap.String("path", path),
				zap.String("msg", "error accessing path"),
				zap.Error(err))
			return nil // 继续处理其他文件
		}

		// 跳过目录本身
		if info.IsDir() {
			return nil
		}

		// 检查文件修改时间
		if info.ModTime().Before(twoHoursAgo) {
			if removeErr := os.Remove(path); removeErr != nil {
				global.Logger.Warn("task log",
					zap.String("task", t.Name()),
					zap.String("type", "loopRun"),
					zap.String("path", path),
					zap.String("msg", "failed to remove old file"),
					zap.Error(removeErr))
				errorCount++
			} else {
				deletedCount++
			}
		}

		return nil
	})

	if err != nil {
		global.Logger.Error("task log",
			zap.String("task", t.Name()),
			zap.String("type", "loopRun"),
			zap.String("path", tempDir),
			zap.String("msg", "failed"),
			zap.Error(err))
		return err
	}

	global.Logger.Info("task log",
		zap.String("task", t.Name()),
		zap.String("type", "loopRun"),
		zap.Int("deletedCount", deletedCount),
		zap.Int("errorCount", errorCount),
		zap.String("msg", "success"))

	return nil
}

// NewFileSessionTempCleanTask 创建临时文件清理任务
func NewFileSessionTempCleanTask() (Task, error) {
	return &FileSessionTempCleanTask{
		firstRun: true,
	}, nil
}

func init() {
	Register(NewFileSessionTempCleanTask)
}

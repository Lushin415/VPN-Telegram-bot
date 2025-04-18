package admin

import (
	"context"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// BackupDatabase создает дамп БД Postgres в указанный файл
func BackupDatabase(filename string, dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pg_dump", dsn, "-Fc", "-f", filename)
	return cmd.Run()
}

// RestoreDatabase восстанавливает БД из дампа
func RestoreDatabase(filename string, dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pg_restore", "-d", dsn, filename)
	return cmd.Run()
}

// CleanOldBackups удаляет все дампы старше monthDuration в директории backups
func CleanOldBackups(dir string, monthDuration time.Duration) error {
	files, err := filepath.Glob(filepath.Join(dir, "*backup_*.dump"))
	if err != nil {
		return err
	}
	files2, err := filepath.Glob(filepath.Join(dir, "*autobackup_*.dump"))
	if err != nil {
		return err
	}
	files = append(files, files2...)
	cutoff := time.Now().Add(-monthDuration)
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(f)
		}
	}
	return nil
}

// AutoBackupDatabase запускает бэкап и чистку, уведомляет админа
func AutoBackupDatabase(bot *tgbotapi.BotAPI, adminID int64, dsn string) {
	backupDir := "backups"
	os.MkdirAll(backupDir, 0o755)
	filename := filepath.Join(backupDir, "autobackup_"+time.Now().Format("20060102_150405")+".dump")
	err := BackupDatabase(filename, dsn)
	if err != nil {
		log.Println("[AUTO BACKUP] Ошибка резервного копирования: " + err.Error())
		return
	}
	CleanOldBackups(backupDir, 31*24*time.Hour)
	log.Println("[AUTO BACKUP] Резервная копия БД успешно создана: " + filename)
}

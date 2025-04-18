package services

import (
	"VPN-Telegram-bot/internal/db"
	"VPN-Telegram-bot/internal/logger"
	"net"
	"time"
)

type ServerStatus struct {
	Name        string
	IP          string
	Status      string
	Load        int
	LastChecked time.Time
}

var lastStatuses []ServerStatus

func GetServerStatuses() []ServerStatus {
	return lastStatuses
}

func UpdateAllServerStatuses() {
	var statuses []ServerStatus
	var servers []db.Server
	db.DB.Where("is_active = true").Find(&servers)
	for _, srv := range servers {
		status := ServerStatus{
			Name: srv.Name,
			IP:   srv.IP,
			Load: 0, // Можно доработать: получать нагрузку с сервера
		}
		conn, err := net.DialTimeout("tcp", srv.IP+":443", 2*time.Second)
		if err != nil {
			status.Status = "❌ offline"
			logger.NotifyAdmin("Сервер " + srv.Name + " (" + srv.IP + ") недоступен!")
		} else {
			status.Status = "✅ online"
			conn.Close()
		}
		status.LastChecked = time.Now()
		statuses = append(statuses, status)
	}
	lastStatuses = statuses
}

func ReloadServers() error {
	UpdateAllServerStatuses()
	return nil
}

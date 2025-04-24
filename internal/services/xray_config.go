package services

import (
	"VPN-Telegram-b
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

type Config struct {
	Log       interface{} `json:"log,omitempty"`
	Inbounds  []Inbound   `json:"inbounds"`
	Outbounds interface{} `json:"outbounds,omitempty"`
	Routing   interface{} `json:"routing,omitempty"`
}

type Inbound struct {
	Listen         string          `json:"listen,omitempty"`
	Port           int             `json:"port,omitempty"`
	Protocol       string          `json:"protocol,omitempty"`
	Tag            string          `json:"tag,omitempty"`
	Settings       InboundSettings `json:"settings"`
	StreamSettings interface{}     `json:"streamSettings,omitempty"`
	Sniffing       interface{}     `json:"sniffing,omitempty"`
}

type InboundSettings struct {
	Clients    []Client `json:"clients"`
	Decryption string   `json:"decryption,omitempty"`
}

type Client struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Level int    `json:"level,omitempty"`
	Flow  string `json:"flow,omitempty"`
}

// AddClientToRemoteXrayConfig скачивает config.json с сервера, добавляет клиента и загружает обратно
func AddClientToRemoteXrayConfig(
	sshUser, sshHost, sshPort, sshKeyPath, clientID, clientEmail, clientFlow string,
) error {
	remotePath := "/usr/local/etc/xray/config.json"
	localPath := "config.json.tmp"

	log.Printf("[XrayConfig] Step 1: Downloading config.json from %s@%s:%s", sshUser, sshHost, remotePath)
	// 1. Скачать config.json с сервера
	scpDownload := exec.Command(
		"scp",
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKeyPath,
		"-P", sshPort,
		fmt.Sprintf("%s@%s:%s", sshUser, sshHost, remotePath),
		localPath,
	)
	scpDownload.Stdout = os.Stdout
	scpDownload.Stderr = os.Stderr
	if err := scpDownload.Run(); err != nil {
		log.Printf("[XrayConfig] scp download failed: %v", err)
		return fmt.Errorf("scp download failed: %w", err)
	}

	log.Printf("[XrayConfig] Step 2: Reading local config file %s", localPath)
	// 2. Прочитать и распарсить config.json
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		log.Printf("[XrayConfig] read local config failed: %v", err)
		return fmt.Errorf("read local config failed: %w", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("[XrayConfig] unmarshal config failed: %v", err)
		return fmt.Errorf("unmarshal config failed: %w", err)
	}

	log.Printf("[XrayConfig] Step 3: Adding client %s to inbounds", clientID)
	// 3. Добавить клиента в первый inbound
	newClient := Client{
		ID:    clientID,
		Email: clientEmail,
		Flow:  clientFlow,
	}
	if len(config.Inbounds) > 0 {
		config.Inbounds[0].Settings.Clients = append(config.Inbounds[0].Settings.Clients, newClient)
	} else {
		log.Printf("[XrayConfig] no inbounds in config")
		return fmt.Errorf("no inbounds in config")
	}

	log.Printf("[XrayConfig] Step 4: Writing updated config to %s", localPath)
	// 4. Сохранить изменённый config.json
	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("[XrayConfig] marshal config failed: %v", err)
		return fmt.Errorf("marshal config failed: %w", err)
	}
	if err := ioutil.WriteFile(localPath, out, 0644); err != nil {
		log.Printf("[XrayConfig] write local config failed: %v", err)
		return fmt.Errorf("write local config failed: %w", err)
	}

	log.Printf("[XrayConfig] Step 5: Uploading updated config.json to server %s@%s:%s", sshUser, sshHost, remotePath)
	// 5. Загрузить изменённый config.json обратно на сервер
	scpUpload := exec.Command(
		"scp",
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKeyPath,
		"-P", sshPort,
		localPath,
		fmt.Sprintf("%s@%s:%s", sshUser, sshHost, remotePath),
	)
	scpUpload.Stdout = os.Stdout
	scpUpload.Stderr = os.Stderr
	if err := scpUpload.Run(); err != nil {
		log.Printf("[XrayConfig] scp upload failed: %v", err)
		return fmt.Errorf("scp upload failed: %w", err)
	}

	log.Printf("[XrayConfig] Step 6: Restarting xray service via ssh on %s@%s", sshUser, sshHost)
	// 6. Перезапустить сервис Xray на сервере
	sshRestart := exec.Command(
		"ssh",
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKeyPath,
		"-p", sshPort,
		fmt.Sprintf("%s@%s", sshUser, sshHost),
		"systemctl restart xray",
	)
	sshRestart.Stdout = os.Stdout
	sshRestart.Stderr = os.Stderr
	if err := sshRestart.Run(); err != nil {
		log.Printf("[XrayConfig] restart xray failed: %v", err)
		return fmt.Errorf("restart xray failed: %w", err)
	}

	log.Printf("[XrayConfig] Step 7: Removing local tmp file %s", localPath)
	// 7. Удалить временный файл
	_ = os.Remove(localPath)

	log.Printf("[XrayConfig] Done: client %s added and xray restarted", clientID)
	return nil
}

// GetAllXrayUUIDsFromRemote скачивает config.json с сервера и возвращает список всех uuid (ID) клиентов
func GetAllXrayUUIDsFromRemote(sshUser, sshHost, sshPort, sshKeyPath string) ([]string, error) {
	remotePath := "/usr/local/etc/xray/config.json"
	localPath := "config.json.tmp"

	scpDownload := exec.Command(
		"scp",
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKeyPath,
		"-P", sshPort,
		fmt.Sprintf("%s@%s:%s", sshUser, sshHost, remotePath),
		localPath,
	)
	scpDownload.Stdout = os.Stdout
	scpDownload.Stderr = os.Stderr
	if err := scpDownload.Run(); err != nil {
		return nil, fmt.Errorf("scp download failed: %w", err)
	}
	data, err := ioutil.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read local config failed: %w", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	var uuids []string
	for _, inbound := range config.Inbounds {
		for _, client := range inbound.Settings.Clients {
			uuids = append(uuids, client.ID)
		}
	}
	_ = os.Remove(localPath)
	return uuids, nil
}

// CleanDBKeysNotInXray удаляет из БД все ключи, которых нет в списке актуальных uuid
func CleanDBKeysNotInXray(actualUUIDs []string) error {
	var allKeys []db.VLESSKey
	db.DB.Find(&allKeys)
	uuidSet := make(map[string]struct{})
	for _, uuid := range actualUUIDs {
		uuidSet[uuid] = struct{}{}
	}
	for _, key := range allKeys {
		if _, ok := uuidSet[key.Key]; !ok {
			// Ключа нет в актуальном списке — удаляем
			db.DB.Delete(&key)
		}
	}
	return nil
}

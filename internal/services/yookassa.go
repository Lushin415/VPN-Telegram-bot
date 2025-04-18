package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type PaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

func CreateYooKassaPayment(userID uint, amount int, shopID, secretKey string) (paymentID, paymentURL string, err error) {
	// Минимальный пример, требует доработки и секретов
	url := "https://api.yookassa.ru/v3/payments"
	body := map[string]interface{}{
		"amount":       map[string]interface{}{"value": fmt.Sprintf("%d.00", amount), "currency": "RUB"},
		"confirmation": map[string]string{"type": "redirect"},
		"capture":      true,
		"description":  fmt.Sprintf("VPN for user %d", userID),
	}
	jsonBody, _ := json.Marshal(body)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(shopID, secretKey)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", "", errors.New("YooKassa error")
	}
	var pr PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", "", err
	}
	return pr.ID, pr.Confirmation.ConfirmationURL, nil
}

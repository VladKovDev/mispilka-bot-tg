package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"time"
)

type PaymentRequest struct {
	OrderID    string  `json:"order_id"`
	Amount     float64 `json:"amount"`
	Currency   string  `json:"currency"`
	Customer   string  `json:"customer,omitempty"`
	WebhookURL string  `json:"webhook_url,omitempty"`
}

type PaymentLink struct {
	URL      string `json:"url"`
	OrderID  string `json:"order_id"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func GeneratePaymentLink(orderID string, amount float64, currency string, apiURL string) (*PaymentLink, error) {
	if apiURL == "" {
		return nil, fmt.Errorf("PRODAMUS_API_URL is not configured")
	}

	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %v", err)
	}

	values := url.Values{}
	values.Set("order_id", orderID)
	values.Set("amount", fmt.Sprintf("%.2f", amount))
	values.Set("currency", currency)

	baseURL.RawQuery = values.Encode()

	return &PaymentLink{
		URL:      baseURL.String(),
		OrderID:  orderID,
		Amount:   fmt.Sprintf("%.2f", amount),
		Currency: currency,
	}, nil
}

func GeneratePaymentLinkWithSignature(orderID string, amount float64, currency string, webhookURL string, apiURL string, secretKey string) (*PaymentLink, error) {
	if apiURL == "" {
		return nil, fmt.Errorf("PRODAMUS_API_URL is not configured")
	}

	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API URL: %v", err)
	}

	values := url.Values{}
	values.Set("order_id", orderID)
	values.Set("amount", fmt.Sprintf("%.2f", amount))
	values.Set("currency", currency)

	if webhookURL != "" {
		values.Set("webhook_url", webhookURL)
	}

	queryString := values.Encode()

	if secretKey != "" {
		signature := calculateSignature(queryString, secretKey)
		values.Set("signature", signature)
	}

	baseURL.RawQuery = values.Encode()

	return &PaymentLink{
		URL:      baseURL.String(),
		OrderID:  orderID,
		Amount:   fmt.Sprintf("%.2f", amount),
		Currency: currency,
	}, nil
}

func calculateSignature(data string, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func Now() time.Time {
	return time.Now()
}

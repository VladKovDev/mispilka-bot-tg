package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"mispilkabot/config"
)

// ProdamusClient handles payment link generation via Prodamus API
type ProdamusClient struct {
	apiURL            string
	sysCode           string
	productUniqueName string
	httpClient        *http.Client
}

// NewProdamusClient creates a new Prodamus client with config
func NewProdamusClient(cfg *config.Config) *ProdamusClient {
	return &ProdamusClient{
		apiURL:     cfg.ProdamusAPIURL,
		httpClient: &http.Client{},
	}
}

// GeneratePaymentLink creates a payment link via Prodamus API via GET request
// Documentation: https://help.prodamus.ru/payform/integracii/rest-api/
func (p *ProdamusClient) GeneratePaymentLink(productName string, price string, paidContent string) (string, error) {
	// Build query parameters for GET request
	params := url.Values{}
	params.Set("do", "link")
	params.Set("paid_content", paidContent)
	params.Set("products[0][name]", productName)
	params.Set("products[0][price]", price)
	params.Set("products[0][quantity]", "1")

	// Construct full URL with query string
	fullURL := fmt.Sprintf("%s?%s", p.apiURL, params.Encode())

	log.Printf("Sending Prodamus GET request to %s", fullURL)

	// Make GET request
	resp, err := p.httpClient.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response as plain text
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("Prodamus response: %s", string(body))

	// Parse as plain text URL (expected format)
	link := strings.TrimSpace(string(body))
	if isValidURL(link) {
		return link, nil
	}

	return "", fmt.Errorf("invalid response format - expected plain text URL, got: %s", string(body))
}

// isValidURL checks if a string looks like a valid URL
func isValidURL(s string) bool {
	_, err := url.ParseRequestURI(s)
	return err == nil && len(s) > 4 && (s[:4] == "http")
}

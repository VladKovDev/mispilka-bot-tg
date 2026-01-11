package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"mispilkabot/config"
	"mispilkabot/internal/services/hmac"
)

type SignatureInput struct {
	UserID   string
	Products []SignatureProduct
}

type SignatureProduct struct {
	Name     string
	Price    string
	Quantity string
}

func BuildSignaturePayload(input SignatureInput) map[string]string {
	payload := map[string]string{
		"_param_user_id": input.UserID,
	}

	if len(input.Products) > 0 {
		product := input.Products[0]

		payload["product_name"] = product.Name
		payload["product_price"] = product.Price
		payload["product_quantity"] = product.Quantity
	}

	return payload
}

type ProdamusClient struct {
	apiURL     string
	secretKey  string
	httpClient *http.Client
}

// NewProdamusClient creates a new Prodamus client with config
func NewProdamusClient(cfg *config.Config) *ProdamusClient {
	return &ProdamusClient{
		apiURL:     cfg.ProdamusAPIURL,
		secretKey:  cfg.ProdamusSecret,
		httpClient: &http.Client{},
	}
}

// GeneratePaymentLink creates a payment link via Prodamus API with signature
// Documentation: https://help.prodamus.ru/payform/integracii/rest-api/
func (p *ProdamusClient) GeneratePaymentLink(userID string, productName string, price string, paidContent string) (string, error) {
	signInput := SignatureInput{
		UserID: userID,
		Products: []SignatureProduct{
			{
				Name:     productName,
				Price:    price,
				Quantity: "1",
			},
		},
	}

	// Build signature payload
	signData := BuildSignaturePayload(signInput)

	// Create signature using Prodamus algorithm
	signature, err := hmac.CreateSignature(signData, p.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	log.Printf("Generated signature: %s", signature)

	// Build query parameters for GET request
	params := url.Values{}
	params.Set("do", "link")
	params.Set("paid_content", paidContent)
	params.Set("payments_limit", "1")
	params.Set("_param_user_id", userID)
	params.Set("products[0][name]", productName)
	params.Set("products[0][price]", price)
	params.Set("products[0][quantity]", "1")
	params.Set("sign", signature)

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

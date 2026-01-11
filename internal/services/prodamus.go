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

// SignatureKeys defines the fields that are included in Prodamus signature calculation.
// These fields must match exactly between payment link generation and webhook verification.
var SignatureKeys = []string{
	"do",
	"order_id",
	"paid_content",
	"payments_limit",
	"products",
}

// ProductSignatureKeys defines the fields inside each product that are included in signature calculation.
var ProductSignatureKeys = []string{
	"name",
	"price",
	"quantity",
}

// filterProductMap extracts only the product signature keys from a product map.
// Returns nil if the input is not a valid product map.
func filterProductMap(product interface{}) map[string]interface{} {
	productMap, ok := product.(map[string]interface{})
	if !ok {
		return nil
	}

	filtered := make(map[string]interface{}, len(ProductSignatureKeys))
	for _, key := range ProductSignatureKeys {
		if value, exists := productMap[key]; exists {
			filtered[key] = value
		}
	}
	return filtered
}

// filterProductsArray filters an array of products to only include product signature keys.
// Returns nil if the input is not a valid products array.
func filterProductsArray(products interface{}) []interface{} {
	productsSlice, ok := products.([]interface{})
	if !ok {
		return nil
	}

	filtered := make([]interface{}, 0, len(productsSlice))
	for _, product := range productsSlice {
		if filteredProduct := filterProductMap(product); filteredProduct != nil {
			filtered = append(filtered, filteredProduct)
		}
	}
	return filtered
}

// BuildSignaturePayload filters a full payload to include only the keys required for
// signature calculation. For the "products" array, each product is filtered to only
// include keys from ProductSignatureKeys.
func BuildSignaturePayload(fullPayload map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(SignatureKeys))

	for _, key := range SignatureKeys {
		value, exists := fullPayload[key]
		if !exists {
			continue
		}

		// Special handling for products array - filter each product
		if key == "products" {
			if filteredProducts := filterProductsArray(value); filteredProducts != nil {
				result[key] = filteredProducts
			}
		} else {
			result[key] = value
		}
	}

	return result
}

// ProdamusClient handles payment link generation via Prodamus API
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
	// Build data map for signature creation according to Prodamus algorithm
	signData := map[string]interface{}{
		"do":             "link",
		"paid_content":   paidContent,
		"order_id":       userID,
		"payments_limit": "1",
		"products": []map[string]interface{}{
			{
				"name":     productName,
				"price":    price,
				"quantity": "1",
			},
		},
	}

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
	params.Set("order_id", userID)
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

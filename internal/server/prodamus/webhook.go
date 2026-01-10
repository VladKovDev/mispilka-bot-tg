package prodamus

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mispilkabot/internal/services"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Handler handles Prodamus webhook requests
type Handler struct {
	secretKey            string
	generateInviteLinkFn func(chatID, groupID string) (string, error)
	sendInviteMessage    func(chatID, inviteLink string)
	mu                   sync.Mutex // Protect user mutations
	privateGroupID       string     // Private group ID for invite link generation
}

func NewHandler() *Handler {
	return &Handler{}
}

// SetSecretKey sets the Prodamus secret key for webhook signature verification
func (h *Handler) SetSecretKey(secretKey string) {
	h.secretKey = secretKey
}

// SetPrivateGroupID sets the private group ID for invite link generation
func (h *Handler) SetPrivateGroupID(groupID string) {
	h.privateGroupID = groupID
}

// SetGenerateInviteLinkCallback sets the callback for generating invite links
func (h *Handler) SetGenerateInviteLinkCallback(callback func(chatID, groupID string) (string, error)) {
	h.generateInviteLinkFn = callback
}

// SetInviteMessageCallback sets the callback for sending invite messages
func (h *Handler) SetInviteMessageCallback(callback func(chatID, inviteLink string)) {
	h.sendInviteMessage = callback
}

type WebhookPayload struct {
	OrderID   string `json:"order_id"`
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	Status    string `json:"status"`
	PaymentID string `json:"payment_id"`
	Signature string `json:"signature"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("=== Prodamus Webhook Received ===")

	// Validate HTTP method
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read and parse request body
	payload, bodyBytes, err := h.readAndParseBody(r)
	if err != nil {
		log.Printf("Failed to read and parse body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Log the request details
	h.logRequest(r, bodyBytes)
	log.Printf("Webhook payload: %+v", payload)

	// Verify signature
	if !h.verifySignature(bodyBytes) {
		log.Printf("Invalid signature for order_id: %s", payload.OrderID)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Process successful payments
	if h.isSuccessStatus(payload.Status) {
		if err := h.processPayment(payload.OrderID); err != nil {
			log.Printf("Error processing payment: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("Payment processed successfully for chat_id: %s", payload.OrderID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// readAndParseBody reads the request body and unmarshals it into a WebhookPayload
func (h *Handler) readAndParseBody(r *http.Request) (*WebhookPayload, []byte, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	var payload WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &payload, bodyBytes, nil
}

// isSuccessStatus checks if the status indicates a successful payment
func (h *Handler) isSuccessStatus(status string) bool {
	return status == "success" || status == "paid"
}

func (h *Handler) verifySignature(body []byte) bool {
	if h.secretKey == "" {
		log.Println("Warning: PRODAMUS_SECRET_KEY is not set, skipping signature verification")
		return true
	}

	mac := hmac.New(sha256.New, []byte(h.secretKey))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}

	receivedSignature := payload.Signature
	if receivedSignature == "" {
		log.Println("Warning: No signature in payload")
		return false
	}

	isValid := hmac.Equal([]byte(expectedSignature), []byte(receivedSignature))
	log.Printf("Signature verification: expected=%s, received=%s, valid=%v", expectedSignature, receivedSignature, isValid)
	return isValid
}

func (h *Handler) processPayment(chatID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	userData, err := services.GetUser(chatID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	now := time.Now()
	userData.PaymentDate = &now
	userData.IsMessaging = false

	// Generate and send invite link for the paid user
	inviteLink, err := h.generateInviteLink(chatID)
	if err != nil {
		log.Printf("Failed to generate invite link for %s: %v", chatID, err)
		// Don't fail the webhook, just log and continue
	} else {
		// Update user with invite link info
		userData.InviteLink = inviteLink

		// Send invite message via callback asynchronously
		if h.sendInviteMessage != nil {
			go h.sendInviteMessage(chatID, inviteLink)
		}
		log.Printf("invite link generated and queued for sending to %s", chatID)
	}

	if err := services.ChangeUser(chatID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// generateInviteLink creates an invite link for the user
func (h *Handler) generateInviteLink(chatID string) (string, error) {
	if h.privateGroupID == "" {
		return "", fmt.Errorf("PRIVATE_GROUP_ID not set")
	}

	if h.generateInviteLinkFn == nil {
		return "", fmt.Errorf("generateInviteLink callback not set")
	}

	inviteLink, err := h.generateInviteLinkFn(chatID, h.privateGroupID)
	if err != nil {
		return "", fmt.Errorf("failed to generate invite link: %w", err)
	}

	return inviteLink, nil
}

func (h *Handler) logRequest(r *http.Request, bodyBytes []byte) {
	// Log basic request info
	h.logRequestBasic(r)

	// Log headers
	h.logHeaders(r.Header)

	// Log query parameters
	h.logQueryParams(r.URL.Query())

	// Log body
	h.logBody(r.Header.Get("Content-Type"), bodyBytes)

	log.Println("=== End of Webhook Request ===")
}

// logRequestBasic logs basic request information
func (h *Handler) logRequestBasic(r *http.Request) {
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Host: %s", r.Host)
}

// logHeaders logs all request headers
func (h *Handler) logHeaders(header http.Header) {
	log.Println("--- Headers ---")
	for key, values := range header {
		for _, value := range values {
			log.Printf("  %s: %s", key, value)
		}
	}
}

// logQueryParams logs query parameters
func (h *Handler) logQueryParams(query url.Values) {
	log.Println("--- Query Parameters ---")
	for key, values := range query {
		for _, value := range values {
			log.Printf("  %s: %s", key, value)
		}
	}
}

// logBody logs the request body with appropriate formatting
func (h *Handler) logBody(contentType string, bodyBytes []byte) {
	log.Printf("--- Body (raw) ---")
	log.Printf("%s", string(bodyBytes))

	log.Println("--- Parsed Body ---")

	if contentType == "application/json" || contentType == "application/json; charset=utf-8" {
		var jsonData interface{}
		if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
			log.Printf("Error parsing JSON: %v", err)
		} else {
			h.logJSONData(jsonData, "")
		}
	} else {
		log.Printf("Non-JSON content type: %s", contentType)
		log.Printf("Body preview: %s", string(bodyBytes[:min(len(bodyBytes), 200)]))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *Handler) logJSONData(data interface{}, indent string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch val := value.(type) {
			case map[string]interface{}:
				log.Printf("%s%s:", indent, key)
				h.logJSONData(val, indent+"  ")
			case []interface{}:
				log.Printf("%s%s (array, len=%d):", indent, key, len(val))
				for i, item := range val {
					log.Printf("%s  [%d]:", indent, i)
					h.logJSONData(item, indent+"    ")
				}
			default:
				log.Printf("%s%s: %v (%T)", indent, key, val, val)
			}
		}
	case []interface{}:
		for i, item := range v {
			log.Printf("%s[%d]:", indent, i)
			h.logJSONData(item, indent+"  ")
		}
	default:
		log.Printf("%s%v (%T)", indent, v, v)
	}
}

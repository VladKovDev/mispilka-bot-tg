package prodamus

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mispilkabot/internal/logger"
	"mispilkabot/internal/models"
	"mispilkabot/internal/services"
	"mispilkabot/internal/services/hmac"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Handler handles Prodamus webhook requests
type Handler struct {
	secretKey            string
	generateInviteLinkFn func(userID, groupID string) (string, error)
	sendInviteMessage    func(userID, inviteLink string)
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
func (h *Handler) SetGenerateInviteLinkCallback(callback func(userID, groupID string) (string, error)) {
	h.generateInviteLinkFn = callback
}

// SetInviteMessageCallback sets the callback for sending invite messages
func (h *Handler) SetInviteMessageCallback(callback func(userID, inviteLink string)) {
	h.sendInviteMessage = callback
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("=== Prodamus Webhook Received ===")

	// Read raw body for logging before parsing
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read raw body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	r.Body.Close()
	log.Printf("Raw Body (%d bytes):\n%s", len(bodyBytes), string(bodyBytes))

	// Validate HTTP method
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read and parse request body (pass pre-read bodyBytes to avoid double-read)
	payload, payloadMap, err := h.parseFormBody(bodyBytes)
	if err != nil {
		log.Printf("Failed to parse form body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Log the request details
	logger.LogRequest(r, bodyBytes)
	log.Printf("Webhook payload: %+v", payload)
	log.Printf("Key fields - order_id: %s, customer_extra: %s, sum: %s, payment_status: %s, payment_type: %s",
		payload.OrderID, payload.CustomerExtra, payload.Sum, payload.PaymentStatus, payload.PaymentType)
	log.Printf("Customer info - phone: %s, email: %s", payload.CustomerPhone, payload.CustomerEmail)

	// Log payment status
	if h.isSuccessStatus(payload.PaymentStatus) {
		log.Printf("Payment status: SUCCESS (order_id: %s)", payload.OrderID)
	} else if h.isFailedStatus(payload.PaymentStatus) {
		log.Printf("Payment status: FAILED (order_id: %s, status: %s, description: %s)",
			payload.OrderID, payload.PaymentStatus, payload.PaymentStatusDescription)
	} else {
		log.Printf("Payment status: UNEXPECTED %s (order_id: %s, status: %s, description: %s)",
			payload.PaymentStatus, payload.OrderID, payload.PaymentStatus, payload.PaymentStatusDescription)
	}

	// Verify signature (Sign is in headers according to Prodamus docs)
	if !h.verifySignature(r, payloadMap) {
		log.Printf("Invalid signature for order_id: %s", payload.OrderID)
		// Return non-200 code to indicate failure and stop processing
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid signature"))
		return
	}

	// Process successful payments
	if h.isSuccessStatus(payload.PaymentStatus) {
		// Determine user_id: customer_extra has priority over order_id
		userID := payload.CustomerExtra

		if userID == "" {
			log.Printf("Error: No user_id found in payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := h.processPayment(userID, payload); err != nil {
			log.Printf("Error processing payment: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("Payment processed successfully for user_id: %s", userID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// parseFormBody parses the form data from pre-read body bytes
func (h *Handler) parseFormBody(bodyBytes []byte) (*models.WebhookPayload, map[string]interface{}, error) {
	var payload models.WebhookPayload

	// Parse URL-encoded form data directly from bytes
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	// Create map for signature verification
	payloadMap := make(map[string]interface{})
	for key, vals := range values {
		if len(vals) > 0 {
			// For signature verification, use the first value
			// Это можно допустить, так как в массиве приходят только товары
			// И мы создаем ссылку на оплату ОДНОГО товара, TODO: в будущем можно расширить
			payloadMap[key] = vals[0]
		}
	}

	// Map form-data to WebhookPayload (Prodamus field names)
	payload.OrderID = values.Get("order_id")
	payload.OrderNum = values.Get("order_num")
	payload.Domain = values.Get("domain")
	payload.Sum = values.Get("sum")
	payload.CustomerPhone = values.Get("customer_phone")
	payload.CustomerEmail = values.Get("customer_email")
	payload.CustomerExtra = values.Get("customer_extra")
	payload.PaymentType = values.Get("payment_type")
	payload.PaymentInit = values.Get("payment_init")
	payload.Commission = values.Get("commission")
	payload.CommissionSum = values.Get("commission_sum")
	payload.Attempt = values.Get("attempt")
	payload.Date = values.Get("date")
	payload.PaymentStatus = values.Get("payment_status")
	payload.PaymentStatusDescription = values.Get("payment_status_description")

	// Parse products (may come as JSON string)
	if productsStr := values.Get("products"); productsStr != "" {
		if err := json.Unmarshal([]byte(productsStr), &payload.Products); err != nil {
			log.Printf("Warning: Failed to parse products as JSON: %v", err)
		} else {
			// Also add products to payloadMap for signature verification
			var productsArr []interface{}
			if err := json.Unmarshal([]byte(productsStr), &productsArr); err == nil {
				payloadMap["products"] = productsArr
			}
		}
	}

	return &payload, payloadMap, nil
}

// isSuccessStatus checks if the status indicates a successful payment
func (h *Handler) isSuccessStatus(status string) bool {
	return status == models.PaymentStatusSuccess
}

// isFailedStatus checks if status indicates a failed/canceled payment
func (h *Handler) isFailedStatus(status string) bool {
	return status == models.PaymentStatusOrderCanceled || status == models.PaymentStatusOrderDenied
}

// Проверить подпись согласно Prodamus Docs: https://help.prodamus.ru/payform/integracii/rest-api/instrukcii-dlya-samostoyatelnaya-integracii-servisov#kak-prinyat-uvedomlenie-ob-uspeshnoi-oplate
func (h *Handler) verifySignature(r *http.Request, payload map[string]interface{}) bool {
	if h.secretKey == "" {
		log.Println("Warning: PRODAMUS_SECRET_KEY is not set, skipping signature verification")
		return true
	}

	// Get signature from headers (Sign header according to Prodamus docs)
	receivedSignature := r.Header.Get("Sign")
	if receivedSignature == "" {
		log.Println("Warning: No Sign header found")
		return false
	}

	signaturePayload := services.BuildSignaturePayload(payload)

	// Verify signature using Prodamus algorithm
	isValid, err := hmac.VerifySignature(signaturePayload, h.secretKey, receivedSignature)
	if err != nil {
		log.Printf("Signature verification error: %v", err)
		return false
	}

	log.Printf("Signature verification: valid=%v, received=%s", isValid, receivedSignature)

	if !isValid {
		log.Println("Warning: Signature verification failed - webhook may be from unauthorized source")
	}

	return isValid
}

func (h *Handler) processPayment(userID string, payload *models.WebhookPayload) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	userData, err := services.GetUser(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	now := time.Now()
	userData.PaymentDate = &now
	userData.IsMessaging = false

	// Store full webhook payload as payment_info (direct assignment)
	userData.PaymentInfo = payload

	// Generate and send invite link for the paid user
	inviteLink, err := h.generateInviteLink(userID)
	if err != nil {
		log.Printf("Failed to generate invite link for %s: %v", userID, err)
		// Don't fail the webhook, just log and continue
	} else {
		// Update user with invite link info
		userData.InviteLink = inviteLink

		// Send invite message via callback asynchronously
		if h.sendInviteMessage != nil {
			go h.sendInviteMessage(userID, inviteLink)
		}
		log.Printf("invite link generated and queued for sending to %s", userID)
	}

	if err := services.ChangeUser(userID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// generateInviteLink creates an invite link for the user
func (h *Handler) generateInviteLink(userID string) (string, error) {
	if h.privateGroupID == "" {
		return "", fmt.Errorf("PRIVATE_GROUP_ID not set")
	}

	if h.generateInviteLinkFn == nil {
		return "", fmt.Errorf("generateInviteLink callback not set")
	}

	inviteLink, err := h.generateInviteLinkFn(userID, h.privateGroupID)
	if err != nil {
		return "", fmt.Errorf("failed to generate invite link: %w", err)
	}

	return inviteLink, nil
}

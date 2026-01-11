package prodamus

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"mispilkabot/internal/logger"
	"mispilkabot/internal/models"
	"mispilkabot/internal/services"
	"mispilkabot/internal/services/hmac"

	"github.com/go-playground/form/v4"
)

// Handler handles Prodamus webhook requests
type Handler struct {
	secretKey            string
	privateGroupID       string
	generateInviteLinkFn func(userID, groupID string) (string, error)
	sendInviteMessage    func(userID, inviteLink string)
	mu                   sync.Mutex // Protect user mutations during payment processing
}

// NewHandler creates a new webhook handler
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

// ServeHTTP handles incoming webhook requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("=== Prodamus Webhook Received ===")

	// Read raw body for logging before parsing
	bodyBytes, err := h.readRequestBody(r)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate HTTP method
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse form-encoded request body
	payload, rawFormValues, err := h.parseFormBody(bodyBytes)
	if err != nil {
		log.Printf("Failed to parse form body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Log the request details for debugging
	h.logWebhookRequest(r, bodyBytes, payload)

	// Verify signature from headers using ALL raw form values
	if !h.verifySignature(r, *payload, rawFormValues) {
		log.Printf("Invalid signature for order_id: %s", payload.OrderID)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid signature"))
		return
	}

	// Process only successful payments
	if h.isSuccessStatus(payload.PaymentStatus) {
		userID := payload.ParamUserID
		if userID == "" {
			log.Printf("Error: No customer_extra (user_id) found in payload")
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

// readRequestBody reads and returns the request body bytes
func (h *Handler) readRequestBody(r *http.Request) ([]byte, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	defer r.Body.Close()
	log.Printf("Raw Body (%d bytes):\n%s", len(bodyBytes), string(bodyBytes))
	return bodyBytes, nil
}

// parseFormBody parses URL-encoded form data into a WebhookPayload struct
// Returns the decoded payload and the raw form values (original PHP notation)
func (h *Handler) parseFormBody(bodyBytes []byte) (*models.WebhookPayload, url.Values, error) {
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse form data: %w", err)
	}

	// Transform PHP-style array keys to go-playground/form format.
	// Prodamus webhooks send form data as "products[0][name]" (PHP notation),
	// but the form decoder expects "products[0].name" (Go notation).
	// This transformation converts the nested bracket notation to dot notation.
	transformedValues := make(url.Values)
	for key, vals := range values {
		transformedValues[hmac.TransformPHPKeyToGoKey(key)] = vals
	}

	var payload models.WebhookPayload
	decoder := form.NewDecoder()
	if err := decoder.Decode(&payload, transformedValues); err != nil {
		return nil, nil, fmt.Errorf("failed to decode form values: %w", err)
	}

	return &payload, values, nil
}

// logWebhookRequest logs the webhook request details and payment status
func (h *Handler) logWebhookRequest(r *http.Request, bodyBytes []byte, payload *models.WebhookPayload) {
	logger.LogRequest(r, bodyBytes)
	log.Printf("Webhook payload: %+v", payload)

	if h.isSuccessStatus(payload.PaymentStatus) {
		log.Printf("Payment status: SUCCESS (order_id: %s)", payload.OrderID)
	} else if h.isFailedStatus(payload.PaymentStatus) {
		log.Printf("Payment status: FAILED (order_id: %s, status: %s, description: %s)",
			payload.OrderID, payload.PaymentStatus, payload.PaymentStatusDescription)
	} else {
		log.Printf("Payment status: UNEXPECTED %s (order_id: %s, status: %s, description: %s)",
			payload.PaymentStatus, payload.OrderID, payload.PaymentStatus, payload.PaymentStatusDescription)
	}
}

// isSuccessStatus checks if the status indicates a successful payment
func (h *Handler) isSuccessStatus(status string) bool {
	return status == models.PaymentStatusSuccess
}

// isFailedStatus checks if status indicates a failed/canceled payment
func (h *Handler) isFailedStatus(status string) bool {
	return status == models.PaymentStatusOrderCanceled || status == models.PaymentStatusOrderDenied
}

func mapProducts(src []models.Product) []services.SignatureProduct {
	if len(src) == 0 {
		return nil
	}

	result := make([]services.SignatureProduct, 0, len(src))
	for _, p := range src {
		result = append(result, services.SignatureProduct{
			Name:     p.Name,
			Price:    p.Price,
			Quantity: p.Quantity,
		})
	}

	return result
}

// verifySignature validates the webhook signature using Prodamus algorithm
// Documentation: https://help.prodamus.ru/payform/integracii/rest-api/instrukcii-dlya-samostoyatelnaya-integracii-servisov#kak-prinyat-uvedomlenie-ob-uspeshnoi-oplate
func (h *Handler) verifySignature(r *http.Request, payload models.WebhookPayload, rawFormValues url.Values) bool {
	if h.secretKey == "" {
		log.Println("Warning: PRODAMUS_SECRET_KEY is not set, rejecting webhook")
		return false
	}

	receivedSignature := r.Header.Get("Sign")
	if receivedSignature == "" {
		log.Println("Warning: No Sign header found")
		return false
	}

	// Use ALL raw form values for signature calculation (not just a subset)
	// The rawFormValues contain all fields in original PHP notation from Prodamus
	isValid, err := hmac.VerifySignatureFromFormValues(rawFormValues, h.secretKey, receivedSignature)
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

// processPayment updates user data after successful payment
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
	userData.PaymentInfo = payload

	// Generate and send invite link for the paid user
	if err := h.handleInviteLinkGeneration(userID, &userData); err != nil {
		log.Printf("Warning: %v", err)
	}

	if err := services.ChangeUser(userID, userData); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// handleInviteLinkGeneration generates invite link and queues message sending
func (h *Handler) handleInviteLinkGeneration(userID string, userData *services.User) error {
	inviteLink, err := h.generateInviteLink(userID)
	if err != nil {
		return fmt.Errorf("failed to generate invite link: %w", err)
	}

	userData.InviteLink = inviteLink

	if h.sendInviteMessage != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in sendInviteMessage for user %s: %v", userID, r)
				}
			}()
			h.sendInviteMessage(userID, inviteLink)
		}()
	}
	log.Printf("Invite link generated and queued for sending to %s", userID)

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

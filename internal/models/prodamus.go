package models

// Product represents a product in the order
type Product struct {
	Name     string `json:"name"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
	Sum      string `json:"sum"`
}

// PaymentStatus constants
const (
	PaymentStatusSuccess       = "success"
	PaymentStatusOrderCanceled = "order_canceled"
	PaymentStatusOrderDenied   = "order_denied"
)

// WebhookPayload represents the data sent by Prodamus webhook
// https://help.prodamus.ru/payform/uvedomleniya/kak-ustroena-otpravka-uvedomlenii-ob-oplate#primer-url-uvedomleniya
type WebhookPayload struct {
	OrderID                  string    `json:"order_id"` // order ID
	OrderNum                 string    `json:"order_num"`
	Domain                   string    `json:"domain"`
	Sum                      string    `json:"sum"` // amount
	CustomerPhone            string    `json:"customer_phone"`
	CustomerEmail            string    `json:"customer_email"`
	CustomerExtra            string    `json:"customer_extra"` // user_id (your custom field)
	PaymentType              string    `json:"payment_type"`
	PaymentInit              string    `json:"payment_init"` // Api (по токену), Auto (автоплатеж по подписке), Manual (клиентом)
	Commission               string    `json:"commission"`
	CommissionSum            string    `json:"commission_sum"`
	Attempt                  string    `json:"attempt"`
	Date                     string    `json:"date"`
	PaymentStatus            string    `json:"payment_status"` // success, order_canceled, order_denied
	PaymentStatusDescription string    `json:"payment_status_description"`
	Products                 []Product `json:"products"`
}

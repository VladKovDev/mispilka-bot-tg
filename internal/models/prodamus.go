package models

// Product represents a product in the order
type Product struct {
	Name     string `form:"name" json:"name"`
	Price    string `form:"price" json:"price"`
	Quantity string `form:"quantity" json:"quantity"`
	Sum      string `form:"sum" json:"sum"`
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
	OrderID                  string    `form:"order_id" json:"order_id"` // order ID
	OrderNum                 string    `form:"order_num" json:"order_num"`
	Domain                   string    `form:"domain" json:"domain"`
	Sum                      string    `form:"sum" json:"sum"` // amount
	CustomerPhone            string    `form:"customer_phone" json:"customer_phone"`
	CustomerEmail            string    `form:"customer_email" json:"customer_email"`
	CustomerExtra            string    `form:"customer_extra" json:"customer_extra"` // user_id (your custom field)
	PaymentType              string    `form:"payment_type" json:"payment_type"`
	PaymentInit              string    `form:"payment_init" json:"payment_init"` // Api (по токену), Auto (автоплатеж по подписке), Manual (клиентом)
	Commission               string    `form:"commission" json:"commission"`
	CommissionSum            string    `form:"commission_sum" json:"commission_sum"`
	Attempt                  string    `form:"attempt" json:"attempt"`
	Date                     string    `form:"date" json:"date"`
	PaymentStatus            string    `form:"payment_status" json:"payment_status"` // success, order_canceled, order_denied
	PaymentStatusDescription string    `form:"payment_status_description" json:"payment_status_description"`
	Products                 []Product `form:"products" json:"products"`
}

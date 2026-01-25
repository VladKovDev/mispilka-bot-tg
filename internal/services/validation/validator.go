package validation

import (
	"errors"
	"fmt"
	"regexp"
	"unicode"
)

var (
	// ErrInvalidScenarioName is returned when scenario name is invalid
	ErrInvalidScenarioName = errors.New("scenario name must be 1-200 characters")
	// ErrInvalidProductPrice is returned when product price is invalid
	ErrInvalidProductPrice = errors.New("price must be a positive integer (e.g., 500)")
	// ErrInvalidPrivateGroupID is returned when group ID format is invalid
	ErrInvalidPrivateGroupID = errors.New("group ID must be in format -100XXXXXXXXXX")
	// ErrFieldTooLong is returned when a field exceeds max length
	ErrFieldTooLong = errors.New("field exceeds maximum length")
	// ErrFieldEmpty is returned when a required field is empty
	ErrFieldEmpty = errors.New("field cannot be empty")
)

const (
	// MaxScenarioNameLength is the maximum length for scenario names
	MaxScenarioNameLength = 200
	// MaxProductNameLength is the maximum length for product names
	MaxProductNameLength = 200
	// MaxPaidContentLength is the maximum length for paid content description
	MaxPaidContentLength = 2000
	// MinProductPrice is the minimum allowed price (in kopeks/rubles)
	MinProductPrice = 1
	// MaxProductPrice is the maximum allowed price
	MaxProductPrice = 1000000
)

// priceRegex matches positive integers only
var priceRegex = regexp.MustCompile(`^\d+$`)

// groupIDRegex matches Telegram private group IDs: -100 followed by digits
var groupIDRegex = regexp.MustCompile(`^-100\d+$`)

// ValidateScenarioName validates the scenario name
func ValidateScenarioName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: scenario name", ErrFieldEmpty)
	}
	if len(name) > MaxScenarioNameLength {
		return fmt.Errorf("%w: max %d characters", ErrInvalidScenarioName, MaxScenarioNameLength)
	}
	return nil
}

// ValidateProductName validates the product name
func ValidateProductName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: product name", ErrFieldEmpty)
	}
	if len(name) > MaxProductNameLength {
		return fmt.Errorf("%w: max %d characters", ErrFieldTooLong, MaxProductNameLength)
	}
	return nil
}

// ValidateProductPrice validates the product price
func ValidateProductPrice(price string) error {
	if price == "" {
		return fmt.Errorf("%w: price", ErrFieldEmpty)
	}
	if !priceRegex.MatchString(price) {
		return ErrInvalidProductPrice
	}
	// Parse and check range
	var priceInt int
	for _, c := range price {
		if !unicode.IsDigit(c) {
			return ErrInvalidProductPrice
		}
		priceInt = priceInt*10 + int(c-'0')
		if priceInt > MaxProductPrice {
			return fmt.Errorf("%w: price too high (max %d)", ErrInvalidProductPrice, MaxProductPrice)
		}
	}
	if priceInt < MinProductPrice {
		return fmt.Errorf("%w: price too low (min %d)", ErrInvalidProductPrice, MinProductPrice)
	}
	return nil
}

// ValidatePaidContent validates the paid content description
func ValidatePaidContent(content string) error {
	if content == "" {
		return fmt.Errorf("%w: paid content", ErrFieldEmpty)
	}
	if len(content) > MaxPaidContentLength {
		return fmt.Errorf("%w: max %d characters", ErrFieldTooLong, MaxPaidContentLength)
	}
	return nil
}

// ValidatePrivateGroupID validates the Telegram private group ID
func ValidatePrivateGroupID(groupID string) error {
	if groupID == "" {
		return fmt.Errorf("%w: group ID", ErrFieldEmpty)
	}
	if !groupIDRegex.MatchString(groupID) {
		return ErrInvalidPrivateGroupID
	}
	return nil
}

// ValidateScenarioConfig validates all scenario configuration fields
func ValidateScenarioConfig(name, productName, productPrice, paidContent, groupID string) error {
	if err := ValidateScenarioName(name); err != nil {
		return err
	}
	if err := ValidateProductName(productName); err != nil {
		return err
	}
	if err := ValidateProductPrice(productPrice); err != nil {
		return err
	}
	if err := ValidatePaidContent(paidContent); err != nil {
		return err
	}
	if err := ValidatePrivateGroupID(groupID); err != nil {
		return err
	}
	return nil
}

package validation

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

var (
	// ErrInvalidMessageTiming is returned when message timing format is invalid
	ErrInvalidMessageTiming = errors.New("timing must be in format like '1h 30m', '90m', or '2h'")
	// ErrInvalidButtonFormat is returned when button format is invalid
	ErrInvalidButtonFormat = errors.New("button format must be: Button Text|url|https://example.com or Button Text|callback|action_name")
)

// timingRegex matches timing formats like "1h 30m", "90m", "2h"
var timingRegex = regexp.MustCompile(`^(?:(\d+)h)?\s*(?:(\d+)m)?$`)

// ValidateMessageTiming validates the message timing input
// Accepts formats: "1h 30m", "90m", "2h", "1h", "30m", etc.
func ValidateMessageTiming(timing string) error {
	if timing == "" {
		return fmt.Errorf("%w: timing cannot be empty", ErrInvalidMessageTiming)
	}

	// Trim spaces
	timing = strings.TrimSpace(timing)

	// Check if it matches the timing pattern
	if !timingRegex.MatchString(timing) {
		return ErrInvalidMessageTiming
	}

	// Parse and validate the timing
	matches := timingRegex.FindStringSubmatch(timing)
	hoursStr := matches[1]
	minutesStr := matches[2]

	hours := 0
	minutes := 0

	if hoursStr != "" {
		h, err := strconv.Atoi(hoursStr)
		if err != nil || h < 0 || h > 8760 { // Max 1 year
			return fmt.Errorf("%w: hours must be 0-8760", ErrInvalidMessageTiming)
		}
		hours = h
	}

	if minutesStr != "" {
		m, err := strconv.Atoi(minutesStr)
		if err != nil || m < 0 || m > 59 {
			return fmt.Errorf("%w: minutes must be 0-59", ErrInvalidMessageTiming)
		}
		minutes = m
	}

	// Check that total time is at least 1 minute
	totalMinutes := hours*60 + minutes
	if totalMinutes < 1 {
		return fmt.Errorf("%w: total time must be at least 1 minute", ErrInvalidMessageTiming)
	}

	// Check that total time is not too long (max 1 year)
	if totalMinutes > 525600 { // 365 days * 24 hours * 60 minutes
		return fmt.Errorf("%w: total time cannot exceed 1 year", ErrInvalidMessageTiming)
	}

	return nil
}

// ParseMessageTiming parses the timing string and returns hours and minutes
func ParseMessageTiming(timing string) (hours, minutes int, err error) {
	timing = strings.TrimSpace(timing)
	matches := timingRegex.FindStringSubmatch(timing)

	if matches == nil {
		return 0, 0, ErrInvalidMessageTiming
	}

	if matches[1] != "" {
		h, _ := strconv.Atoi(matches[1])
		hours = h
	}

	if matches[2] != "" {
		m, _ := strconv.Atoi(matches[2])
		minutes = m
	}

	return hours, minutes, nil
}

// ValidateMessageButtons validates the button configuration
// Format: "Button Text|url|https://example.com" or "Button Text|callback|action_name"
// Multiple buttons separated by newlines
func ValidateMessageButtons(buttons string) error {
	if buttons == "" {
		return nil // Empty is valid (no buttons)
	}

	lines := strings.Split(strings.TrimSpace(buttons), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Each button should have: Text|Type|Value
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			return fmt.Errorf("%w: line %d must have format 'Text|type|value'", ErrInvalidButtonFormat, i+1)
		}

		text := strings.TrimSpace(parts[0])
		btnType := strings.TrimSpace(parts[1])
		value := strings.TrimSpace(parts[2])

		if text == "" {
			return fmt.Errorf("%w: button text cannot be empty (line %d)", ErrInvalidButtonFormat, i+1)
		}

		if btnType != "url" && btnType != "callback" {
			return fmt.Errorf("%w: button type must be 'url' or 'callback' (line %d)", ErrInvalidButtonFormat, i+1)
		}

		if value == "" {
			return fmt.Errorf("%w: button value cannot be empty (line %d)", ErrInvalidButtonFormat, i+1)
		}

		if btnType == "url" && !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("%w: URL buttons must start with http:// or https:// (line %d)", ErrInvalidButtonFormat, i+1)
		}
	}

	return nil
}

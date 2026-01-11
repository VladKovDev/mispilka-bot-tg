package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

// VerifySignatureFromFormValues verifies a signature using raw form values from a webhook.
// This is used for Prodamus webhook verification where ALL fields from the POST body
// must be included in the signature calculation.
// Keys are transformed from PHP notation (e.g., "products[0][name]") to Go notation
// (e.g., "products[0].name") before signature calculation.
func VerifySignatureFromFormValues(values url.Values, secretKey, receivedSignature string) (bool, error) {
	if receivedSignature == "" {
		return false, fmt.Errorf("received signature is empty")
	}

	// Convert url.Values (map[string][]string) to map[string]string with key transformation
	// Transform PHP-style array keys to Go notation for signature calculation
	data := make(map[string]string)
	for key, vals := range values {
		if len(vals) > 0 {
			data[TransformPHPKeyToGoKey(key)] = vals[0]
		}
	}

	log.Printf("values for signature: %+v", values)
	log.Printf("data for signature: %+v", data)

	// Calculate expected signature using all fields
	expectedSignature, err := CreateSignature(data, secretKey)
	if err != nil {
		return false, err
	}
	log.Printf("expectedSignature: %s", expectedSignature)

	// Compare signatures case-insensitively (as per Prodamus docs)
	return strings.EqualFold(expectedSignature, receivedSignature), nil
}

// CreateSignature creates a signature according to Prodamus algorithm:
// 1. Convert all values to strings
// 2. Sort all keys alphabetically, recursively (including nested)
// 3. Convert to JSON string
// 4. Escape forward slashes in JSON
// 5. Sign with HMAC-SHA256 using the secret key
func CreateSignature(data interface{}, secretKey string) (string, error) {
	if secretKey == "" {
		return "", fmt.Errorf("secret key is empty")
	}

	// Convert data to map[string]interface{} for processing
	dataMap, err := convertToMap(data)
	if err != nil {
		return "", fmt.Errorf("failed to convert data to map: %w", err)
	}

	// Step 1: Convert all values to strings recursively
	dataMap = convertValuesToStrings(dataMap)

	// Step 2: Sort keys alphabetically, recursively
	sortMapKeys(dataMap)

	// Step 3: Convert to JSON string
	jsonBytes, err := json.Marshal(dataMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	jsonStr := string(jsonBytes)

	// Note: Step 4 (escape forward slashes) is NOT included because:
	// 1. The PHP library code in Prodamus docs doesn't do it
	// 2. json_encode with JSON_UNESCAPED_UNICODE doesn't escape '/'
	// 3. Go's json.Marshal doesn't escape '/' by default
	// This appears to be a documentation inconsistency

	// Step 5: Sign with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(jsonStr))
	signature := hex.EncodeToString(mac.Sum(nil))

	return signature, nil
}

// VerifySignature verifies the signature according to Prodamus algorithm
func VerifySignature(data interface{}, secretKey string, receivedSignature string) (bool, error) {
	if receivedSignature == "" {
		return false, fmt.Errorf("received signature is empty")
	}

	// Calculate expected signature
	expectedSignature, err := CreateSignature(data, secretKey)
	if err != nil {
		return false, err
	}

	// Compare signatures case-insensitively (as per Prodamus docs)
	return strings.EqualFold(expectedSignature, receivedSignature), nil
}

// convertToMap converts any data to map[string]interface{}
func convertToMap(data interface{}) (map[string]interface{}, error) {
	val := reflect.ValueOf(data)

	// Handle nil
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return make(map[string]interface{}), nil
		}
		val = val.Elem()
	}

	if val.Kind() == reflect.Map {
		result := make(map[string]interface{})
		for _, key := range val.MapKeys() {
			// Only accept string keys
			if key.Kind() == reflect.String {
				result[key.String()] = val.MapIndex(key).Interface()
			}
		}
		return result, nil
	}

	// Try to marshal and unmarshal as JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// convertValuesToStrings recursively converts all values to strings
func convertValuesToStrings(data map[string]interface{}) map[string]interface{} {
	for key, value := range data {
		data[key] = convertValueToString(value)
	}
	return data
}

// convertValueToString converts a single value to string
func convertValueToString(value interface{}) interface{} {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string, float32, float64, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, bool:
		return fmt.Sprintf("%v", v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertValueToString(item)
		}
		return result
	case map[string]interface{}:
		return convertValuesToStrings(v)
	default:
		// For any other type, try to convert to string
		return fmt.Sprintf("%v", v)
	}
}

// sortMapKeys recursively sorts map keys alphabetically
func sortMapKeys(data map[string]interface{}) {
	// Get sorted keys
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Create new map with sorted keys
	sortedData := make(map[string]interface{})
	for _, key := range keys {
		sortedData[key] = data[key]
	}

	// Recursively sort nested maps
	for key, value := range sortedData {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			sortMapKeys(nestedMap)
			sortedData[key] = nestedMap
		} else if nestedSlice, ok := value.([]interface{}); ok {
			// Sort arrays that contain maps
			for i, item := range nestedSlice {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					sortMapKeys(nestedMap)
					nestedSlice[i] = nestedMap
				}
			}
		}
	}

	// Copy back to original map (clear first)
	for key := range data {
		delete(data, key)
	}
	for key, value := range sortedData {
		data[key] = value
	}
}

// TransformPHPKeyToGoKey converts PHP-style array keys to Go notation.
// PHP notation: products[0][name]
// Go notation: products[0].name
func TransformPHPKeyToGoKey(key string) string {
	transformedKey := strings.ReplaceAll(key, "][", "].")
	// Remove trailing ] from field names
	if strings.Contains(transformedKey, "[") && strings.HasSuffix(transformedKey, "]") {
		transformedKey = transformedKey[:len(transformedKey)-1]
	}
	return transformedKey
}

package logger

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

// LogRequest logs all request details
func LogRequest(r *http.Request, bodyBytes []byte) {
	contentType := r.Header.Get("Content-Type")

	log.Println("\n=== Start Request ===")

	LogRequestBasic(r)
	LogHeaders(r.Header)
	LogQueryParams(r.URL.Query())
	LogBody(contentType, bodyBytes)

	log.Println("=== End Request ===\n ")
}

// LogRequestBasic logs basic request information
func LogRequestBasic(r *http.Request) {
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Host: %s", r.Host)
}

// LogHeaders logs all request headers
func LogHeaders(header http.Header) {
	log.Printf("--- Headers ---")
	for key, values := range header {
		for _, value := range values {
			log.Printf("  %s: %s", key, value)
		}
	}
}

// LogQueryParams logs query parameters
func LogQueryParams(query url.Values) {
	log.Println("--- Query Parameters ---")
	for key, values := range query {
		for _, value := range values {
			log.Printf("  %s: %s", key, value)
		}
	}
}

// LogBody logs the request body with appropriate formatting
func LogBody(contentType string, bodyBytes []byte) {
	log.Printf("--- Body (raw) ---")
	log.Printf("%s", string(bodyBytes))

	log.Println("--- Parsed Body ---")

	if contentType == "application/json" || contentType == "application/json; charset=utf-8" {
		var jsonData interface{}
		if err := json.Unmarshal(bodyBytes, &jsonData); err != nil {
			log.Printf("Error parsing JSON: %v", err)
		} else {
			LogJSONData(jsonData, "")
		}
	} else {
		log.Printf("Non-JSON content type: %s", contentType)
		// Parse and display all form-data fields
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			log.Printf("Error parsing form-data: %v", err)
		} else {
			for key, vals := range values {
				for _, val := range vals {
					log.Printf("  %s: %s", key, val)
				}
			}
		}
	}
}

// LogJSONData logs JSON data with indentation
func LogJSONData(data interface{}, indent string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch val := value.(type) {
			case map[string]interface{}:
				log.Printf("%s%s:", indent, key)
				LogJSONData(val, indent+"  ")
			case []interface{}:
				log.Printf("%s%s (array, len=%d):", indent, key, len(val))
				for i, item := range val {
					log.Printf("%s  [%d]:", indent, i)
					LogJSONData(item, indent+"    ")
				}
			default:
				log.Printf("%s%s: %v (%T)", indent, key, val, val)
			}
		}
	case []interface{}:
		for i, item := range v {
			log.Printf("%s[%d]:", indent, i)
			LogJSONData(item, indent+"  ")
		}
	default:
		log.Printf("%s%v (%T)", indent, v, v)
	}
}

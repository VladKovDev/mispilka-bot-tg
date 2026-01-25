package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

func ReadJSON[T any](path string) (T, error) {
	var data T
	raw, err := os.ReadFile(path)
	if err != nil {
		return data, fmt.Errorf("read JSON file %q: %w", path, err)
	}

	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("unmarshal JSON file %q: %w", path, err)
	}
	return data, nil
}

func WriteJSON[T any](path string, data T) error {
	raw, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("marshal JSON file %q: %w", path, err)
	}

	err = os.WriteFile(path, raw, 0644)
	if err != nil {
		return fmt.Errorf("write JSON file %q: %w", path, err)
	}

	return nil
}

func ReadJSONRetry[T any](path string, attempts int) (T, error) {
	var data T
	var err error
	for i := 0; i < attempts; i++ {
		data, err = ReadJSON[T](path)
		if err == nil {
			return data, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return data, fmt.Errorf("ReadJSON failed after %d attempts: %w", attempts, err)
}

func WriteJSONRetry[T any](path string, data T, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = WriteJSON(path, data)
		if err == nil {
			return nil
		}
		// Exponential backoff: 100ms, 200ms, 300ms...
		if i < attempts-1 {
			time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("WriteJSON failed after %d attempts: %w", attempts, err)
}

func CheckStorage(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, create it
			_, err = os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create file %q: %w", path, err)
			}
			log.Printf("created file: %v", path)

			// Initialize with empty struct
			v := struct{}{}
			if err := WriteJSON(path, v); err != nil {
				return fmt.Errorf("failed to initialize file %q: %w", path, err)
			}
		} else {
			// Some other error occurred
			return fmt.Errorf("failed to stat file %q: %w", path, err)
		}
	}
	return nil
}

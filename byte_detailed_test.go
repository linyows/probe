//go:build experiments
// +build experiments

package probe

import (
	"encoding/base64"
	"fmt"
	"testing"
)

// TestByteHandlingDetailed provides detailed analysis of []byte handling
func TestByteHandlingDetailed(t *testing.T) {
	binaryData := []byte("Hello, binary world!")

	t.Run("encodeValueWithTypePrefix with []byte", func(t *testing.T) {
		encoded := encodeValueWithTypePrefix(binaryData)
		t.Logf("[]byte encoded as: %q", encoded)
		t.Logf("This goes to default case: fmt.Sprintf(\"%%v\", input)")

		// Verify what fmt.Sprintf("%v", []byte) produces
		expected := fmt.Sprintf("%v", binaryData) // [72 101 108 108 111 44 32 98 105 110 97 114 121 32 119 111 114 108 100 33]
		if encoded != expected {
			t.Errorf("Expected %q, got %q", expected, encoded)
		} else {
			t.Logf("[]byte becomes numeric array representation: %s", encoded)
		}
	})

	t.Run("FlattenInterface with []byte", func(t *testing.T) {
		data := map[string]any{
			"binary": binaryData,
			"text":   "normal string",
		}

		flattened := FlattenInterface(data)

		t.Logf("Flattened map:")
		for k, v := range flattened {
			t.Logf("  %s: %q", k, v)
		}

		// The []byte should be converted to numeric array string representation
		binaryFlattened := flattened["binary"]
		expectedBinary := fmt.Sprintf("%v", binaryData)
		if binaryFlattened != expectedBinary {
			t.Errorf("Expected binary to be %q, got %q", expectedBinary, binaryFlattened)
		}
	})

	t.Run("UnflattenInterface converts array string back", func(t *testing.T) {
		// Create a flattened representation manually
		flattened := map[string]string{
			"binary": "[72 101 108 108 111 44 32 98 105 110 97 114 121 32 119 111 114 108 100 33]",
			"text":   "normal string",
		}

		unflattened := UnflattenInterface(flattened)

		t.Logf("Unflattened map:")
		for k, v := range unflattened {
			t.Logf("  %s: %T = %v", k, v, v)
		}

		// The numeric array string gets parsed as an array
		binaryResult := unflattened["binary"]
		if arr, ok := binaryResult.([]any); ok {
			t.Logf("Array conversion successful with %d elements", len(arr))

			// Convert back to []byte manually to verify data integrity
			restoredBytes := make([]byte, len(arr))
			for i, val := range arr {
				if intVal, ok := val.(int); ok {
					restoredBytes[i] = byte(intVal)
				} else {
					t.Errorf("Array element %d is not int: %T", i, val)
				}
			}

			if string(restoredBytes) == string(binaryData) {
				t.Logf("Data integrity preserved: %q", string(restoredBytes))
			} else {
				t.Errorf("Data integrity lost: expected %q, got %q", string(binaryData), string(restoredBytes))
			}
		} else {
			t.Errorf("Expected array, got %T", binaryResult)
		}
	})
}

// TestActualHTTPBodyHandling tests how HTTP body is actually handled in practice
func TestActualHTTPBodyHandling(t *testing.T) {
	// Simulate what happens in actual HTTP action usage
	httpBody := []byte(`{"user":"john","data":"\x00\x01\x02"}`) // JSON with some binary

	httpData := map[string]any{
		"method": "POST",
		"url":    "https://api.example.com",
		"body":   httpBody,
	}

	t.Run("Full cycle: Flatten -> Unflatten", func(t *testing.T) {
		// Step 1: Flatten (like when preparing for protobuf)
		flattened := FlattenInterface(httpData)
		t.Logf("Flattened body: %q", flattened["body"])

		// Step 2: Unflatten (like when reconstructing from protobuf)
		unflattened := UnflattenInterface(flattened)
		bodyResult := unflattened["body"]

		t.Logf("Body type after unflatten: %T", bodyResult)

		if arr, ok := bodyResult.([]any); ok {
			// Convert array back to []byte
			reconstructedBody := make([]byte, len(arr))
			for i, val := range arr {
				if intVal, ok := val.(int); ok {
					reconstructedBody[i] = byte(intVal)
				}
			}

			t.Logf("Original body:      %q", string(httpBody))
			t.Logf("Reconstructed body: %q", string(reconstructedBody))

			if string(reconstructedBody) == string(httpBody) {
				t.Logf("HTTP body survives Flatten/Unflatten cycle")
			} else {
				t.Errorf("HTTP body corrupted during cycle")
			}
		} else {
			t.Errorf("Expected body to be array after unflatten, got %T", bodyResult)
		}
	})
}

// TestCompareByteHandlingApproaches compares different approaches for handling []byte
func TestCompareByteHandlingApproaches(t *testing.T) {
	testBytes := []byte("Binary data: \x00\x01\x02\x03\xFF test")

	approaches := map[string]func([]byte) (string, []byte){
		"Current FlattenInterface": func(b []byte) (string, []byte) {
			data := map[string]any{"data": b}
			flattened := FlattenInterface(data)
			unflattened := UnflattenInterface(flattened)

			if arr, ok := unflattened["data"].([]any); ok {
				result := make([]byte, len(arr))
				for i, val := range arr {
					if intVal, ok := val.(int); ok {
						result[i] = byte(intVal)
					}
				}
				return "array of ints", result
			}
			return "unknown", nil
		},
		"Base64 encoding": func(b []byte) (string, []byte) {
			// This would be the approach needed for structpb
			encoded := base64.StdEncoding.EncodeToString(b)
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			return "base64 string", decoded
		},
	}

	for name, approach := range approaches {
		t.Run(name, func(t *testing.T) {
			method, result := approach(testBytes)

			t.Logf("Method: %s", method)
			t.Logf("Original:  %q (len=%d)", testBytes, len(testBytes))
			t.Logf("Result:    %q (len=%d)", result, len(result))

			if string(result) == string(testBytes) {
				t.Logf("Data integrity preserved")
			} else {
				t.Errorf("Data integrity lost")
			}
		})
	}
}

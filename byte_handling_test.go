//go:build experiments
// +build experiments

package probe

import (
	"encoding/base64"
	"reflect"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestByteHandling tests how []byte is handled in both approaches
func TestByteHandling(t *testing.T) {
	// Test data with []byte (like HTTP body)
	binaryData := []byte("Hello, World! This is binary data: \x00\x01\x02\x03\xFF")
	
	testData := map[string]any{
		"text": "normal string",
		"binary_data": binaryData,
		"nested": map[string]any{
			"image": []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), // PNG header-like data
			"count": 42,
		},
	}

	t.Run("FlattenInterface handles []byte", func(t *testing.T) {
		flattened := FlattenInterface(testData)
		
		// Check how []byte is flattened
		t.Logf("Flattened binary_data: %q", flattened["binary_data"])
		t.Logf("Flattened nested image: %q", flattened["nested__image"])
		
		// Verify it's converted to string representation
		if _, exists := flattened["binary_data"]; !exists {
			t.Error("binary_data should exist in flattened map")
		}
		
		if _, exists := flattened["nested__image"]; !exists {
			t.Error("nested__image should exist in flattened map")
		}
	})

	t.Run("UnflattenInterface restores []byte as string", func(t *testing.T) {
		flattened := FlattenInterface(testData)
		unflattened := UnflattenInterface(flattened)
		
		// Check what type binary_data becomes after unflatten
		binaryResult := unflattened["binary_data"]
		t.Logf("Unflattened binary_data type: %T, value: %q", binaryResult, binaryResult)
		
		nestedMap := unflattened["nested"].(map[string]any)
		imageResult := nestedMap["image"]
		t.Logf("Unflattened image type: %T, value: %q", imageResult, imageResult)
		
		// FlattenInterface/UnflattenInterface converts []byte to string via fmt.Sprintf
		if _, ok := binaryResult.(string); !ok {
			t.Errorf("Expected binary_data to be string after unflatten, got %T", binaryResult)
		}
		
		if _, ok := imageResult.(string); !ok {
			t.Errorf("Expected image to be string after unflatten, got %T", imageResult)
		}
	})

	t.Run("google.protobuf.Struct cannot handle []byte directly", func(t *testing.T) {
		_, err := structpb.NewStruct(testData)
		if err == nil {
			t.Error("Expected error when converting []byte to structpb.Struct")
		}
		
		t.Logf("structpb error (expected): %v", err)
		
		// This should fail because structpb doesn't support []byte directly
	})

	t.Run("google.protobuf.Struct with base64 encoded []byte", func(t *testing.T) {
		// Convert []byte to base64 strings for structpb compatibility
		convertedData := map[string]any{
			"text": "normal string",
			"binary_data": base64.StdEncoding.EncodeToString(binaryData),
			"nested": map[string]any{
				"image": base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")),
				"count": 42,
			},
		}
		
		structData, err := structpb.NewStruct(convertedData)
		if err != nil {
			t.Fatal(err)
		}
		
		result := structData.AsMap()
		
		// Verify base64 strings are preserved
		binaryResult := result["binary_data"].(string)
		decodedData, err := base64.StdEncoding.DecodeString(binaryResult)
		if err != nil {
			t.Fatal(err)
		}
		
		if !reflect.DeepEqual(decodedData, binaryData) {
			t.Error("Base64 round-trip failed")
		}
		
		t.Logf("structpb with base64: Success ✓")
	})
}

// TestHTTPBodyLikeScenario tests HTTP-like scenario with binary body
func TestHTTPBodyLikeScenario(t *testing.T) {
	// Simulate HTTP request/response with binary body
	imageData := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x10\x00\x00\x00\x10") // Minimal PNG
	
	httpReqRes := map[string]any{
		"req": map[string]any{
			"method": "POST",
			"url":    "https://api.example.com/upload",
			"headers": map[string]any{
				"Content-Type": "image/png",
			},
			"body": imageData, // []byte body
		},
		"res": map[string]any{
			"status": "200 OK",
			"code":   200,
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": []byte(`{"success":true,"id":"img123"}`), // []byte response
		},
	}

	t.Run("Current FlattenInterface approach", func(t *testing.T) {
		flattened := FlattenInterface(httpReqRes)
		
		// Check how request and response bodies are flattened
		reqBody := flattened["req__body"]
		resBody := flattened["res__body"]
		
		t.Logf("Flattened req body: %q", reqBody)
		t.Logf("Flattened res body: %q", resBody)
		
		// Unflatten back
		unflattened := UnflattenInterface(flattened)
		
		reqMap := unflattened["req"].(map[string]any)
		resMap := unflattened["res"].(map[string]any)
		
		reqBodyResult := reqMap["body"]
		resBodyResult := resMap["body"]
		
		t.Logf("Unflattened req body type: %T", reqBodyResult)
		t.Logf("Unflattened res body type: %T", resBodyResult)
		
		// Both should be strings (not []byte)
		if _, ok := reqBodyResult.(string); !ok {
			t.Errorf("Expected req body to be string, got %T", reqBodyResult)
		}
		if _, ok := resBodyResult.(string); !ok {
			t.Errorf("Expected res body to be string, got %T", resBodyResult)
		}
	})

	t.Run("structpb approach requires pre-encoding", func(t *testing.T) {
		// Pre-encode binary data as base64
		encodedHttpReqRes := map[string]any{
			"req": map[string]any{
				"method": "POST",
				"url":    "https://api.example.com/upload",
				"headers": map[string]any{
					"Content-Type": "image/png",
				},
				"body": base64.StdEncoding.EncodeToString(imageData),
			},
			"res": map[string]any{
				"status": "200 OK",
				"code":   200,
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
				"body": base64.StdEncoding.EncodeToString([]byte(`{"success":true,"id":"img123"}`)),
			},
		}
		
		structData, err := structpb.NewStruct(encodedHttpReqRes)
		if err != nil {
			t.Fatal(err)
		}
		
		result := structData.AsMap()
		
		// Verify we can decode back to original binary data
		reqMap := result["req"].(map[string]any)
		resMap := result["res"].(map[string]any)
		
		reqBodyB64 := reqMap["body"].(string)
		resBodyB64 := resMap["body"].(string)
		
		reqBodyDecoded, _ := base64.StdEncoding.DecodeString(reqBodyB64)
		resBodyDecoded, _ := base64.StdEncoding.DecodeString(resBodyB64)
		
		if !reflect.DeepEqual(reqBodyDecoded, imageData) {
			t.Error("Request body decode failed")
		}
		
		expectedResBody := []byte(`{"success":true,"id":"img123"}`)
		if !reflect.DeepEqual(resBodyDecoded, expectedResBody) {
			t.Error("Response body decode failed")
		}
		
		t.Logf("structpb with pre-encoded base64: Success ✓")
	})
}

// TestByteTypePrefix tests if there's a type prefix for []byte
func TestByteTypePrefix(t *testing.T) {
	testBytes := []byte("test binary data")
	
	// Test what happens when we encode []byte using encodeValueWithTypePrefix
	encoded := encodeValueWithTypePrefix(testBytes)
	t.Logf("Encoded []byte: %q", encoded)
	
	// Test decoding
	decoded := decodeValueWithTypePrefix(encoded)
	t.Logf("Decoded type: %T, value: %q", decoded, decoded)
	
	// Since []byte is not in the switch cases, it should go to default case
	// which uses fmt.Sprintf("%v", input)
	expectedString := string(testBytes) // []byte converts to string via %v
	if decoded != expectedString {
		t.Errorf("Expected %q, got %q", expectedString, decoded)
	}
}
//go:build experiments
// +build experiments

package probe

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestArrayOrderPreservationSimple tests whether array order is preserved
func TestArrayOrderPreservationSimple(t *testing.T) {
	// Simple test data focusing on array order
	testData := map[string]any{
		"browser_actions": []any{
			"step1_navigate",
			"step2_click", 
			"step3_type",
			"step4_submit",
		},
		"priorities": []any{"high", "medium", "low"},
		"numbers":    []any{10, 20, 30, 40, 50},
	}

	t.Run("FlattenInterface preserves order", func(t *testing.T) {
		flattened := FlattenInterface(testData)
		unflattened := UnflattenInterface(flattened)
		
		// Check browser_actions order
		origActions := testData["browser_actions"].([]any)
		convActions := unflattened["browser_actions"].([]any)
		
		if len(origActions) != len(convActions) {
			t.Fatalf("Length mismatch: %d vs %d", len(origActions), len(convActions))
		}
		
		for i, orig := range origActions {
			if orig.(string) != convActions[i].(string) {
				t.Errorf("Order mismatch at index %d: %s vs %s", i, orig, convActions[i])
			}
		}
		
		t.Logf("FlattenInterface: Order preserved ✓")
	})

	t.Run("google.protobuf.Struct preserves order", func(t *testing.T) {
		structData, err := structpb.NewStruct(testData)
		if err != nil {
			t.Fatal(err)
		}
		
		converted := structData.AsMap()
		
		// Check browser_actions order
		origActions := testData["browser_actions"].([]any)
		convActions := converted["browser_actions"].([]any)
		
		if len(origActions) != len(convActions) {
			t.Fatalf("Length mismatch: %d vs %d", len(origActions), len(convActions))
		}
		
		for i, orig := range origActions {
			if orig.(string) != convActions[i].(string) {
				t.Errorf("Order mismatch at index %d: %s vs %s", i, orig, convActions[i])
			}
		}
		
		t.Logf("google.protobuf.Struct: Order preserved ✓")
	})

	t.Run("JSON baseline preserves order", func(t *testing.T) {
		jsonData, err := json.Marshal(testData)
		if err != nil {
			t.Fatal(err)
		}
		
		var converted map[string]any
		err = json.Unmarshal(jsonData, &converted)
		if err != nil {
			t.Fatal(err)
		}
		
		// Check browser_actions order
		origActions := testData["browser_actions"].([]any)
		convActions := converted["browser_actions"].([]any)
		
		if len(origActions) != len(convActions) {
			t.Fatalf("Length mismatch: %d vs %d", len(origActions), len(convActions))
		}
		
		for i, orig := range origActions {
			if orig.(string) != convActions[i].(string) {
				t.Errorf("Order mismatch at index %d: %s vs %s", i, orig, convActions[i])
			}
		}
		
		t.Logf("JSON: Order preserved ✓")
	})
}

// TestComplexArrayOrder tests with more complex nested structures
func TestComplexArrayOrder(t *testing.T) {
	// Browser action-like data structure
	testData := map[string]any{
		"actions": []any{
			map[string]any{
				"type":     "navigate",
				"url":      "https://example.com",
				"sequence": 1,
			},
			map[string]any{
				"type":     "wait",
				"selector": "#content",
				"sequence": 2,
			},
			map[string]any{
				"type":     "click",
				"selector": "#login-btn",
				"sequence": 3,
			},
			map[string]any{
				"type":     "type",
				"selector": "#username",
				"value":    "testuser",
				"sequence": 4,
			},
		},
	}

	// Test FlattenInterface
	flattened := FlattenInterface(testData)
	unflattened := UnflattenInterface(flattened)
	
	convActions := unflattened["actions"].([]any)
	
	// Verify order by checking sequence numbers
	for i, action := range convActions {
		actionMap := action.(map[string]any)
		expectedSeq := i + 1
		actualSeq := int(actionMap["sequence"].(int)) // UnflattenInterface preserves int type
		
		if expectedSeq != actualSeq {
			t.Errorf("FlattenInterface: Order mismatch at index %d: expected sequence %d, got %d", 
				i, expectedSeq, actualSeq)
		}
	}

	// Test google.protobuf.Struct
	structData, err := structpb.NewStruct(testData)
	if err != nil {
		t.Fatal(err)
	}
	
	structConverted := structData.AsMap()
	structActions := structConverted["actions"].([]any)
	
	// Verify order by checking sequence numbers
	for i, action := range structActions {
		actionMap := action.(map[string]any)
		expectedSeq := i + 1
		actualSeq := int(actionMap["sequence"].(float64)) // structpb converts int to float64
		
		if expectedSeq != actualSeq {
			t.Errorf("google.protobuf.Struct: Order mismatch at index %d: expected sequence %d, got %d", 
				i, expectedSeq, actualSeq)
		}
	}
	
	t.Logf("Both methods preserve complex array order ✓")
}

// TestStabilityUnderMultipleConversions tests stability over multiple round-trips
func TestStabilityUnderMultipleConversions(t *testing.T) {
	originalData := map[string]any{
		"steps": []any{
			"initialize",
			"authenticate", 
			"process",
			"cleanup",
		},
	}

	// Test FlattenInterface stability
	result := originalData
	for i := 0; i < 5; i++ {
		flattened := FlattenInterface(result)
		result = UnflattenInterface(flattened)
	}
	
	origSteps := originalData["steps"].([]any)
	resultSteps := result["steps"].([]any)
	
	for i, orig := range origSteps {
		if orig.(string) != resultSteps[i].(string) {
			t.Errorf("FlattenInterface stability: Order changed after multiple conversions at index %d: %s vs %s", 
				i, orig, resultSteps[i])
		}
	}

	// Test google.protobuf.Struct stability
	result2 := originalData
	for i := 0; i < 5; i++ {
		structData, err := structpb.NewStruct(result2)
		if err != nil {
			t.Fatal(err)
		}
		result2 = structData.AsMap()
	}
	
	result2Steps := result2["steps"].([]any)
	
	for i, orig := range origSteps {
		if orig.(string) != result2Steps[i].(string) {
			t.Errorf("google.protobuf.Struct stability: Order changed after multiple conversions at index %d: %s vs %s", 
				i, orig, result2Steps[i])
		}
	}
	
	t.Logf("Both methods maintain order stability over multiple conversions ✓")
}
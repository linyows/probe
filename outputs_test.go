package probe

import (
	"testing"
)

func TestOutputsFlatAccess(t *testing.T) {
	outputs := NewOutputs()

	// Test case 1: Normal case - no conflicts
	outputs.Set("auth", map[string]any{
		"token":   "secret123",
		"user_id": "user456",
	})

	// Both access methods should work
	stepOutputs, exists := outputs.Get("auth")
	if !exists {
		t.Fatal("Expected step 'auth' to exist")
	}
	if stepOutputs["token"] != "secret123" {
		t.Errorf("Expected token 'secret123', got %v", stepOutputs["token"])
	}

	flatToken, exists := outputs.GetFlat("token")
	if !exists {
		t.Fatal("Expected flat access to 'token' to exist")
	}
	if flatToken != "secret123" {
		t.Errorf("Expected flat token 'secret123', got %v", flatToken)
	}

	flatUserID, exists := outputs.GetFlat("user_id")
	if !exists {
		t.Fatal("Expected flat access to 'user_id' to exist")
	}
	if flatUserID != "user456" {
		t.Errorf("Expected flat user_id 'user456', got %v", flatUserID)
	}

	// Test case 2: Conflict with step ID
	outputs.Set("token", map[string]any{
		"value": "different_token",
	})

	// Debug: Check all data
	allData := outputs.GetAll()
	t.Logf("All data after setting step 'token': %+v", allData)

	// Flat access for "token" should still point to original
	flatToken, exists = outputs.GetFlat("token")
	if !exists {
		t.Fatalf("Expected flat access to 'token' to still exist. Current data: %+v", outputs.data)
	}
	if flatToken != "secret123" {
		t.Errorf("Expected original flat token 'secret123', got %v", flatToken)
	}

	// Step-based access should NOT work for conflicting step because flat data takes priority
	tokenStepOutputs, exists := outputs.Get("token")
	if exists {
		t.Errorf("Expected step 'token' to NOT exist due to flat data conflict, but got: %+v", tokenStepOutputs)
	}

	// Test case 3: Duplicate output names
	outputs.Set("profile", map[string]any{
		"token": "profile_token", // Conflicts with existing "token"
		"name":  "john_doe",
	})

	// Original flat access should be preserved
	flatToken, exists = outputs.GetFlat("token")
	if !exists {
		t.Fatal("Expected flat access to 'token' to still exist")
	}
	if flatToken != "secret123" {
		t.Errorf("Expected original flat token 'secret123', got %v", flatToken)
	}

	// New non-conflicting output should be accessible via flat access
	flatName, exists := outputs.GetFlat("name")
	if !exists {
		t.Fatal("Expected flat access to 'name' to exist")
	}
	if flatName != "john_doe" {
		t.Errorf("Expected flat name 'john_doe', got %v", flatName)
	}

	// Check conflicts
	conflicts := outputs.GetConflicts()
	if len(conflicts["token"]) != 2 {
		t.Errorf("Expected 2 conflicts for 'token', got %v", conflicts["token"])
	}

	expectedSteps := map[string]bool{"auth": true, "profile": true}
	for _, stepID := range conflicts["token"] {
		if !expectedSteps[stepID] {
			t.Errorf("Unexpected step in conflicts: %s", stepID)
		}
	}
}

func TestOutputsGetAll(t *testing.T) {
	outputs := NewOutputs()

	outputs.Set("auth", map[string]any{
		"token": "secret123",
	})

	outputs.Set("profile", map[string]any{
		"name": "john_doe",
	})

	all := outputs.GetAll()

	// Check step-based access
	authOutputs, ok := all["auth"].(map[string]any)
	if !ok {
		t.Fatal("Expected auth outputs to be map[string]any")
	}
	if authOutputs["token"] != "secret123" {
		t.Errorf("Expected auth token 'secret123', got %v", authOutputs["token"])
	}

	// Check flat access
	if all["token"] != "secret123" {
		t.Errorf("Expected flat token 'secret123', got %v", all["token"])
	}

	if all["name"] != "john_doe" {
		t.Errorf("Expected flat name 'john_doe', got %v", all["name"])
	}
}

func TestOutputsNoFlatAccessForStepIDConflict(t *testing.T) {
	outputs := NewOutputs()

	// Create a step with ID "token"
	outputs.Set("token", map[string]any{
		"value": "step_value",
	})

	// Try to create another step with output name "token"
	outputs.Set("auth", map[string]any{
		"token": "auth_token",
	})

	// Flat access to "token" should not work because "token" is a step ID
	_, exists := outputs.GetFlat("token")
	if exists {
		t.Error("Expected flat access to 'token' to not exist due to step ID conflict")
	}

	// Step-based access should still work
	tokenStep, exists := outputs.Get("token")
	if !exists {
		t.Fatal("Expected step 'token' to exist")
	}
	if tokenStep["value"] != "step_value" {
		t.Errorf("Expected step token value 'step_value', got %v", tokenStep["value"])
	}

	authStep, exists := outputs.Get("auth")
	if !exists {
		t.Fatal("Expected step 'auth' to exist")
	}
	if authStep["token"] != "auth_token" {
		t.Errorf("Expected auth token 'auth_token', got %v", authStep["token"])
	}
}
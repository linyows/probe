package probe

import (
	"reflect"
	"testing"
)

func TestOutputsFlatAccess(t *testing.T) {
	outputs := NewOutputs()

	// Test case 1: Normal case - no conflicts
	err := outputs.Set("auth", map[string]any{
		"token":   "secret123",
		"user_id": "user456",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

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
	err = outputs.Set("token", map[string]any{
		"value": "different_token",
	})
	if err == nil {
		t.Errorf("Expected conflict error but got none")
	}

	// Verify all data structure after conflict
	allData := outputs.GetAll()
	expected := map[string]any{
		"auth": map[string]any{
			"token":   "secret123",
			"user_id": "user456",
		},
		"token":   "secret123",
		"user_id": "user456",
		"value":   "different_token",
	}
	
	if !reflect.DeepEqual(allData, expected) {
		t.Errorf("GetAll() data mismatch.\nExpected: %+v\nGot: %+v", expected, allData)
	}

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
	err = outputs.Set("profile", map[string]any{
		"token": "profile_token", // Conflicts with existing "token"
		"name":  "john_doe",
	})
	if err != nil {
		t.Errorf("Unexpected error for profile step: %v", err)
	}

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

	err := outputs.Set("auth", map[string]any{
		"token": "secret123",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = outputs.Set("profile", map[string]any{
		"name": "john_doe",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

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
	err := outputs.Set("token", map[string]any{
		"value": "step_value",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Try to create another step with output name "token"
	err = outputs.Set("auth", map[string]any{
		"token": "auth_token",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

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

func TestOutputsConflictWarning(t *testing.T) {
	outputs := NewOutputs()

	// First, create a flat output 'foo'
	err := outputs.Set("step1", map[string]any{
		"foo": "flat_value",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Now try to create a step with ID 'foo' - this should return an error
	err = outputs.Set("foo", map[string]any{
		"bar": "step_value",
	})
	if err == nil {
		t.Error("Expected conflict error when step ID conflicts with flat output name")
	}

	// Verify the error message
	expectedMsg := "cannot create step-based outputs for 'foo' because flat output with same name exists"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Flat access should still work
	flatFoo, exists := outputs.GetFlat("foo")
	if !exists {
		t.Error("Expected flat access to 'foo' to still work")
	}
	if flatFoo != "flat_value" {
		t.Errorf("Expected flat foo 'flat_value', got %v", flatFoo)
	}

	// Step-based access should not work due to conflict
	_, exists = outputs.Get("foo")
	if exists {
		t.Error("Expected step-based access to 'foo' to fail due to conflict")
	}
}

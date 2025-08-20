package signaling

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWelcomePacketStructure(t *testing.T) {
	// Test that WelcomePacket includes protocol version
	packet := WelcomePacket{
		Type:            "welcome",
		ID:              "test-id",
		Secret:          "test-secret",
		ProtocolVersion: ProtocolVersion,
		Warnings:        []string{"test warning"},
	}

	// Marshal to JSON to verify the structure
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("Failed to marshal WelcomePacket: %v", err)
	}

	// Unmarshal back to verify structure
	var unmarshaled WelcomePacket
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal WelcomePacket: %v", err)
	}

	// Verify all fields are preserved
	if unmarshaled.Type != "welcome" {
		t.Errorf("Expected Type 'welcome', got %s", unmarshaled.Type)
	}
	if unmarshaled.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", unmarshaled.ID)
	}
	if unmarshaled.Secret != "test-secret" {
		t.Errorf("Expected Secret 'test-secret', got %s", unmarshaled.Secret)
	}
	if unmarshaled.ProtocolVersion != ProtocolVersion {
		t.Errorf("Expected ProtocolVersion %s, got %s", ProtocolVersion, unmarshaled.ProtocolVersion)
	}
	if len(unmarshaled.Warnings) != 1 || unmarshaled.Warnings[0] != "test warning" {
		t.Errorf("Expected Warnings ['test warning'], got %v", unmarshaled.Warnings)
	}
}

func TestWelcomePacketWithoutWarnings(t *testing.T) {
	// Test that warnings field is omitted when empty
	packet := WelcomePacket{
		Type:            "welcome",
		ID:              "test-id",
		Secret:          "test-secret",
		ProtocolVersion: ProtocolVersion,
	}

	// Marshal to JSON
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("Failed to marshal WelcomePacket: %v", err)
	}

	// Verify warnings field is not in JSON when empty
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, hasWarnings := jsonMap["warnings"]; hasWarnings {
		t.Errorf("Warnings field should be omitted when empty, but was present in JSON: %s", string(data))
	}

	// Verify protocol version is always present
	if _, hasProtocolVersion := jsonMap["protocolVersion"]; !hasProtocolVersion {
		t.Errorf("ProtocolVersion field should always be present in JSON: %s", string(data))
	}
}

func TestProtocolVersionConstant(t *testing.T) {
	// Verify protocol version is defined and not empty
	if ProtocolVersion == "" {
		t.Error("ProtocolVersion constant should not be empty")
	}

	// Verify it follows semver-like pattern (basic check)
	if len(ProtocolVersion) < 5 { // At minimum "1.0.0"
		t.Errorf("ProtocolVersion should be a proper version string, got: %s", ProtocolVersion)
	}
}

func TestWelcomePacketWarningsExample(t *testing.T) {
	// Test how warnings could be used in practice
	packet := WelcomePacket{
		Type:            "welcome",
		ID:              "test-id",
		Secret:          "test-secret",
		ProtocolVersion: ProtocolVersion,
		Warnings:        []string{"Example warning: This feature will be deprecated in version 2.0.0"},
	}

	// Marshal to JSON
	data, err := json.Marshal(packet)
	if err != nil {
		t.Fatalf("Failed to marshal WelcomePacket with warnings: %v", err)
	}

	// Verify the JSON contains the warning
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "Example warning") {
		t.Errorf("Expected JSON to contain warning message, got: %s", jsonStr)
	}

	if !strings.Contains(jsonStr, "protocolVersion") {
		t.Errorf("Expected JSON to contain protocolVersion, got: %s", jsonStr)
	}
}
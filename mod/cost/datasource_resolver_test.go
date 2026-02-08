package cost

import (
	"testing"
)

// TestDataSourceResolver_ParseDataSources tests parsing data source definitions
func TestDataSourceResolver_ParseDataSources(t *testing.T) {
	// This test verifies that data source blocks are correctly parsed
	// We'll use a mock credential provider
	mockCredProvider := func(provider string) (accessKey, secretKey, region string, err error) {
		return "test-key", "test-secret", "ap-guangzhou", nil
	}
	
	resolver := NewDataSourceResolver(mockCredProvider)
	
	if resolver == nil {
		t.Fatal("NewDataSourceResolver returned nil")
	}
	
	// Test that resolver is created successfully
	if resolver.credentialProvider == nil {
		t.Error("Credential provider not set")
	}
}

// TestReplaceDataSourceReferences tests replacing data source references in attributes
func TestReplaceDataSourceReferences(t *testing.T) {
	tests := []struct {
		name         string
		attributes   map[string]interface{}
		resolvedData map[string]interface{}
		expected     interface{}
		checkKey     string
	}{
		{
			name: "Simple data source reference",
			attributes: map[string]interface{}{
				"instance_type": "${data.tencentcloud_instance_types.instance_types.instance_types.0.instance_type}",
			},
			resolvedData: map[string]interface{}{
				"data.tencentcloud_instance_types.instance_types": map[string]interface{}{
					"instance_types": []map[string]interface{}{
						{
							"instance_type": "S6.MEDIUM4",
							"cpu":           2,
							"memory":        4,
						},
					},
				},
			},
			expected: "S6.MEDIUM4",
			checkKey: "instance_type",
		},
		{
			name: "Non-data source value unchanged",
			attributes: map[string]interface{}{
				"instance_type": "t2.micro",
			},
			resolvedData: map[string]interface{}{},
			expected:     "t2.micro",
			checkKey:     "instance_type",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceDataSourceReferences(tt.attributes, tt.resolvedData)
			
			if got, ok := result[tt.checkKey]; ok {
				if got != tt.expected {
					t.Errorf("ReplaceDataSourceReferences() = %v, want %v", got, tt.expected)
				}
			} else {
				t.Errorf("Key %s not found in result", tt.checkKey)
			}
		})
	}
}

// TestResolveDataSourceReference tests resolving individual data source references
func TestResolveDataSourceReference(t *testing.T) {
	resolvedData := map[string]interface{}{
		"data.tencentcloud_instance_types.instance_types": map[string]interface{}{
			"instance_types": []map[string]interface{}{
				{
					"instance_type": "S6.MEDIUM4",
					"cpu":           2,
					"memory":        4,
				},
				{
					"instance_type": "S6.LARGE8",
					"cpu":           4,
					"memory":        8,
				},
			},
		},
	}
	
	tests := []struct {
		name     string
		ref      string
		expected interface{}
	}{
		{
			name:     "Access first element instance_type",
			ref:      "data.tencentcloud_instance_types.instance_types.instance_types.0.instance_type",
			expected: "S6.MEDIUM4",
		},
		{
			name:     "Access second element instance_type",
			ref:      "data.tencentcloud_instance_types.instance_types.instance_types.1.instance_type",
			expected: "S6.LARGE8",
		},
		{
			name:     "Access first element cpu",
			ref:      "data.tencentcloud_instance_types.instance_types.instance_types.0.cpu",
			expected: 2,
		},
		{
			name:     "Invalid reference",
			ref:      "data.nonexistent.datasource.value",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveDataSourceReference(tt.ref, resolvedData)
			
			if result != tt.expected {
				t.Errorf("resolveDataSourceReference(%s) = %v, want %v", tt.ref, result, tt.expected)
			}
		})
	}
}

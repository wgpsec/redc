package cost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseTemplate_SimpleAlicloud tests parsing a simple Alibaba Cloud template
func TestParseTemplate_SimpleAlicloud(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-alicloud")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if resources == nil {
		t.Fatal("Expected resources, got nil")
	}

	// Check provider
	if resources.Provider != "alicloud" {
		t.Errorf("Expected provider 'alicloud', got '%s'", resources.Provider)
	}

	// Check that we have at least one resource
	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource, got none")
	}

	// Check the first resource
	res := resources.Resources[0]
	if res.Type != "alicloud_instance" {
		t.Errorf("Expected resource type 'alicloud_instance', got '%s'", res.Type)
	}

	if res.Name != "web" {
		t.Errorf("Expected resource name 'web', got '%s'", res.Name)
	}

	if res.Provider != "alicloud" {
		t.Errorf("Expected resource provider 'alicloud', got '%s'", res.Provider)
	}

	// Check that attributes were extracted
	if len(res.Attributes) == 0 {
		t.Error("Expected resource attributes, got none")
	}

	// Check for specific attributes
	if _, ok := res.Attributes["instance_type"]; !ok {
		t.Error("Expected 'instance_type' attribute")
	}

	if _, ok := res.Attributes["image_id"]; !ok {
		t.Error("Expected 'image_id' attribute")
	}
}

// TestParseTemplate_SimpleAWS tests parsing a simple AWS template
func TestParseTemplate_SimpleAWS(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-aws")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if resources == nil {
		t.Fatal("Expected resources, got nil")
	}

	// Check provider
	if resources.Provider != "aws" {
		t.Errorf("Expected provider 'aws', got '%s'", resources.Provider)
	}

	// Check that we have at least one resource
	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource, got none")
	}

	// Check the first resource
	res := resources.Resources[0]
	if res.Type != "aws_instance" {
		t.Errorf("Expected resource type 'aws_instance', got '%s'", res.Type)
	}

	if res.Name != "web" {
		t.Errorf("Expected resource name 'web', got '%s'", res.Name)
	}

	if res.Provider != "aws" {
		t.Errorf("Expected resource provider 'aws', got '%s'", res.Provider)
	}
}

// TestParseTemplate_NonExistentPath tests error handling for non-existent paths
func TestParseTemplate_NonExistentPath(t *testing.T) {
	_, err := ParseTemplate("/nonexistent/path", nil)
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}
}

// TestParseTemplate_EmptyDirectory tests error handling for empty directories
func TestParseTemplate_EmptyDirectory(t *testing.T) {
	// Create a temporary empty directory
	tmpDir := t.TempDir()
	
	_, err := ParseTemplate(tmpDir, nil)
	if err == nil {
		t.Error("Expected error for empty directory, got nil")
	}
}

// TestExtractProviderFromResourceType tests provider extraction from resource types
func TestExtractProviderFromResourceType(t *testing.T) {
	tests := []struct {
		resourceType string
		expected     string
	}{
		{"alicloud_instance", "alicloud"},
		{"aws_instance", "aws"},
		{"tencentcloud_instance", "tencentcloud"},
		{"google_compute_instance", "google"},
		{"azurerm_virtual_machine", "azurerm"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			result := extractProviderFromResourceType(tt.resourceType)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestParseTemplate_ResourceAttributes tests that resource attributes are extracted correctly
func TestParseTemplate_ResourceAttributes(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-alicloud")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that we extracted the image_id attribute (literal string)
	imageID, ok := res.Attributes["image_id"]
	if !ok {
		t.Error("Expected 'image_id' attribute to be extracted")
	} else if imageID != "ubuntu_20_04_x64_20G_alibase_20210420.vhd" {
		t.Errorf("Expected image_id to be 'ubuntu_20_04_x64_20G_alibase_20210420.vhd', got '%v'", imageID)
	}

	// Check that we have instance_type (variable reference)
	if _, ok := res.Attributes["instance_type"]; !ok {
		t.Error("Expected 'instance_type' attribute to be extracted")
	}

	// Check that we have system_disk_size (variable reference)
	if _, ok := res.Attributes["system_disk_size"]; !ok {
		t.Error("Expected 'system_disk_size' attribute to be extracted")
	}

	// Check that we have availability_zone (template expression)
	if _, ok := res.Attributes["availability_zone"]; !ok {
		t.Error("Expected 'availability_zone' attribute to be extracted")
	}

	// Check that we have tags (object)
	tags, ok := res.Attributes["tags"]
	if !ok {
		t.Error("Expected 'tags' attribute to be extracted")
	} else {
		tagsMap, ok := tags.(map[string]interface{})
		if !ok {
			t.Errorf("Expected tags to be a map, got %T", tags)
		} else if len(tagsMap) == 0 {
			t.Errorf("Expected tags map to have entries, got empty map. Tags: %+v", tags)
		}
	}
}

// TestParseTemplate_MultipleResources tests parsing templates with multiple resources
func TestParseTemplate_MultipleResources(t *testing.T) {
	// Create a temporary template with multiple resources
	tmpDir := t.TempDir()
	
	tfContent := `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}

resource "aws_instance" "db" {
  ami           = "ami-67890"
  instance_type = "t2.small"
}

resource "aws_s3_bucket" "storage" {
  bucket = "my-bucket"
}
`
	
	err := writeFile(filepath.Join(tmpDir, "main.tf"), tfContent)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	// Should have 3 resources
	if len(resources.Resources) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(resources.Resources))
	}

	// Check resource types
	expectedTypes := map[string]bool{
		"aws_instance": false,
		"aws_s3_bucket": false,
	}

	for _, res := range resources.Resources {
		if _, ok := expectedTypes[res.Type]; ok {
			expectedTypes[res.Type] = true
		}
	}

	for resType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected to find resource type '%s'", resType)
		}
	}
}

// Helper function to write test files
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// TestParseTemplate_ComprehensiveExtraction tests all aspects of resource extraction
// This test validates Requirements 1.1, 1.2, and 1.3
func TestParseTemplate_ComprehensiveExtraction(t *testing.T) {
	// Create a comprehensive test template
	tmpDir := t.TempDir()
	
	tfContent := `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

# Compute instance resource
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
  
  root_block_device {
    volume_size = 20
  }
  
  tags = {
    Name = "web-server"
    Environment = "production"
  }
}

# Storage resource
resource "aws_s3_bucket" "data" {
  bucket = "my-data-bucket"
  
  tags = {
    Purpose = "data-storage"
  }
}

# Networking resource
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
  
  tags = {
    Name = "main-vpc"
  }
}
`
	
	err := writeFile(filepath.Join(tmpDir, "main.tf"), tfContent)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Parse the template
	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	// Requirement 1.1: Extract all resource blocks
	if len(resources.Resources) != 3 {
		t.Errorf("Expected 3 resources, got %d", len(resources.Resources))
	}

	// Requirement 1.2: Identify resource types (compute, storage, networking)
	resourceTypes := make(map[string]bool)
	for _, res := range resources.Resources {
		resourceTypes[res.Type] = true
	}

	expectedTypes := []string{"aws_instance", "aws_s3_bucket", "aws_vpc"}
	for _, expectedType := range expectedTypes {
		if !resourceTypes[expectedType] {
			t.Errorf("Expected to find resource type '%s'", expectedType)
		}
	}

	// Requirement 1.3: Extract resource specifications
	for _, res := range resources.Resources {
		// Check that each resource has a type
		if res.Type == "" {
			t.Error("Resource type should not be empty")
		}

		// Check that each resource has a name
		if res.Name == "" {
			t.Error("Resource name should not be empty")
		}

		// Check that each resource has attributes
		if len(res.Attributes) == 0 {
			t.Errorf("Resource %s.%s should have attributes", res.Type, res.Name)
		}

		// Check that provider is correctly extracted
		if res.Provider != "aws" {
			t.Errorf("Expected provider 'aws', got '%s'", res.Provider)
		}

		// Verify specific attributes based on resource type
		switch res.Type {
		case "aws_instance":
			if _, ok := res.Attributes["ami"]; !ok {
				t.Error("aws_instance should have 'ami' attribute")
			}
			if _, ok := res.Attributes["instance_type"]; !ok {
				t.Error("aws_instance should have 'instance_type' attribute")
			}
			// Check that nested block (root_block_device) is extracted
			if _, ok := res.Attributes["root_block_device"]; !ok {
				t.Error("aws_instance should have 'root_block_device' attribute")
			}

		case "aws_s3_bucket":
			if _, ok := res.Attributes["bucket"]; !ok {
				t.Error("aws_s3_bucket should have 'bucket' attribute")
			}

		case "aws_vpc":
			if _, ok := res.Attributes["cidr_block"]; !ok {
				t.Error("aws_vpc should have 'cidr_block' attribute")
			}
		}

		// Check that tags are extracted (common across all resources)
		if _, ok := res.Attributes["tags"]; !ok {
			t.Errorf("Resource %s.%s should have 'tags' attribute", res.Type, res.Name)
		}
	}
}

// TestVariableResolution_WithDefaults tests variable resolution using default values from variables.tf
// This validates Requirement 1.4
func TestVariableResolution_WithDefaults(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-aws")
	
	// Parse without providing any user variables - should use defaults
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that instance_type was resolved to default value "t2.micro"
	instanceType, ok := res.Attributes["instance_type"]
	if !ok {
		t.Fatal("Expected 'instance_type' attribute")
	}
	if instanceType != "t2.micro" {
		t.Errorf("Expected instance_type to be 't2.micro' (default), got '%v'", instanceType)
	}

	// Check that count was resolved to default value 1
	if res.Count != 1 {
		t.Errorf("Expected count to be 1 (default), got %d", res.Count)
	}

	// Check that disk_size in root_block_device was resolved
	rootBlock, ok := res.Attributes["root_block_device"]
	if !ok {
		t.Fatal("Expected 'root_block_device' attribute")
	}
	rootBlockMap, ok := rootBlock.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected root_block_device to be a map, got %T", rootBlock)
	}
	diskSize, ok := rootBlockMap["volume_size"]
	if !ok {
		t.Fatal("Expected 'volume_size' in root_block_device")
	}
	if diskSize != 20 {
		t.Errorf("Expected volume_size to be 20 (default), got %v", diskSize)
	}
}

// TestVariableResolution_WithUserOverrides tests variable resolution with user-provided values
// This validates Requirement 1.4
func TestVariableResolution_WithUserOverrides(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-aws")
	
	// Provide user variables that override defaults
	userVars := map[string]string{
		"instance_type": "t2.large",
		"node_count":    "3",
		"disk_size":     "50",
	}
	
	resources, err := ParseTemplate(templatePath, userVars)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that instance_type was resolved to user-provided value
	instanceType, ok := res.Attributes["instance_type"]
	if !ok {
		t.Fatal("Expected 'instance_type' attribute")
	}
	if instanceType != "t2.large" {
		t.Errorf("Expected instance_type to be 't2.large' (user override), got '%v'", instanceType)
	}

	// Check that count was resolved to user-provided value
	if res.Count != 3 {
		t.Errorf("Expected count to be 3 (user override), got %d", res.Count)
	}

	// Check that disk_size was resolved to user-provided value
	rootBlock, ok := res.Attributes["root_block_device"]
	if !ok {
		t.Fatal("Expected 'root_block_device' attribute")
	}
	rootBlockMap, ok := rootBlock.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected root_block_device to be a map, got %T", rootBlock)
	}
	diskSize, ok := rootBlockMap["volume_size"]
	if !ok {
		t.Fatal("Expected 'volume_size' in root_block_device")
	}
	// Check if it's an int 50 (could be int or string)
	switch v := diskSize.(type) {
	case int:
		if v != 50 {
			t.Errorf("Expected volume_size to be 50 (user override), got %v", diskSize)
		}
	case string:
		if v != "50" {
			t.Errorf("Expected volume_size to be '50' (user override), got %v", diskSize)
		}
	default:
		t.Errorf("Expected volume_size to be int or string, got %T: %v", diskSize, diskSize)
	}
}

// TestVariableResolution_WithTfvars tests variable resolution with terraform.tfvars
// This validates Requirement 1.4
func TestVariableResolution_WithTfvars(t *testing.T) {
	// Create a temporary template with variables.tf and terraform.tfvars
	tmpDir := t.TempDir()
	
	variablesTf := `
variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}
`
	
	tfvarsContent := `
instance_type = "t2.medium"
region        = "us-west-2"
`
	
	mainTf := `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
  
  tags = {
    Region = var.region
  }
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "variables.tf"), variablesTf); err != nil {
		t.Fatalf("Failed to write variables.tf: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "terraform.tfvars"), tfvarsContent); err != nil {
		t.Fatalf("Failed to write terraform.tfvars: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), mainTf); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Parse without user variables - should use tfvars values
	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that instance_type was resolved from terraform.tfvars
	instanceType, ok := res.Attributes["instance_type"]
	if !ok {
		t.Fatal("Expected 'instance_type' attribute")
	}
	if instanceType != "t2.medium" {
		t.Errorf("Expected instance_type to be 't2.medium' (from tfvars), got '%v'", instanceType)
	}

	// Check that region in tags was resolved from terraform.tfvars
	tags, ok := res.Attributes["tags"]
	if !ok {
		t.Fatal("Expected 'tags' attribute")
	}
	tagsMap, ok := tags.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tags to be a map, got %T", tags)
	}
	region, ok := tagsMap["Region"]
	if !ok {
		t.Fatal("Expected 'Region' in tags")
	}
	if region != "us-west-2" {
		t.Errorf("Expected Region to be 'us-west-2' (from tfvars), got '%v'", region)
	}
}

// TestVariableResolution_PriorityOrder tests that user variables override tfvars which override defaults
// This validates Requirement 1.4
func TestVariableResolution_PriorityOrder(t *testing.T) {
	// Create a temporary template with all three sources
	tmpDir := t.TempDir()
	
	variablesTf := `
variable "instance_type" {
  description = "Instance type"
  type        = string
  default     = "t2.micro"
}
`
	
	tfvarsContent := `
instance_type = "t2.small"
`
	
	mainTf := `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.instance_type
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "variables.tf"), variablesTf); err != nil {
		t.Fatalf("Failed to write variables.tf: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "terraform.tfvars"), tfvarsContent); err != nil {
		t.Fatalf("Failed to write terraform.tfvars: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), mainTf); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Test 1: No user vars - should use tfvars value (t2.small)
	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}
	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}
	instanceType := resources.Resources[0].Attributes["instance_type"]
	if instanceType != "t2.small" {
		t.Errorf("Without user vars, expected 't2.small' (tfvars), got '%v'", instanceType)
	}

	// Test 2: With user vars - should use user value (t2.large)
	userVars := map[string]string{
		"instance_type": "t2.large",
	}
	resources, err = ParseTemplate(tmpDir, userVars)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}
	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}
	instanceType = resources.Resources[0].Attributes["instance_type"]
	if instanceType != "t2.large" {
		t.Errorf("With user vars, expected 't2.large' (user override), got '%v'", instanceType)
	}
}

// TestVariableResolution_TypeConversion tests that variable types are correctly converted
// This validates Requirement 1.4
func TestVariableResolution_TypeConversion(t *testing.T) {
	tmpDir := t.TempDir()
	
	variablesTf := `
variable "count_var" {
  type    = number
  default = 1
}

variable "enabled" {
  type    = bool
  default = false
}

variable "name" {
  type    = string
  default = "test"
}
`
	
	mainTf := `
resource "aws_instance" "web" {
  count         = var.count_var
  ami           = "ami-12345"
  instance_type = "t2.micro"
  
  tags = {
    Name    = var.name
    Enabled = var.enabled
  }
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "variables.tf"), variablesTf); err != nil {
		t.Fatalf("Failed to write variables.tf: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), mainTf); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Provide user variables as strings (as they would come from UI)
	userVars := map[string]string{
		"count_var": "5",
		"enabled":   "true",
		"name":      "production",
	}
	
	resources, err := ParseTemplate(tmpDir, userVars)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that count was converted to int
	if res.Count != 5 {
		t.Errorf("Expected count to be 5 (int), got %d", res.Count)
	}

	// Check that tags were resolved with correct types
	tags, ok := res.Attributes["tags"]
	if !ok {
		t.Fatal("Expected 'tags' attribute")
	}
	tagsMap, ok := tags.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tags to be a map, got %T", tags)
	}

	// Check string variable
	name, ok := tagsMap["Name"]
	if !ok {
		t.Fatal("Expected 'Name' in tags")
	}
	if name != "production" {
		t.Errorf("Expected Name to be 'production', got '%v'", name)
	}

	// Check bool variable
	enabled, ok := tagsMap["Enabled"]
	if !ok {
		t.Fatal("Expected 'Enabled' in tags")
	}
	// Check if it's a bool true (could be bool or string)
	switch v := enabled.(type) {
	case bool:
		if v != true {
			t.Errorf("Expected Enabled to be true, got %v", enabled)
		}
	case string:
		if v != "true" {
			t.Errorf("Expected Enabled to be 'true', got %v", enabled)
		}
	default:
		t.Errorf("Expected Enabled to be bool or string, got %T: %v", enabled, enabled)
	}
}

// TestVariableResolution_UndefinedVariable tests handling of undefined variables
// This validates Requirement 1.4
func TestVariableResolution_UndefinedVariable(t *testing.T) {
	tmpDir := t.TempDir()
	
	// No variables.tf file - variable is undefined
	mainTf := `
resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = var.undefined_var
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), mainTf); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Parse without providing the undefined variable
	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that undefined variable is kept as a placeholder
	instanceType, ok := res.Attributes["instance_type"]
	if !ok {
		t.Fatal("Expected 'instance_type' attribute")
	}
	
	// Should be a placeholder string like "${var.undefined_var}"
	instanceTypeStr, ok := instanceType.(string)
	if !ok {
		t.Errorf("Expected instance_type to be a string placeholder, got %T", instanceType)
	} else if !strings.Contains(instanceTypeStr, "undefined_var") {
		t.Errorf("Expected instance_type to contain 'undefined_var', got '%s'", instanceTypeStr)
	}
}

// TestVariableResolution_TemplateInterpolation tests variable resolution in template strings
// This validates Requirement 1.4
func TestVariableResolution_TemplateInterpolation(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "simple-aws")
	
	userVars := map[string]string{
		"node_count": "2",
	}
	
	resources, err := ParseTemplate(templatePath, userVars)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]

	// Check that tags with template interpolation are handled
	tags, ok := res.Attributes["tags"]
	if !ok {
		t.Fatal("Expected 'tags' attribute")
	}
	tagsMap, ok := tags.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tags to be a map, got %T", tags)
	}

	// The Name tag uses "web-${count.index}" which should be preserved
	name, ok := tagsMap["Name"]
	if !ok {
		t.Fatal("Expected 'Name' in tags")
	}
	
	// Should contain the template expression
	nameStr, ok := name.(string)
	if !ok {
		t.Errorf("Expected Name to be a string, got %T", name)
	} else if !strings.Contains(nameStr, "web-") {
		t.Errorf("Expected Name to contain 'web-', got '%s'", nameStr)
	}
}

// TestCountHandling_LiteralValue tests count with a literal integer value
// This validates Requirement 1.5
func TestCountHandling_LiteralValue(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfContent := `
resource "aws_instance" "web" {
  count         = 5
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), tfContent); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]
	if res.Count != 5 {
		t.Errorf("Expected count to be 5, got %d", res.Count)
	}
}

// TestCountHandling_VariableReference tests count with a variable reference
// This validates Requirement 1.5
func TestCountHandling_VariableReference(t *testing.T) {
	tmpDir := t.TempDir()
	
	variablesTf := `
variable "instance_count" {
  type    = number
  default = 3
}
`
	
	mainTf := `
resource "aws_instance" "web" {
  count         = var.instance_count
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "variables.tf"), variablesTf); err != nil {
		t.Fatalf("Failed to write variables.tf: %v", err)
	}
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), mainTf); err != nil {
		t.Fatalf("Failed to write main.tf: %v", err)
	}

	// Test with default value
	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]
	if res.Count != 3 {
		t.Errorf("Expected count to be 3 (default), got %d", res.Count)
	}

	// Test with user-provided value
	userVars := map[string]string{
		"instance_count": "7",
	}
	resources, err = ParseTemplate(tmpDir, userVars)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res = resources.Resources[0]
	if res.Count != 7 {
		t.Errorf("Expected count to be 7 (user override), got %d", res.Count)
	}
}

// TestForEachHandling_MapLiteral tests for_each with a literal map
// This validates Requirement 1.5
func TestForEachHandling_MapLiteral(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "for-each-test")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	// Find the servers resource (first resource with for_each map)
	var serversRes *ResourceSpec
	for i := range resources.Resources {
		if resources.Resources[i].Name == "servers" {
			serversRes = &resources.Resources[i]
			break
		}
	}

	if serversRes == nil {
		t.Fatal("Expected to find 'servers' resource")
	}

	// The for_each map has 3 entries: web, api, db
	if serversRes.Count != 3 {
		t.Errorf("Expected count to be 3 (from for_each map), got %d", serversRes.Count)
	}
}

// TestForEachHandling_VariableMap tests for_each with a variable map
// This validates Requirement 1.5
func TestForEachHandling_VariableMap(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "for-each-test")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	// Find the groups resource (uses variable for for_each)
	var groupsRes *ResourceSpec
	for i := range resources.Resources {
		if resources.Resources[i].Name == "groups" {
			groupsRes = &resources.Resources[i]
			break
		}
	}

	if groupsRes == nil {
		t.Fatal("Expected to find 'groups' resource")
	}

	// The variable security_groups has 2 entries by default
	if groupsRes.Count != 2 {
		t.Errorf("Expected count to be 2 (from variable map), got %d", groupsRes.Count)
	}
}

// TestForEachHandling_SetExpression tests for_each with a set expression
// This validates Requirement 1.5
func TestForEachHandling_SetExpression(t *testing.T) {
	templatePath := filepath.Join("testdata", "templates", "for-each-test")
	
	resources, err := ParseTemplate(templatePath, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	// Find the buckets resource (uses toset() function)
	var bucketsRes *ResourceSpec
	for i := range resources.Resources {
		if resources.Resources[i].Name == "buckets" {
			bucketsRes = &resources.Resources[i]
			break
		}
	}

	if bucketsRes == nil {
		t.Fatal("Expected to find 'buckets' resource")
	}

	// The toset() function is not evaluated at parse time, so count should be 1 (conservative estimate)
	// In a real scenario, we'd need runtime evaluation
	if bucketsRes.Count != 1 {
		t.Errorf("Expected count to be 1 (conservative estimate for function), got %d", bucketsRes.Count)
	}
}

// TestCountAndForEach_DefaultBehavior tests that resources without count or for_each have count=1
// This validates Requirement 1.5
func TestCountAndForEach_DefaultBehavior(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfContent := `
resource "aws_instance" "single" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), tfContent); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]
	if res.Count != 1 {
		t.Errorf("Expected default count to be 1, got %d", res.Count)
	}
}

// TestCountHandling_ZeroValue tests count with zero value
// This validates Requirement 1.5
func TestCountHandling_ZeroValue(t *testing.T) {
	tmpDir := t.TempDir()
	
	tfContent := `
resource "aws_instance" "conditional" {
  count         = 0
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	
	if err := writeFile(filepath.Join(tmpDir, "main.tf"), tfContent); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	resources, err := ParseTemplate(tmpDir, nil)
	if err != nil {
		t.Fatalf("ParseTemplate failed: %v", err)
	}

	if len(resources.Resources) == 0 {
		t.Fatal("Expected at least one resource")
	}

	res := resources.Resources[0]
	if res.Count != 0 {
		t.Errorf("Expected count to be 0, got %d", res.Count)
	}
}

// TestCalculateForEachCount_MapInput tests the calculateForEachCount function with map input
func TestCalculateForEachCount_MapInput(t *testing.T) {
	testMap := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	count := calculateForEachCount(testMap)
	if count != 3 {
		t.Errorf("Expected count to be 3 for map with 3 keys, got %d", count)
	}
}

// TestCalculateForEachCount_ListInput tests the calculateForEachCount function with list input
func TestCalculateForEachCount_ListInput(t *testing.T) {
	testList := []interface{}{"item1", "item2", "item3", "item4"}
	
	count := calculateForEachCount(testList)
	if count != 4 {
		t.Errorf("Expected count to be 4 for list with 4 items, got %d", count)
	}
}

// TestCalculateForEachCount_StringInput tests the calculateForEachCount function with string input
func TestCalculateForEachCount_StringInput(t *testing.T) {
	testString := "${var.some_set}"
	
	count := calculateForEachCount(testString)
	if count != 1 {
		t.Errorf("Expected count to be 1 (conservative estimate) for string, got %d", count)
	}
}

// TestCalculateForEachCount_EmptyMap tests the calculateForEachCount function with empty map
func TestCalculateForEachCount_EmptyMap(t *testing.T) {
	testMap := map[string]interface{}{}
	
	count := calculateForEachCount(testMap)
	if count != 0 {
		t.Errorf("Expected count to be 0 for empty map, got %d", count)
	}
}

// TestCalculateForEachCount_EmptyList tests the calculateForEachCount function with empty list
func TestCalculateForEachCount_EmptyList(t *testing.T) {
	testList := []interface{}{}
	
	count := calculateForEachCount(testList)
	if count != 0 {
		t.Errorf("Expected count to be 0 for empty list, got %d", count)
	}
}

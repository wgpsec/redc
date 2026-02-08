package cost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	_ "github.com/hashicorp/terraform-config-inspect/tfconfig"
	_ "github.com/leanovate/gopter"
)

// Package cost provides cost estimation functionality for Terraform templates.
// It parses templates, retrieves pricing data, and calculates cost estimates.

// Global credential provider for data source resolution
var globalCredentialProvider CredentialProvider

// SetGlobalCredentialProvider sets the global credential provider for data source resolution
func SetGlobalCredentialProvider(provider CredentialProvider) {
	globalCredentialProvider = provider
}

// TemplateResources represents all resources in a Terraform template
type TemplateResources struct {
	Provider  string         `json:"provider"`  // Primary provider (alicloud, tencentcloud, aws)
	Region    string         `json:"region"`    // Deployment region
	Resources []ResourceSpec `json:"resources"` // List of resources to be created
}

// ResourceSpec represents a single resource specification
type ResourceSpec struct {
	Type       string                 `json:"type"`       // e.g., "alicloud_instance", "aws_instance"
	Name       string                 `json:"name"`       // Resource name in template
	Count      int                    `json:"count"`      // Number of instances (from count or for_each)
	Attributes map[string]interface{} `json:"attributes"` // Resource attributes
	Provider   string                 `json:"provider"`   // Provider name
	Region     string                 `json:"region"`     // Resource region
}
// VariableDefinition represents a variable definition from variables.tf
type VariableDefinition struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default"`
}

// VariableValues holds resolved variable values from multiple sources
type VariableValues map[string]interface{}


// ParseTemplate parses a Terraform template and returns resource specifications
// ParseTemplate parses a Terraform template and returns resource specifications
func ParseTemplate(templatePath string, variables map[string]string) (*TemplateResources, error) {
	// Find all .tf files in the template directory
	tfFiles, err := findTerraformFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find terraform files: %w", err)
	}

	if len(tfFiles) == 0 {
		return nil, fmt.Errorf("no .tf files found in %s", templatePath)
	}

	// Parse all .tf files
	parser := hclparse.NewParser()
	var allFiles []*hcl.File
	var diags hcl.Diagnostics

	for _, tfFile := range tfFiles {
		file, fileDiags := parser.ParseHCLFile(tfFile)
		diags = append(diags, fileDiags...)
		if file != nil {
			allFiles = append(allFiles, file)
		}
	}

	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parsing errors: %s", diags.Error())
	}

	// Resolve variables from multiple sources
	resolvedVars, err := resolveVariables(templatePath, allFiles, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve variables: %w", err)
	}

	// Resolve data sources if credential provider is available
	var resolvedData map[string]interface{}
	if globalCredentialProvider != nil {
		resolver := NewDataSourceResolver(globalCredentialProvider)
		resolvedData, err = resolver.ResolveDataSources(allFiles, resolvedVars)
		if err != nil {
			// Log error but continue - data source resolution is optional
			// In production, we'd use proper logging here
		}
	}

	// Extract resources from parsed files
	resources := &TemplateResources{
		Resources: []ResourceSpec{},
	}

	for _, file := range allFiles {
		if err := extractResourcesFromFile(file, resources, resolvedVars); err != nil {
			return nil, fmt.Errorf("failed to extract resources: %w", err)
		}
	}

	// Replace data source references in resource attributes
	if resolvedData != nil && len(resolvedData) > 0 {
		for i := range resources.Resources {
			resources.Resources[i].Attributes = ReplaceDataSourceReferences(
				resources.Resources[i].Attributes,
				resolvedData,
			)
		}
	}

	// Determine primary provider from resources
	if len(resources.Resources) > 0 {
		resources.Provider = extractProviderFromResourceType(resources.Resources[0].Type)
	}

	return resources, nil
}

// findTerraformFiles finds all .tf files in the given directory
func findTerraformFiles(dir string) ([]string, error) {
	var tfFiles []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".tf") {
			tfFiles = append(tfFiles, filepath.Join(dir, entry.Name()))
		}
	}

	return tfFiles, nil
}
// resolveVariables resolves variables from multiple sources:
// 1. Default values from variables.tf
// 2. Values from terraform.tfvars (if exists)
// 3. User-provided values (highest priority)
func resolveVariables(templatePath string, allFiles []*hcl.File, userVars map[string]string) (VariableValues, error) {
	// Step 1: Parse variable definitions from variables.tf to get defaults
	varDefs := parseVariableDefinitions(allFiles)

	// Start with defaults from variable definitions
	resolved := make(VariableValues)
	for name, varDef := range varDefs {
		if varDef.Default != nil {
			resolved[name] = varDef.Default
		}
	}

	// Step 2: Override with values from terraform.tfvars if it exists
	tfvarsPath := filepath.Join(templatePath, "terraform.tfvars")
	if _, err := os.Stat(tfvarsPath); err == nil {
		tfvarsValues, err := parseTerraformTfvars(tfvarsPath)
		if err != nil {
			// Log but don't fail - tfvars is optional
			// In production, we'd use proper logging here
		} else {
			for name, value := range tfvarsValues {
				resolved[name] = value
			}
		}
	}

	// Step 3: Override with user-provided values (highest priority)
	for name, value := range userVars {
		// Convert string values to appropriate types based on variable definition
		if varDef, ok := varDefs[name]; ok {
			converted, err := convertVariableValue(value, varDef.Type)
			if err == nil {
				resolved[name] = converted
			} else {
				// If conversion fails, use string value
				resolved[name] = value
			}
		} else {
			// Variable not defined, use string value
			resolved[name] = value
		}
	}

	return resolved, nil
}

// parseVariableDefinitions extracts variable definitions from parsed HCL files
func parseVariableDefinitions(allFiles []*hcl.File) map[string]*VariableDefinition {
	varDefs := make(map[string]*VariableDefinition)

	for _, file := range allFiles {
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		for _, block := range body.Blocks {
			if block.Type == "variable" && len(block.Labels) > 0 {
				varName := block.Labels[0]
				varDef := &VariableDefinition{
					Name: varName,
				}

				// Extract variable attributes
				for attrName, attr := range block.Body.Attributes {
					value, err := extractAttributeValue(attr.Expr)
					if err != nil {
						continue
					}

					switch attrName {
					case "type":
						if typeStr, ok := value.(string); ok {
							varDef.Type = typeStr
						}
					case "description":
						if desc, ok := value.(string); ok {
							varDef.Description = desc
						}
					case "default":
						varDef.Default = value
					}
				}

				varDefs[varName] = varDef
			}
		}
	}

	return varDefs
}

// parseTerraformTfvars parses a terraform.tfvars file and returns variable values
func parseTerraformTfvars(tfvarsPath string) (map[string]interface{}, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(tfvarsPath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse terraform.tfvars: %s", diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type in terraform.tfvars: %T", file.Body)
	}

	values := make(map[string]interface{})
	for name, attr := range body.Attributes {
		value, err := extractAttributeValue(attr.Expr)
		if err != nil {
			continue
		}
		values[name] = value
	}

	return values, nil
}

// convertVariableValue converts a string value to the appropriate type
func convertVariableValue(value string, varType string) (interface{}, error) {
	switch varType {
	case "string":
		return value, nil
	case "number":
		// Try to parse as int first, then float
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal, nil
		}
		var floatVal float64
		if _, err := fmt.Sscanf(value, "%f", &floatVal); err == nil {
			return floatVal, nil
		}
		return nil, fmt.Errorf("cannot convert '%s' to number", value)
	case "bool":
		if value == "true" {
			return true, nil
		} else if value == "false" {
			return false, nil
		}
		return nil, fmt.Errorf("cannot convert '%s' to bool", value)
	default:
		// For complex types (list, map, object), return as string
		return value, nil
	}
}

// resolveVariableReference resolves a variable reference like "var.region" to its actual value
func resolveVariableReference(varRef string, resolvedVars VariableValues) interface{} {
	// Remove "var." prefix if present
	varName := strings.TrimPrefix(varRef, "var.")

	// Look up the variable value
	if value, ok := resolvedVars[varName]; ok {
		return value
	}

	// If not found, return the original reference as a placeholder
	return fmt.Sprintf("${%s}", varRef)
}

// extractResourcesFromFile extracts resource blocks from a parsed HCL file
// extractResourcesFromFile extracts resource blocks from a parsed HCL file
func extractResourcesFromFile(file *hcl.File, resources *TemplateResources, resolvedVars VariableValues) error {
	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return fmt.Errorf("unexpected body type: %T", file.Body)
	}

	for _, block := range body.Blocks {
		if block.Type == "resource" {
			if len(block.Labels) < 2 {
				continue // Invalid resource block
			}

			resourceType := block.Labels[0]
			resourceName := block.Labels[1]

			// Extract attributes from resource block
			attributes := make(map[string]interface{})
			count := 1 // Default count

			for name, attr := range block.Body.Attributes {
				value, err := extractAttributeValueWithVars(attr.Expr, resolvedVars)
				if err != nil {
					// Skip attributes we can't extract
					continue
				}
				attributes[name] = value

				// Check for count attribute
				if name == "count" {
					// Try to convert to int if it's not already
					switch v := value.(type) {
					case int:
						count = v
					case float64:
						count = int(v)
					case string:
						// Try to parse string as int
						var intVal int
						if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
							count = intVal
						}
					}
				}

				// Check for for_each attribute
				if name == "for_each" {
					// Calculate count from for_each
					count = calculateForEachCount(value)
				}
			}

			// Extract nested blocks (like root_block_device, ebs_block_device, etc.)
			for _, nestedBlock := range block.Body.Blocks {
				blockName := nestedBlock.Type
				blockAttrs := make(map[string]interface{})

				for name, attr := range nestedBlock.Body.Attributes {
					value, err := extractAttributeValueWithVars(attr.Expr, resolvedVars)
					if err != nil {
						continue
					}
					blockAttrs[name] = value
				}

				// Store nested block as an attribute
				if len(blockAttrs) > 0 {
					attributes[blockName] = blockAttrs
				}
			}

			// Create resource spec
			spec := ResourceSpec{
				Type:       resourceType,
				Name:       resourceName,
				Count:      count,
				Attributes: attributes,
				Provider:   extractProviderFromResourceType(resourceType),
			}

			resources.Resources = append(resources.Resources, spec)
		}
	}

	return nil
}

// extractAttributeValue extracts a value from an HCL expression
func extractAttributeValue(expr hclsyntax.Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		// Direct literal value (string, number, bool)
		val := e.Val
		switch val.Type().FriendlyName() {
		case "string":
			return val.AsString(), nil
		case "number":
			f, _ := val.AsBigFloat().Float64()
			// Check if it's an integer
			if f == float64(int(f)) {
				return int(f), nil
			}
			return f, nil
		case "bool":
			return val.True(), nil
		default:
			return val.AsString(), nil
		}

	case *hclsyntax.TemplateExpr:
		// Template expression (string interpolation)
		// Check if it's a simple string literal (no interpolation)
		if len(e.Parts) == 1 {
			if lit, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				return lit.Val.AsString(), nil
			}
		}
		// For actual interpolation, return a placeholder string
		return fmt.Sprintf("${template}"), nil

	case *hclsyntax.ScopeTraversalExpr:
		// Variable reference (e.g., var.region)
		// Return the traversal as a string for now
		return fmt.Sprintf("${%s}", traversalToString(e.Traversal)), nil

	case *hclsyntax.FunctionCallExpr:
		// Function call (e.g., format(), etc.)
		return fmt.Sprintf("${function:%s}", e.Name), nil

	case *hclsyntax.ObjectConsExpr:
		// Object/map construction
		obj := make(map[string]interface{})
		for _, item := range e.Items {
			keyExpr := item.KeyExpr
			valExpr := item.ValueExpr

			// Extract key
			var key string
			switch k := keyExpr.(type) {
			case *hclsyntax.ObjectConsKeyExpr:
				// Wrapped key expression (common in HCL2)
				if lit, ok := k.Wrapped.(*hclsyntax.LiteralValueExpr); ok {
					key = lit.Val.AsString()
				} else if traversal, ok := k.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok {
					key = traversalToString(traversal.Traversal)
				} else {
					continue
				}
			case *hclsyntax.LiteralValueExpr:
				key = k.Val.AsString()
			case *hclsyntax.ScopeTraversalExpr:
				key = traversalToString(k.Traversal)
			default:
				continue
			}

			// Extract value - don't skip on error, use placeholder
			val, err := extractAttributeValue(valExpr)
			if err != nil {
				// Use a placeholder for unsupported expressions
				obj[key] = fmt.Sprintf("${unsupported}")
				continue
			}
			obj[key] = val
		}
		return obj, nil

	case *hclsyntax.TupleConsExpr:
		// Array/list construction
		var arr []interface{}
		for _, elemExpr := range e.Exprs {
			val, err := extractAttributeValue(elemExpr)
			if err != nil {
				continue
			}
			arr = append(arr, val)
		}
		return arr, nil

	default:
		// Unknown expression type
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}
// extractAttributeValueWithVars extracts a value from an HCL expression and resolves variable references
func extractAttributeValueWithVars(expr hclsyntax.Expression, resolvedVars VariableValues) (interface{}, error) {
	switch e := expr.(type) {
	case *hclsyntax.LiteralValueExpr:
		// Direct literal value (string, number, bool)
		val := e.Val
		switch val.Type().FriendlyName() {
		case "string":
			return val.AsString(), nil
		case "number":
			f, _ := val.AsBigFloat().Float64()
			// Check if it's an integer
			if f == float64(int(f)) {
				return int(f), nil
			}
			return f, nil
		case "bool":
			return val.True(), nil
		default:
			return val.AsString(), nil
		}

	case *hclsyntax.TemplateExpr:
		// Template expression (string interpolation)
		// Check if it's a simple string literal (no interpolation)
		if len(e.Parts) == 1 {
			if lit, ok := e.Parts[0].(*hclsyntax.LiteralValueExpr); ok {
				return lit.Val.AsString(), nil
			}
		}
		// For actual interpolation, try to resolve variables
		result := ""
		for _, part := range e.Parts {
			switch p := part.(type) {
			case *hclsyntax.LiteralValueExpr:
				result += p.Val.AsString()
			case *hclsyntax.ScopeTraversalExpr:
				// Try to resolve variable reference
				varRef := traversalToString(p.Traversal)
				if resolved := resolveVariableReference(varRef, resolvedVars); resolved != nil {
					result += fmt.Sprintf("%v", resolved)
				} else {
					result += fmt.Sprintf("${%s}", varRef)
				}
			default:
				// For other expression types, use placeholder
				result += "${...}"
			}
		}
		return result, nil

	case *hclsyntax.ScopeTraversalExpr:
		// Variable reference (e.g., var.region)
		varRef := traversalToString(e.Traversal)
		if resolved := resolveVariableReference(varRef, resolvedVars); resolved != nil {
			return resolved, nil
		}
		// If not resolved, return the reference as a string
		return fmt.Sprintf("${%s}", varRef), nil

	case *hclsyntax.FunctionCallExpr:
		// Function call (e.g., format(), etc.)
		return fmt.Sprintf("${function:%s}", e.Name), nil

	case *hclsyntax.ObjectConsExpr:
		// Object/map construction
		obj := make(map[string]interface{})
		for _, item := range e.Items {
			keyExpr := item.KeyExpr
			valExpr := item.ValueExpr

			// Extract key
			var key string
			switch k := keyExpr.(type) {
			case *hclsyntax.ObjectConsKeyExpr:
				// Wrapped key expression (common in HCL2)
				if lit, ok := k.Wrapped.(*hclsyntax.LiteralValueExpr); ok {
					key = lit.Val.AsString()
				} else if traversal, ok := k.Wrapped.(*hclsyntax.ScopeTraversalExpr); ok {
					key = traversalToString(traversal.Traversal)
				} else {
					continue
				}
			case *hclsyntax.LiteralValueExpr:
				key = k.Val.AsString()
			case *hclsyntax.ScopeTraversalExpr:
				key = traversalToString(k.Traversal)
			default:
				continue
			}

			// Extract value with variable resolution
			val, err := extractAttributeValueWithVars(valExpr, resolvedVars)
			if err != nil {
				// Use a placeholder for unsupported expressions
				obj[key] = fmt.Sprintf("${unsupported}")
				continue
			}
			obj[key] = val
		}
		return obj, nil

	case *hclsyntax.TupleConsExpr:
		// Array/list construction
		var arr []interface{}
		for _, elemExpr := range e.Exprs {
			val, err := extractAttributeValueWithVars(elemExpr, resolvedVars)
			if err != nil {
				continue
			}
			arr = append(arr, val)
		}
		return arr, nil

	default:
		// Unknown expression type
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// traversalToString converts an HCL traversal to a string representation
func traversalToString(traversal hcl.Traversal) string {
	var parts []string
	for _, part := range traversal {
		switch p := part.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, p.Name)
		case hcl.TraverseAttr:
			parts = append(parts, p.Name)
		case hcl.TraverseIndex:
			// Handle index access - extract the actual value from cty.Value
			key := p.Key
			switch key.Type().FriendlyName() {
			case "number":
				// Numeric index
				f, _ := key.AsBigFloat().Float64()
				parts = append(parts, fmt.Sprintf("%d", int(f)))
			case "string":
				// String key
				parts = append(parts, fmt.Sprintf("[%s]", key.AsString()))
			default:
				// Fallback
				parts = append(parts, fmt.Sprintf("[%v]", key))
			}
		}
	}
	return strings.Join(parts, ".")
}

// extractProviderFromResourceType extracts the provider name from a resource type
// e.g., "alicloud_instance" -> "alicloud", "aws_instance" -> "aws"
func extractProviderFromResourceType(resourceType string) string {
	parts := strings.SplitN(resourceType, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// calculateForEachCount calculates the number of resource instances from a for_each value
// for_each can be a set, map, or list
func calculateForEachCount(forEachValue interface{}) int {
	switch v := forEachValue.(type) {
	case map[string]interface{}:
		// for_each with a map - count is the number of keys
		return len(v)
	case []interface{}:
		// for_each with a list/set - count is the number of elements
		return len(v)
	case string:
		// for_each with a variable reference or expression
		// We can't determine the count at parse time, so return 1 as a conservative estimate
		// In a real scenario, this would need runtime evaluation
		return 1
	default:
		// Unknown type, return 1 as default
		return 1
	}
}

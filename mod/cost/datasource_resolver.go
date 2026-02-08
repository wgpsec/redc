package cost

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

// DataSourceResolver resolves Terraform data sources by querying cloud provider APIs
type DataSourceResolver struct {
	credentialProvider CredentialProvider
}

// NewDataSourceResolver creates a new data source resolver
func NewDataSourceResolver(credentialProvider CredentialProvider) *DataSourceResolver {
	return &DataSourceResolver{
		credentialProvider: credentialProvider,
	}
}

// DataSourceDefinition represents a parsed data source block
type DataSourceDefinition struct {
	Type       string                 // e.g., "tencentcloud_instance_types"
	Name       string                 // e.g., "instance_types"
	Attributes map[string]interface{} // Data source attributes
	Filters    []DataSourceFilter     // Filter blocks
}

// DataSourceFilter represents a filter block in a data source
type DataSourceFilter struct {
	Name   string
	Values []string
}

// ResolveDataSources parses and resolves all data sources in the template
func (r *DataSourceResolver) ResolveDataSources(allFiles []*hcl.File, resolvedVars VariableValues) (map[string]interface{}, error) {
	// Parse data source definitions
	dataSources := r.parseDataSources(allFiles, resolvedVars)
	
	// Resolve each data source by querying the provider API
	resolvedData := make(map[string]interface{})
	
	for key, ds := range dataSources {
		result, err := r.resolveDataSource(ds)
		if err != nil {
			// Log error but continue with other data sources
			continue
		}
		resolvedData[key] = result
	}
	
	return resolvedData, nil
}

// parseDataSources extracts data source definitions from HCL files
func (r *DataSourceResolver) parseDataSources(allFiles []*hcl.File, resolvedVars VariableValues) map[string]*DataSourceDefinition {
	dataSources := make(map[string]*DataSourceDefinition)
	
	for _, file := range allFiles {
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}
		
		for _, block := range body.Blocks {
			if block.Type == "data" && len(block.Labels) >= 2 {
				dsType := block.Labels[0]
				dsName := block.Labels[1]
				
				ds := &DataSourceDefinition{
					Type:       dsType,
					Name:       dsName,
					Attributes: make(map[string]interface{}),
					Filters:    []DataSourceFilter{},
				}
				
				// Extract attributes
				for attrName, attr := range block.Body.Attributes {
					value, err := extractAttributeValueWithVars(attr.Expr, resolvedVars)
					if err != nil {
						continue
					}
					ds.Attributes[attrName] = value
				}
				
				// Extract filter blocks
				for _, nestedBlock := range block.Body.Blocks {
					if nestedBlock.Type == "filter" {
						filter := DataSourceFilter{}
						
						for attrName, attr := range nestedBlock.Body.Attributes {
							value, err := extractAttributeValueWithVars(attr.Expr, resolvedVars)
							if err != nil {
								continue
							}
							
							if attrName == "name" {
								if nameStr, ok := value.(string); ok {
									filter.Name = nameStr
								}
							} else if attrName == "values" {
								// Convert values to string array
								if valArray, ok := value.([]interface{}); ok {
									for _, v := range valArray {
										if vStr, ok := v.(string); ok {
											filter.Values = append(filter.Values, vStr)
										}
									}
								}
							}
						}
						
						if filter.Name != "" {
							ds.Filters = append(ds.Filters, filter)
						}
					}
				}
				
				// Store with key "data.type.name"
				key := fmt.Sprintf("data.%s.%s", dsType, dsName)
				dataSources[key] = ds
			}
		}
	}
	
	return dataSources
}

// resolveDataSource queries the cloud provider API to resolve a data source
func (r *DataSourceResolver) resolveDataSource(ds *DataSourceDefinition) (interface{}, error) {
	switch ds.Type {
	case "tencentcloud_instance_types":
		return r.resolveTencentCloudInstanceTypes(ds)
	// Add more data source types as needed
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", ds.Type)
	}
}

// resolveTencentCloudInstanceTypes resolves tencentcloud_instance_types data source
func (r *DataSourceResolver) resolveTencentCloudInstanceTypes(ds *DataSourceDefinition) (interface{}, error) {
	// Get Tencent Cloud credentials
	accessKey, secretKey, region, err := r.credentialProvider("tencentcloud")
	if err != nil {
		return nil, fmt.Errorf("failed to get Tencent Cloud credentials: %w", err)
	}
	
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("missing Tencent Cloud credentials")
	}
	
	// Use default region if not provided
	if region == "" {
		region = "ap-guangzhou" // Default region
	}
	
	// Create Tencent Cloud client
	credential := common.NewCredential(accessKey, secretKey)
	cpf := profile.NewClientProfile()
	client, err := cvm.NewClient(credential, region, cpf)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tencent Cloud client: %w", err)
	}
	
	// Build DescribeInstanceTypeConfigs request
	request := cvm.NewDescribeInstanceTypeConfigsRequest()
	
	// Apply filters
	for _, filter := range ds.Filters {
		if filter.Name == "instance-family" && len(filter.Values) > 0 {
			// Filter by instance family
			families := make([]*string, len(filter.Values))
			for i, v := range filter.Values {
				families[i] = common.StringPtr(v)
			}
			request.Filters = append(request.Filters, &cvm.Filter{
				Name:   common.StringPtr("instance-family"),
				Values: families,
			})
		}
	}
	
	// Apply CPU and memory filters from attributes
	// Note: Tencent Cloud API doesn't support direct CPU/memory filtering
	// We'll filter the results after getting the response
	
	// Call API
	response, err := client.DescribeInstanceTypeConfigs(request)
	if err != nil {
		return nil, fmt.Errorf("failed to query Tencent Cloud instance types: %w", err)
	}
	
	if response.Response == nil || len(response.Response.InstanceTypeConfigSet) == 0 {
		return nil, fmt.Errorf("no instance types found matching criteria")
	}
	
	// Filter results by CPU and memory if specified
	var filteredResults []*cvm.InstanceTypeConfig
	cpuFilter := -1
	memFilter := -1
	
	if cpuCount, ok := ds.Attributes["cpu_core_count"]; ok {
		if cpu, ok := cpuCount.(int); ok {
			cpuFilter = cpu
		}
	}
	
	if memSize, ok := ds.Attributes["memory_size"]; ok {
		if mem, ok := memSize.(int); ok {
			memFilter = mem
		}
	}
	
	for _, it := range response.Response.InstanceTypeConfigSet {
		// Apply CPU filter
		if cpuFilter > 0 && it.CPU != nil && int(*it.CPU) != cpuFilter {
			continue
		}
		
		// Apply memory filter
		if memFilter > 0 && it.Memory != nil && int(*it.Memory) != memFilter {
			continue
		}
		
		filteredResults = append(filteredResults, it)
	}
	
	if len(filteredResults) == 0 {
		return nil, fmt.Errorf("no instance types found matching CPU=%d, Memory=%d", cpuFilter, memFilter)
	}
	
	// Convert response to a map structure that matches Terraform data source output
	instanceTypes := make([]map[string]interface{}, 0)
	for _, it := range filteredResults {
		instanceType := make(map[string]interface{})
		
		if it.InstanceType != nil {
			instanceType["instance_type"] = *it.InstanceType
		}
		if it.CPU != nil {
			instanceType["cpu"] = *it.CPU
		}
		if it.Memory != nil {
			instanceType["memory"] = *it.Memory
		}
		if it.InstanceFamily != nil {
			instanceType["family"] = *it.InstanceFamily
		}
		
		instanceTypes = append(instanceTypes, instanceType)
	}
	
	// Return result in the format: { "instance_types": [...] }
	result := map[string]interface{}{
		"instance_types": instanceTypes,
	}
	
	return result, nil
}

// ReplaceDataSourceReferences replaces data source references in resource attributes
func ReplaceDataSourceReferences(attributes map[string]interface{}, resolvedData map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range attributes {
		result[key] = replaceDataSourceValue(value, resolvedData)
	}
	
	return result
}

// replaceDataSourceValue recursively replaces data source references in a value
func replaceDataSourceValue(value interface{}, resolvedData map[string]interface{}) interface{} {
	switch v := value.(type) {
	case string:
		// Check if this is a data source reference
		if strings.HasPrefix(v, "${data.") && strings.HasSuffix(v, "}") {
			// Extract the reference path
			ref := strings.TrimPrefix(v, "${")
			ref = strings.TrimSuffix(ref, "}")
			
			// Try to resolve the reference
			if resolved := resolveDataSourceReference(ref, resolvedData); resolved != nil {
				return resolved
			}
		}
		return v
		
	case map[string]interface{}:
		// Recursively process map values
		result := make(map[string]interface{})
		for k, val := range v {
			result[k] = replaceDataSourceValue(val, resolvedData)
		}
		return result
		
	case []interface{}:
		// Recursively process array elements
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = replaceDataSourceValue(val, resolvedData)
		}
		return result
		
	default:
		return v
	}
}

// resolveDataSourceReference resolves a data source reference path
// e.g., "data.tencentcloud_instance_types.instance_types.instance_types.0.instance_type"
func resolveDataSourceReference(ref string, resolvedData map[string]interface{}) interface{} {
	// Split the reference into parts
	parts := strings.Split(ref, ".")
	
	if len(parts) < 3 || parts[0] != "data" {
		return nil
	}
	
	// Build the data source key: "data.type.name"
	dsKey := strings.Join(parts[0:3], ".")
	
	// Look up the resolved data source
	data, ok := resolvedData[dsKey]
	if !ok {
		return nil
	}
	
	// Navigate through the remaining path
	current := data
	for i := 3; i < len(parts); i++ {
		part := parts[i]
		
		// Check if this is a numeric index (e.g., "0", "1", "2")
		var index int
		if _, err := fmt.Sscanf(part, "%d", &index); err == nil {
			// This is an array index
			if arr, ok := current.([]interface{}); ok {
				if index >= 0 && index < len(arr) {
					current = arr[index]
				} else {
					return nil
				}
			} else if arr, ok := current.([]map[string]interface{}); ok {
				if index >= 0 && index < len(arr) {
					current = arr[index]
				} else {
					return nil
				}
			} else {
				return nil
			}
		} else {
			// This is a map key
			if m, ok := current.(map[string]interface{}); ok {
				val, exists := m[part]
				if !exists {
					return nil
				}
				current = val
			} else {
				return nil
			}
		}
	}
	
	return current
}

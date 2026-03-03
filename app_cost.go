package main

import (
	"fmt"
	"strings"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/cost"

	tfjson "github.com/hashicorp/terraform-json"
)

// GetCostEstimate calculates cost estimate for a template
func (a *App) GetCostEstimate(templateName string, variables map[string]string) (*cost.CostEstimate, error) {
	a.mu.Lock()
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	if pricingService == nil || costCalculator == nil {
		err := fmt.Errorf("%s", i18n.T("app_cost_estimate_not_init"))
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Cost estimation service not initialized for template: %s\n", templateName)))
				logger.Close()
			}
		}
		return nil, err
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting cost estimation for template: %s\n", templateName)))
			if len(variables) > 0 {
				logger.Write([]byte(fmt.Sprintf("[INFO] Variables provided: %d\n", len(variables))))
			}
			logger.Close()
		}
	}

	templatePath, err := redc.GetTemplatePath(templateName)
	if err != nil {
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template not found: %s, error: %v\n", templateName, err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf(i18n.Tf("app_template_not_found", err))
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Template path resolved: %s\n", templatePath)))
			logger.Close()
		}
	}

	resources, err := cost.ParseTemplate(templatePath, variables)
	if err != nil {
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template parsing failed for: %s\n", templateName)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Template path: %s\n", templatePath)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Parse error: %v\n", err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf(i18n.Tf("app_template_parse_failed", err))
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Template parsed successfully: %d resources found\n", len(resources.Resources))))
			if resources.Provider != "" {
				logger.Write([]byte(fmt.Sprintf("[INFO] Primary provider: %s\n", resources.Provider)))
			}
			if resources.Region != "" {
				logger.Write([]byte(fmt.Sprintf("[INFO] Primary region: %s\n", resources.Region)))
			}
			logger.Close()
		}
	}

	estimate, err := costCalculator.CalculateCost(resources, pricingService)
	if err != nil {
		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[ERROR] Cost calculation failed for template: %s\n", templateName)))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Resource count: %d\n", len(resources.Resources))))
				logger.Write([]byte(fmt.Sprintf("[ERROR] Calculation error: %v\n", err)))
				logger.Close()
			}
		}
		return nil, fmt.Errorf(i18n.Tf("app_cost_calculate_failed", err))
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-estimation"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Cost estimation completed successfully for template: %s\n", templateName)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total hourly cost: %.4f %s\n", estimate.TotalHourlyCost, estimate.Currency)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total monthly cost: %.2f %s\n", estimate.TotalMonthlyCost, estimate.Currency)))
			logger.Write([]byte(fmt.Sprintf("[INFO] Resources in breakdown: %d\n", len(estimate.Breakdown))))
			if estimate.UnavailableCount > 0 {
				logger.Write([]byte(fmt.Sprintf("[WARN] Resources with unavailable pricing: %d\n", estimate.UnavailableCount)))
			}
			if len(estimate.Warnings) > 0 {
				logger.Write([]byte(fmt.Sprintf("[WARN] Warnings generated: %d\n", len(estimate.Warnings))))
				for i, warning := range estimate.Warnings {
					logger.Write([]byte(fmt.Sprintf("[WARN]   %d. %s\n", i+1, warning)))
				}
			}
			logger.Close()
		}
	}

	return estimate, nil
}

// GetTotalRuntime calculates total runtime of all running cases
func (a *App) GetTotalRuntime() (string, error) {
	a.mu.Lock()
	project := a.project
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return "0h", fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "0h", err
	}

	totalMinutes := 0
	now := time.Now()

	for _, c := range cases {
		if c.State == redc.StateRunning {
			var stateTime time.Time
			var parseErr error

			stateTime, parseErr = time.Parse(time.RFC3339, c.StateTime)
			if parseErr != nil {
				stateTime, parseErr = time.Parse("2006-01-02 15:04:05 -07:00", c.StateTime)
				if parseErr != nil {
					stateTime, parseErr = time.ParseInLocation("2006-01-02 15:04:05", c.StateTime, time.Local)
					if parseErr != nil {
						if logMgr != nil {
							if logger, logErr := logMgr.NewServiceLogger("runtime"); logErr == nil {
								logger.Write([]byte(fmt.Sprintf("[WARN] Failed to parse StateTime for case %s: %s (error: %v)\n", c.Name, c.StateTime, parseErr)))
								logger.Close()
							}
						}
						continue
					}
				}
			}

			duration := now.Sub(stateTime)
			if duration < 0 && strings.HasSuffix(c.StateTime, "Z") {
				timeStr := strings.TrimSuffix(c.StateTime, "Z")
				timeStr = strings.Replace(timeStr, "T", " ", 1)
				stateTime, parseErr = time.ParseInLocation("2006-01-02 15:04:05", timeStr, time.Local)
				if parseErr == nil {
					duration = now.Sub(stateTime)
				}
			}

			minutes := int(duration.Minutes())

			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("runtime"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[DEBUG] Case %s: StateTime=%s, Now=%s, Duration=%v, Minutes=%d\n",
						c.Name, stateTime.Format(time.RFC3339), now.Format(time.RFC3339), duration, minutes)))
					logger.Close()
				}
			}

			if minutes > 0 {
				totalMinutes += minutes
			}
		}
	}

	hours := totalMinutes / 60
	return fmt.Sprintf("%dh", hours), nil
}

// GetPredictedMonthlyCost calculates predicted monthly cost for all running cases
func (a *App) GetPredictedMonthlyCost() (string, error) {
	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return "¥0.00", fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	if pricingService == nil || costCalculator == nil {
		return "¥0.00", fmt.Errorf("%s", i18n.T("app_cost_estimate_not_init"))
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return "¥0.00", err
	}

	totalMonthlyCost := 0.0
	currency := "CNY"
	runningCount := 0

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting predicted monthly cost calculation\n")))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total cases: %d\n", len(cases))))
			logger.Close()
		}
	}

	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Processing running case: %s (path: %s)\n", c.Name, c.Path)))
				logger.Close()
			}
		}

		if c.Path == "" {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has empty path, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		state, err := redc.TfStatus(c.Path)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to get terraform state for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			continue
		}

		if state == nil || state.Values == nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has nil state or values, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		resources := extractResourcesFromState(state)
		if len(resources.Resources) == 0 {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has no resources, skipping\n", c.Name)))
					logger.Close()
				}
			}
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s has %d resources, calculating cost...\n", c.Name, len(resources.Resources))))
				logger.Write([]byte(fmt.Sprintf("[DEBUG] Provider: %s, Region: %s\n", resources.Provider, resources.Region)))
				for i, res := range resources.Resources {
					if i >= 3 {
						break
					}
					instanceType := "N/A"
					if it, ok := res.Attributes["instance_type"].(string); ok {
						instanceType = it
					}
					region := "N/A"
					if r, ok := res.Attributes["region"].(string); ok {
						region = r
					}
					zone := "N/A"
					if z, ok := res.Attributes["zone"].(string); ok {
						zone = z
					}
					availabilityZone := "N/A"
					if az, ok := res.Attributes["availability_zone"].(string); ok {
						availabilityZone = az
					}
					zoneId := "N/A"
					if zid, ok := res.Attributes["zone_id"].(string); ok {
						zoneId = zid
					}
					logger.Write([]byte(fmt.Sprintf("[DEBUG] Resource %d: Type=%s, Name=%s, InstanceType=%s, Region=%s, Zone=%s, AvailabilityZone=%s, ZoneId=%s, ResourceRegion=%s\n",
						i+1, res.Type, res.Name, instanceType, region, zone, availabilityZone, zoneId, res.Region)))
				}
				logger.Close()
			}
		}

		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to calculate cost for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s monthly cost: %.2f %s\n", c.Name, estimate.TotalMonthlyCost, estimate.Currency)))
				logger.Close()
			}
		}

		totalMonthlyCost += estimate.TotalMonthlyCost
		if estimate.Currency != "" {
			currency = estimate.Currency
		}
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-prediction"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Prediction complete - Running cases: %d, Total monthly cost: %.2f %s\n", runningCount, totalMonthlyCost, currency)))
			logger.Close()
		}
	}

	var symbol string
	switch currency {
	case "CNY":
		symbol = "¥"
	case "USD":
		symbol = "$"
	default:
		symbol = currency + " "
	}

	return fmt.Sprintf("%s%.2f", symbol, totalMonthlyCost), nil
}

// extractResourcesFromState converts terraform state to cost.TemplateResources
func extractResourcesFromState(state *tfjson.State) *cost.TemplateResources {
	resources := &cost.TemplateResources{Resources: []cost.ResourceSpec{}}

	if state.Values == nil || state.Values.RootModule == nil {
		return resources
	}

	if len(state.Values.RootModule.Resources) > 0 {
		firstResource := state.Values.RootModule.Resources[0]
		if firstResource.ProviderName != "" {
			providerName := extractShortProviderName(firstResource.ProviderName)
			resources.Provider = providerName
		}
	}

	extractModuleResources(state.Values.RootModule, resources)

	for _, res := range resources.Resources {
		if region, ok := res.Attributes["region"].(string); ok && region != "" {
			resources.Region = region
			break
		} else if availabilityZone, ok := res.Attributes["availability_zone"].(string); ok && availabilityZone != "" {
			if len(availabilityZone) > 2 {
				lastDash := strings.LastIndex(availabilityZone, "-")
				if lastDash > 0 {
					resources.Region = availabilityZone[:lastDash]
					break
				}
			}
		} else if zone, ok := res.Attributes["zone"].(string); ok && zone != "" {
			if len(zone) > 2 {
				lastDash := strings.LastIndex(zone, "-")
				if lastDash > 0 {
					resources.Region = zone[:lastDash]
					break
				}
			}
		} else if zoneId, ok := res.Attributes["zone_id"].(string); ok && zoneId != "" {
			if len(zoneId) > 2 {
				lastDash := strings.LastIndex(zoneId, "-")
				if lastDash > 0 {
					resources.Region = zoneId[:lastDash]
					break
				}
			}
		}
	}

	return resources
}

// extractShortProviderName extracts the short provider name from full registry path
func extractShortProviderName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return fullName
}

// extractModuleResources recursively extracts resources from a terraform module
func extractModuleResources(module *tfjson.StateModule, resources *cost.TemplateResources) {
	if module == nil {
		return
	}

	for _, res := range module.Resources {
		if res.Type == "" {
			continue
		}

		providerName := extractShortProviderName(res.ProviderName)

		costRes := cost.ResourceSpec{
			Type:       res.Type,
			Name:       res.Name,
			Provider:   providerName,
			Count:      1,
			Attributes: make(map[string]interface{}),
		}

		if res.AttributeValues != nil {
			for key, value := range res.AttributeValues {
				costRes.Attributes[key] = value
			}

			if region, ok := res.AttributeValues["region"].(string); ok && region != "" {
				costRes.Region = region
			} else if availabilityZone, ok := res.AttributeValues["availability_zone"].(string); ok && availabilityZone != "" {
				if len(availabilityZone) > 2 {
					lastDash := strings.LastIndex(availabilityZone, "-")
					if lastDash > 0 {
						costRes.Region = availabilityZone[:lastDash]
					}
				}
			} else if zone, ok := res.AttributeValues["zone"].(string); ok && zone != "" {
				if len(zone) > 2 {
					lastDash := strings.LastIndex(zone, "-")
					if lastDash > 0 {
						costRes.Region = zone[:lastDash]
					}
				}
			} else if zoneId, ok := res.AttributeValues["zone_id"].(string); ok && zoneId != "" {
				if len(zoneId) > 2 {
					lastDash := strings.LastIndex(zoneId, "-")
					if lastDash > 0 {
						costRes.Region = zoneId[:lastDash]
					}
				}
			}

			if res.Type == "alicloud_instance" || res.Type == "aws_instance" ||
				res.Type == "tencentcloud_instance" || res.Type == "volcengine_ecs_instance" {
				if instanceType, ok := res.AttributeValues["instance_type"].(string); ok && instanceType != "" {
					costRes.Attributes["instance_type"] = instanceType
				}
			}
		}

		resources.Resources = append(resources.Resources, costRes)
	}

	for _, child := range module.ChildModules {
		extractModuleResources(child, resources)
	}
}

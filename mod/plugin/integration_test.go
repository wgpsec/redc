// mod/plugin/integration_test.go
package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	redc "red-cloud/mod"
	"strconv"
	"strings"
	"testing"
)

func TestIntegration_ClashConfigTemplate(t *testing.T) {
	// Simulate the full flow: plugin loaded → post-apply hook → config generated

	// Setup plugin dir
	pluginDir := t.TempDir()
	os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755)

	tmpl := `mixed-port: 64277
proxies:
{{- range .IPs}}
  - name: "{{.}}"
    type: ss
    server: {{.}}
    port: {{$.Vars.port}}
    password: "{{$.Vars.password}}"
{{- end}}
{{- setOutput "clash_node_count" (printf "%d" (len .IPs)) -}}
`
	os.WriteFile(filepath.Join(pluginDir, "hooks", "post-apply.tmpl"), []byte(tmpl), 0644)

	// Setup case dir
	caseDir := t.TempDir()
	os.WriteFile(filepath.Join(caseDir, "terraform.tfvars"), []byte("port = \"8388\"\npassword = \"secret123\"\n"), 0644)

	// Terraform output
	outputs := map[string]interface{}{
		"ecs_ip": map[string]interface{}{
			"value": []interface{}{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		},
	}
	outputJSON, _ := json.Marshal(outputs)

	hook := HookEntry{
		PluginName:   "redc-plugin-clash-config",
		PluginDir:    pluginDir,
		Type:         "template",
		TemplatePath: filepath.Join(pluginDir, "hooks", "post-apply.tmpl"),
		OutputPath:   "{{.CasePath}}/config.yaml",
		Config:       map[string]interface{}{},
	}

	hctx := &HookContext{
		CaseName:   "my-proxy",
		CasePath:   caseDir,
		OutputJSON: string(outputJSON),
	}

	results, err := executeTemplateHook(hook, hctx)
	if err != nil {
		t.Fatalf("hook failed: %v", err)
	}

	// Verify output file
	content, err := os.ReadFile(filepath.Join(caseDir, "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not created: %v", err)
	}

	s := string(content)
	if !strings.Contains(s, "10.0.1.1") {
		t.Error("missing IP 10.0.1.1")
	}
	if !strings.Contains(s, "10.0.1.3") {
		t.Error("missing IP 10.0.1.3")
	}
	if !strings.Contains(s, `password: "secret123"`) {
		t.Error("missing password")
	}
	if !strings.Contains(s, "port: 8388") {
		t.Error("missing port")
	}

	// Verify outputs
	if results["clash_node_count"] != "3" {
		t.Errorf("expected node_count=3, got %q", results["clash_node_count"])
	}
}

func TestRunHooksPersistsPluginOutputsPrivately(t *testing.T) {
	pluginsDir := t.TempDir()
	pluginDir := filepath.Join(pluginsDir, "redc-plugin-output-test")
	if err := os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "name": "redc-plugin-output-test",
  "version": "1.0.0",
  "description": "test",
  "capabilities": {"hooks": {"post-apply": "hooks/post-apply.sh"}}
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(pluginDir, "hooks", "post-apply.sh"),
		[]byte("echo 'REDC_OUTPUT:new_value=secret-bearing-output'\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	caseDir := t.TempDir()
	outputsPath := filepath.Join(caseDir, pluginOutputsFile)
	if err := os.WriteFile(outputsPath, []byte(`{"existing":"preserved"}`), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewPluginManager(pluginsDir)
	if err := manager.LoadAll(); err != nil {
		t.Fatal(err)
	}
	if err := manager.RunHooks(HookPostApply, &HookContext{
		CasePath:       caseDir,
		AllowedPlugins: []string{"redc-plugin-output-test"},
	}); err != nil {
		t.Fatal(err)
	}

	got := LoadPluginOutputs(caseDir)
	if got["existing"] != "preserved" || got["new_value"] != "secret-bearing-output" {
		t.Fatalf("plugin outputs = %#v", got)
	}
	info, err := os.Stat(outputsPath)
	if err != nil {
		t.Fatal(err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0600 {
		t.Fatalf("plugin_outputs.json mode = %04o, want 0600", gotMode)
	}
	tempFiles, err := filepath.Glob(filepath.Join(caseDir, ".plugin_outputs.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("temporary output files left behind: %v", tempFiles)
	}
}

func TestRunHooksProvidesStableCaseID(t *testing.T) {
	pluginsDir := t.TempDir()
	pluginDir := filepath.Join(pluginsDir, "redc-plugin-case-id-test")
	if err := os.MkdirAll(filepath.Join(pluginDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "name": "redc-plugin-case-id-test",
  "version": "1.0.0",
  "description": "test",
  "capabilities": {"hooks": {"post-apply": "hooks/post-apply.sh"}}
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(pluginDir, "hooks", "post-apply.sh"),
		[]byte("echo \"REDC_OUTPUT:case_id=$REDC_CASE_ID\"\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	manager := NewPluginManager(pluginsDir)
	if err := manager.LoadAll(); err != nil {
		t.Fatal(err)
	}
	caseDir := t.TempDir()
	if err := manager.RunHooks(HookPostApply, &HookContext{
		CaseID:         "4dfaf64b-426c-40ec-84d7-456d2a43dc67",
		CasePath:       caseDir,
		AllowedPlugins: []string{"redc-plugin-case-id-test"},
	}); err != nil {
		t.Fatal(err)
	}

	if got := LoadPluginOutputs(caseDir)["case_id"]; got != "4dfaf64b-426c-40ec-84d7-456d2a43dc67" {
		t.Fatalf("case_id output = %q", got)
	}
}

func TestLoadManifestEnforcesMinimumRedCVersion(t *testing.T) {
	pluginDir := t.TempDir()
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	writeManifest := func(minVersion string) {
		t.Helper()
		raw := `{"name":"versioned-plugin","version":"1.0.0","description":"test","min_redc_version":"` + minVersion + `","capabilities":{}}`
		if err := os.WriteFile(manifestPath, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}

	// 基准取自实际的 redc.Version，避免每次升版本号都要改测试。
	current := strings.TrimPrefix(redc.Version, "v")
	writeManifest(current)
	if _, err := loadManifest(pluginDir); err != nil {
		t.Fatalf("compatible plugin was rejected: %v", err)
	}

	next := nextPatchVersion(t, current)
	writeManifest(next)
	if _, err := loadManifest(pluginDir); err == nil || !strings.Contains(err.Error(), "requires redc "+next+" or newer") {
		t.Fatalf("incompatible plugin error = %v", err)
	}
}

// nextPatchVersion returns the version one patch above v, e.g. "3.3.9" -> "3.3.10".
func nextPatchVersion(t *testing.T, v string) string {
	t.Helper()
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected version format %q", v)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("unexpected patch in %q: %v", v, err)
	}
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts, ".")
}

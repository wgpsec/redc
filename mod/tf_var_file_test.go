package mod

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestWriteVarFile(t *testing.T) {
	dir := t.TempDir()
	vars := []string{
		"instance_name=test_case",
		"node=7",
		"enable_monitor=true",
		"instance_password=R02f6581cbc_+e0c669e5259b",
	}

	path, cleanup, err := writeVarFile(dir, vars)
	if err != nil {
		t.Fatalf("writeVarFile: %v", err)
	}
	defer cleanup()

	if path == "" {
		t.Fatal("expected a var file path, got empty string")
	}
	// 变量文件里可能有实例密码，必须仅属主可读
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("var file mode = %o, want 0600", perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("var file is not valid JSON: %v", err)
	}
	if got["instance_password"] != "R02f6581cbc_+e0c669e5259b" {
		t.Errorf("instance_password = %v, want preserved", got["instance_password"])
	}
	// 值一律以字符串落盘，string→number/bool 的转换由 terraform 解析
	// var-file 时完成（已实测 "7" → number 7、"true" → bool true 均可用）。
	if got["node"] != "7" {
		t.Errorf("node = %v (%T), want \"7\"", got["node"], got["node"])
	}
	if got["enable_monitor"] != "true" {
		t.Errorf("enable_monitor = %v (%T), want \"true\"", got["enable_monitor"], got["enable_monitor"])
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove var file: err=%v", err)
	}
}

func TestWriteVarFileEmpty(t *testing.T) {
	dir := t.TempDir()
	for _, vars := range [][]string{nil, {}, {"no_equals_sign"}, {""}} {
		path, cleanup, err := writeVarFile(dir, vars)
		if err != nil {
			t.Fatalf("writeVarFile(%v): %v", vars, err)
		}
		if path != "" {
			t.Errorf("writeVarFile(%v) created %q, want no file", vars, path)
		}
		cleanup()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dir not empty: %v", entries)
	}
}

func TestWriteVarFilePreservesValueWithEquals(t *testing.T) {
	dir := t.TempDir()
	path, cleanup, err := writeVarFile(dir, []string{"base64_command=aGVsbG8="})
	if err != nil {
		t.Fatalf("writeVarFile: %v", err)
	}
	defer cleanup()

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "aGVsbG8=") {
		t.Errorf("value containing '=' was truncated: %s", data)
	}
}

func TestSecureTfState(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"terraform.tfstate", "terraform.tfstate.backup", "terraform.tfvars", RedcPlanPath} {
		if err := os.WriteFile(dir+"/"+name, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// 不存在的文件应被跳过而不 panic
	SecureTfState(dir)

	for _, name := range []string{"terraform.tfstate", "terraform.tfstate.backup", "terraform.tfvars", RedcPlanPath} {
		info, err := os.Stat(dir + "/" + name)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("%s mode = %o, want 0600", name, perm)
		}
	}
}

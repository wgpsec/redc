# Patched terraform-exec

This is a patched version of `github.com/hashicorp/terraform-exec` v0.24.0.

## Patch Details

### Added: `tfexec/cmd_windows.go`

Added Windows-specific command execution that hides console windows when running Terraform commands in GUI applications.

**Issue**: On Windows, executing Terraform commands via `terraform-exec` causes CMD console windows to pop up, which is undesirable in GUI applications.

**Solution**: Set Windows-specific process creation flags:
- `CreationFlags = 0x08000000` (CREATE_NO_WINDOW)

This prevents the creation of new console windows when Terraform subprocesses are launched.

### Modified: `tfexec/cmd_default.go`

Updated build constraints from `!linux` to `!linux && !windows` to prevent conflicts with the new `cmd_windows.go` file.

## Why Local Patch?

This patch is necessary because the upstream `terraform-exec` library does not currently support hiding console windows on Windows. The change is non-invasive and only affects Windows GUI scenarios.

## Upstream

This fix should ideally be contributed upstream to the `terraform-exec` project. Track progress at:
https://github.com/hashicorp/terraform-exec

## Version

Based on: `github.com/hashicorp/terraform-exec` v0.24.0

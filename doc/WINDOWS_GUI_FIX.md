# Windows GUI CMD Popup Fix

## Problem
When running Redc in GUI mode on Windows (using Wails), Terraform command executions caused CMD console windows to pop up, disrupting the user experience.

## Root Cause
The `terraform-exec` library (v0.24.0) doesn't set Windows-specific process creation flags when spawning Terraform subprocesses. On Windows, by default, new console processes create visible console windows.

## Solution
We've patched the `terraform-exec` library to add Windows-specific behavior:

1. Created `internal/terraform-exec/tfexec/cmd_windows.go` with the `//go:build windows` tag
2. Set `CreationFlags = 0x08000000` (CREATE_NO_WINDOW) in the `SysProcAttr`
3. Updated `cmd_default.go` build constraints to exclude Windows

## Technical Details

The CREATE_NO_WINDOW flag (0x08000000) tells Windows to:
- Not create a new console window for the process
- Use the parent process's console if one exists
- Run without any visible console if the parent has no console (GUI mode)

This is the standard approach for Windows GUI applications that need to spawn command-line tools without showing console windows.

## Files Changed

1. **go.mod**: Added replace directive to use local patched terraform-exec
2. **internal/terraform-exec/tfexec/cmd_windows.go**: New Windows-specific implementation
3. **internal/terraform-exec/tfexec/cmd_default.go**: Updated build constraints
4. **internal/terraform-exec/**: Complete vendored copy of terraform-exec v0.24.0

## Testing

To test this fix:
1. Build the Windows GUI application: `wails build`
2. Run the application on Windows
3. Perform any Terraform operations (init, plan, apply, destroy)
4. Verify that no CMD windows pop up during execution

## Future Work

This fix should be contributed upstream to the terraform-exec project. Monitor:
- https://github.com/hashicorp/terraform-exec/issues

## Platform Support

- **Windows**: Uses cmd_windows.go with CREATE_NO_WINDOW flag
- **Linux**: Uses cmd_linux.go with Pdeathsig and Setpgid
- **macOS/Other**: Uses cmd_default.go with no special process attributes

## References

- Go syscall.SysProcAttr: https://pkg.go.dev/syscall#SysProcAttr
- Windows CreateProcess flags: https://docs.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
- CREATE_NO_WINDOW: Prevents creation of a new console window

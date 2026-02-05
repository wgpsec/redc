# How to Build and Test the Windows GUI Fix

## Building the Windows GUI Application

### On Windows
```bash
# Install Wails if not already installed
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build the GUI application
wails build

# The executable will be in build/bin/
```

### Cross-compile from Linux/macOS
```bash
# Build for Windows
wails build -platform windows/amd64

# Output: build/bin/redc-gui.exe
```

## Testing the Fix

### Before the Fix
When running Terraform commands in the GUI, users would see:
- CMD console windows popping up
- Black terminal windows appearing briefly
- Distracting flashes during operations

### After the Fix
- ✅ No CMD windows should appear
- ✅ Operations run silently in the background
- ✅ GUI remains the only visible window
- ✅ Output still captured in the GUI

### Test Cases

1. **Initialize a Case**
   - Open the GUI application
   - Create a new case
   - Click "Initialize"
   - **Expected**: No CMD window appears

2. **Plan Changes**
   - Select an initialized case
   - Click "Plan"
   - **Expected**: No CMD window appears, plan output shown in GUI

3. **Apply Changes**
   - Click "Apply"
   - **Expected**: No CMD window appears, apply progress shown in GUI

4. **Destroy Resources**
   - Click "Destroy"
   - **Expected**: No CMD window appears, destroy progress shown in GUI

### Verification Steps

1. Build the application:
   ```bash
   cd /home/runner/work/redc/redc
   wails build
   ```

2. Run on Windows machine with GUI

3. Perform any Terraform operation

4. Confirm no CMD windows appear

## Technical Verification

Check that the correct code is being used:

```bash
# Verify Windows-specific code is included in Windows build
GOOS=windows go list -f '{{.GoFiles}}' ./internal/terraform-exec/tfexec

# Should include cmd_windows.go
```

## Rollback

If needed, revert the changes:

```bash
git revert <commit-hash>
```

Or remove the replace directive from go.mod:

```diff
-replace github.com/hashicorp/terraform-exec => ./internal/terraform-exec
```

Then run:
```bash
go mod tidy
```

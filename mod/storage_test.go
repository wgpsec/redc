package mod

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestBboltConcurrentCaseAdd tests adding cases concurrently
func TestBboltConcurrentCaseAdd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "redc-bbolt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	projectName := "test-concurrent-add"
	project, err := NewProjectConfig(projectName, "test-user")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	defer project.Close()

	numCases := 10
	var wg sync.WaitGroup
	wg.Add(numCases)

	for i := 0; i < numCases; i++ {
		go func(id int) {
			defer wg.Done()

			caseID := GenerateCaseID()
			newCase := &Case{
				Id:         caseID,
				Name:       "test-case-" + string(rune('A'+id)),
				Type:       "test",
				Operator:   "test-user",
				Path:       filepath.Join(project.ProjectPath, caseID),
				CreateTime: time.Now().Format("2006-01-02 15:04:05"),
				StateTime:  time.Now().Format("2006-01-02 15:04:05"),
				State:      StateCreated,
				project:    project,
			}

			if err := project.AddCase(newCase); err != nil {
				t.Errorf("Failed to add case: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Close before reloading
	project.Close()

	// Reload project and verify
	reloadedProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to reload project: %v", err)
	}
	defer reloadedProject.Close()

	if len(reloadedProject.Case) != numCases {
		t.Errorf("Expected %d cases, got %d", numCases, len(reloadedProject.Case))
	}
}

// TestBboltConcurrentStateUpdate tests concurrent state updates
func TestBboltConcurrentStateUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "redc-bbolt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	projectName := "test-concurrent-state"
	project, err := NewProjectConfig(projectName, "test-user")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	defer project.Close()

	// Add some cases first
	numCases := 5
	caseIDs := make([]string, numCases)
	for i := 0; i < numCases; i++ {
		caseID := GenerateCaseID()
		caseIDs[i] = caseID
		c := &Case{
			Id:         caseID,
			Name:       "test-case-" + string(rune('A'+i)),
			Type:       "test",
			Operator:   "test-user",
			Path:       filepath.Join(project.ProjectPath, caseID),
			CreateTime: time.Now().Format("2006-01-02 15:04:05"),
			StateTime:  time.Now().Format("2006-01-02 15:04:05"),
			State:      StateCreated,
			project:    project,
		}
		if err := project.AddCase(c); err != nil {
			t.Fatalf("Failed to add case: %v", err)
		}
	}

	// Update states concurrently
	var wg sync.WaitGroup
	wg.Add(numCases)

	for i := 0; i < numCases; i++ {
		go func(index int) {
			defer wg.Done()
			stateTime := time.Now().Add(time.Duration(index) * time.Second).Format("2006-01-02 15:04:05")
			if err := project.UpdateCaseState(caseIDs[index], StateRunning, stateTime); err != nil {
				t.Errorf("Failed to update state: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Close before reloading
	project.Close()

	// Reload and verify
	reloadedProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to reload project: %v", err)
	}
	defer reloadedProject.Close()

	runningCount := 0
	for _, c := range reloadedProject.Case {
		if c.State == StateRunning {
			runningCount++
		}
	}

	if runningCount != numCases {
		t.Errorf("Expected %d running cases, got %d", numCases, runningCount)
	}
}

// TestBboltBasicOperations tests basic CRUD operations
func TestBboltBasicOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "redc-bbolt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	// Create project
	projectName := "test-basic-ops"
	project, err := NewProjectConfig(projectName, "test-user")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	defer project.Close()

	// Add a case
	caseID := GenerateCaseID()
	c := &Case{
		Id:         caseID,
		Name:       "test-case",
		Type:       "test",
		Operator:   "test-user",
		Path:       filepath.Join(project.ProjectPath, caseID),
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		StateTime:  time.Now().Format("2006-01-02 15:04:05"),
		State:      StateCreated,
		project:    project,
	}

	if err := project.AddCase(c); err != nil {
		t.Fatalf("Failed to add case: %v", err)
	}

	// Update state
	if err := project.UpdateCaseState(caseID, StateRunning, time.Now().Format("2006-01-02 15:04:05")); err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Close before reloading
	project.Close()

	// Verify state
	reloadedProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to reload project: %v", err)
	}
	defer reloadedProject.Close()

	if len(reloadedProject.Case) != 1 {
		t.Errorf("Expected 1 case, got %d", len(reloadedProject.Case))
	}

	if reloadedProject.Case[0].State != StateRunning {
		t.Errorf("Expected state Running, got %s", reloadedProject.Case[0].State)
	}

	// Delete case
	if err := reloadedProject.HandleCase(reloadedProject.Case[0]); err != nil {
		t.Fatalf("Failed to delete case: %v", err)
	}

	// Close before final verification
	reloadedProject.Close()

	// Verify deletion
	finalProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to reload project after deletion: %v", err)
	}
	defer finalProject.Close()

	if len(finalProject.Case) != 0 {
		t.Errorf("Expected 0 cases after deletion, got %d", len(finalProject.Case))
	}
}

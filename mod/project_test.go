package mod

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestConcurrentSaveProject tests that concurrent SaveProject calls don't overwrite each other's changes
func TestConcurrentSaveProject(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "redc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the project path for testing
	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	// Create a test project
	projectName := "test-project"
	project := &RedcProject{
		ProjectName: projectName,
		ProjectPath: filepath.Join(tmpDir, projectName),
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		User:        "test-user",
		Case:        make([]*Case, 0),
	}

	// Create project directory
	if err := os.MkdirAll(project.ProjectPath, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Save initial project
	if err := project.SaveProject(); err != nil {
		t.Fatalf("Failed to save initial project: %v", err)
	}

	// Number of concurrent operations
	numGoroutines := 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Each goroutine will add a unique case to the project concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Load the project from disk
			loadedProject, err := ProjectByName(projectName)
			if err != nil {
				t.Errorf("Failed to load project in goroutine %d: %v", id, err)
				return
			}

			// Add a unique case
			caseID := GenerateCaseID()
			newCase := &Case{
				Id:         caseID,
				Name:       "test-case-" + string(rune('A'+id)),
				Type:       "test",
				Operator:   "test-user",
				Path:       filepath.Join(loadedProject.ProjectPath, caseID),
				CreateTime: time.Now().Format("2006-01-02 15:04:05"),
				StateTime:  time.Now().Format("2006-01-02 15:04:05"),
				State:      StateCreated,
			}

			loadedProject.Case = append(loadedProject.Case, newCase)

			// Save the project (this should be thread-safe)
			if err := loadedProject.SaveProject(); err != nil {
				t.Errorf("Failed to save project in goroutine %d: %v", id, err)
				return
			}

			// Small delay to increase chance of concurrent access
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Load the final project and verify all cases were saved
	finalProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to load final project: %v", err)
	}

	// Check that we have all the expected cases
	if len(finalProject.Case) != numGoroutines {
		t.Errorf("Expected %d cases, but got %d", numGoroutines, len(finalProject.Case))
		t.Logf("Cases found: %v", finalProject.Case)
	}

	// Verify each case is unique
	caseIDs := make(map[string]bool)
	for _, c := range finalProject.Case {
		if caseIDs[c.Id] {
			t.Errorf("Duplicate case ID found: %s", c.Id)
		}
		caseIDs[c.Id] = true
	}
}

// TestConcurrentStatusChange tests that concurrent status changes don't cause data loss
func TestConcurrentStatusChange(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "redc-test-status-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the project path for testing
	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	// Create a test project with multiple cases
	projectName := "test-status-project"
	numCases := 5
	cases := make([]*Case, numCases)
	
	for i := 0; i < numCases; i++ {
		caseID := GenerateCaseID()
		cases[i] = &Case{
			Id:         caseID,
			Name:       "test-case-" + string(rune('A'+i)),
			Type:       "test",
			Operator:   "test-user",
			Path:       filepath.Join(tmpDir, projectName, caseID),
			CreateTime: time.Now().Format("2006-01-02 15:04:05"),
			StateTime:  time.Now().Format("2006-01-02 15:04:05"),
			State:      StateCreated,
		}
	}

	project := &RedcProject{
		ProjectName: projectName,
		ProjectPath: filepath.Join(tmpDir, projectName),
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		User:        "test-user",
		Case:        cases,
	}

	// Create project directory
	if err := os.MkdirAll(project.ProjectPath, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Save initial project
	if err := project.SaveProject(); err != nil {
		t.Fatalf("Failed to save initial project: %v", err)
	}

	// Bind handlers for all cases
	for _, c := range cases {
		c.bindHandlers(project)
	}

	// Concurrently change the status of all cases
	var wg sync.WaitGroup
	wg.Add(numCases)

	for i := 0; i < numCases; i++ {
		go func(caseIndex int) {
			defer wg.Done()
			
			// Change status to running
			cases[caseIndex].StatusChange(StateRunning)
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Reload the project from disk
	finalProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to load final project: %v", err)
	}

	// Verify all cases are present and have running state
	if len(finalProject.Case) != numCases {
		t.Errorf("Expected %d cases, but got %d", numCases, len(finalProject.Case))
	}

	// Count how many cases have the running state
	runningCount := 0
	for _, c := range finalProject.Case {
		if c.State == StateRunning {
			runningCount++
		}
	}

	// All cases should be in running state
	if runningCount != numCases {
		t.Errorf("Expected all %d cases to be running, but only %d are running", numCases, runningCount)
		
		// Print detailed state information
		for _, c := range finalProject.Case {
			t.Logf("Case %s: State=%s", c.Name, c.State)
		}
	}
}

// TestSaveProjectPreservesExistingData tests that SaveProject doesn't lose data when multiple instances exist
func TestSaveProjectPreservesExistingData(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "redc-test-preserve-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the project path for testing
	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	// Create a test project with initial cases
	projectName := "test-preserve-project"
	project1 := &RedcProject{
		ProjectName: projectName,
		ProjectPath: filepath.Join(tmpDir, projectName),
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		User:        "test-user",
		Case: []*Case{
			{
				Id:         "case1",
				Name:       "Case 1",
				State:      StateCreated,
				CreateTime: time.Now().Format("2006-01-02 15:04:05"),
			},
		},
	}

	// Create project directory
	if err := os.MkdirAll(project1.ProjectPath, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Save initial project
	if err := project1.SaveProject(); err != nil {
		t.Fatalf("Failed to save initial project: %v", err)
	}

	// Load the same project in a different instance
	project2, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to load project in instance 2: %v", err)
	}

	// Add a case to project2
	project2.Case = append(project2.Case, &Case{
		Id:         "case2",
		Name:       "Case 2",
		State:      StateCreated,
		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
	})

	// Modify the state of case1 in project1
	project1.Case[0].State = StateRunning
	project1.Case[0].StateTime = time.Now().Format("2006-01-02 15:04:05")

	// Save both projects (simulating concurrent operations)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := project1.SaveProject(); err != nil {
			t.Errorf("Failed to save project1: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := project2.SaveProject(); err != nil {
			t.Errorf("Failed to save project2: %v", err)
		}
	}()

	wg.Wait()

	// Load the final project
	finalProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to load final project: %v", err)
	}

	// Verify both cases are present
	if len(finalProject.Case) != 2 {
		t.Errorf("Expected 2 cases, but got %d", len(finalProject.Case))
		
		// Debug: print the project file content
		projectFile := filepath.Join(tmpDir, projectName, ProjectFile)
		data, _ := os.ReadFile(projectFile)
		t.Logf("Project file content:\n%s", string(data))
	}

	// Verify case1 has the updated state
	var case1Found, case2Found bool
	for _, c := range finalProject.Case {
		if c.Id == "case1" {
			case1Found = true
			if c.State != StateRunning {
				t.Errorf("Expected case1 to be running, but got state: %s", c.State)
			}
		}
		if c.Id == "case2" {
			case2Found = true
		}
	}

	if !case1Found {
		t.Error("Case 1 not found in final project")
	}
	if !case2Found {
		t.Error("Case 2 not found in final project")
	}
}

// TestProjectByNameInitializesMutex tests that loading a project from disk properly initializes the mutex
func TestProjectByNameInitializesMutex(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "redc-test-mutex-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the project path for testing
	originalProjectPath := ProjectPath
	ProjectPath = tmpDir
	defer func() { ProjectPath = originalProjectPath }()

	// Create a test project
	projectName := "test-mutex-project"
	projectPath := filepath.Join(tmpDir, projectName)
	
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Manually create a project file (simulating an existing project)
	project := &RedcProject{
		ProjectName: projectName,
		ProjectPath: projectPath,
		CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		User:        "test-user",
		Case:        []*Case{},
	}

	data, err := json.MarshalIndent(project, "", "    ")
	if err != nil {
		t.Fatalf("Failed to marshal project: %v", err)
	}

	projectFile := filepath.Join(projectPath, ProjectFile)
	if err := os.WriteFile(projectFile, data, 0644); err != nil {
		t.Fatalf("Failed to write project file: %v", err)
	}

	// Load the project
	loadedProject, err := ProjectByName(projectName)
	if err != nil {
		t.Fatalf("Failed to load project: %v", err)
	}

	// Try to use the mutex by saving the project (this should not panic)
	if err := loadedProject.SaveProject(); err != nil {
		t.Errorf("Failed to save loaded project: %v", err)
	}
}

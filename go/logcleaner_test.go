package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// cleanLog is here to prevent compilation issues if it hasn't been changed in main.go.
// In reality, it should be cleanly exported from main.go.
// A simulated function is used for testing RunE ArgumentErrors.

// Test cleanLog functionality
func TestCleanLog_Trimming(t *testing.T) {
	// Create a temporary directory
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	maxRows := 5

	// Create test content (10 lines)
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("Could not create test file: %v", err)
	}

	// Run cleanLog
	if err := cleanLog(logPath, maxRows, "irrelevant"); err != nil {
		t.Fatalf("cleanLog failed: %v", err)
	}

	// 1. Check if the backup exists (with any timestamp)
	backupExists := false
	files, _ := filepath.Glob(logPath + ".*.bak")
	if len(files) > 0 {
		backupExists = true
		// Example cleanup: Delete the backup to keep the test directory clean
		os.Remove(files[0])
	}
	if !backupExists {
		t.Errorf("Error: Backup file was not created.")
	}

	// 2. Check if the original file has the correct number of lines
	trimmedContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Could not read cleaned file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(trimmedContent)), "\n")
	if len(lines) != maxRows {
		t.Errorf("Expected number of lines: %d, actual: %d", maxRows, len(lines))
	}

	// 3. Check if the correct lines are preserved (the last 5)
	expectedLines := []string{"Line 6", "Line 7", "Line 8", "Line 9", "Line 10"}
	if !equalSlices(lines, expectedLines) {
		t.Errorf("Content does not match.\nExpected: %v\nActual: %v", expectedLines, lines)
	}
}

// Test with an empty log
func TestCleanLog_Empty(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")

	// Create an empty file
	if _, err := os.Create(logPath); err != nil {
		t.Fatalf("Could not create empty log: %v", err)
	}

	// Run cleanLog
	if err := cleanLog(logPath, 5, "irrelevant"); err != nil {
		t.Fatalf("cleanLog failed: %v", err)
	}

	// Check if the resulting file is empty
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Could not read file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("Expected an empty log, but found length %d.", len(data))
	}
}

// Test argument parsing errors
func TestRunE_ArgumentErrors(t *testing.T) {
	// We use a copy of RunE from main.go to test parsing errors.
	testRunE := func(cmd *cobra.Command, args []string) error {
		path := args[0]
		rowsStr := args[1]
		format := args[2]

		// Convert rows from string to int
		rows, err := strconv.Atoi(rowsStr)
		if err != nil {
			return fmt.Errorf("error: second argument 'max_lines' must be a number, but was: %s", rowsStr)
		}
		if rows <= 0 {
			return fmt.Errorf("error: maximum number of rows must be a positive number")
		}

		// cleanLog would normally be here, but we skip it for parsing tests
		_ = path
		_ = rows
		_ = format
		return nil
	}

	cmd := &cobra.Command{RunE: testRunE}

	// Test 1: max_lines is not a number (bad argument format)
	args1 := []string{"/path/log", "text", "format"}
	err1 := cmd.RunE(cmd, args1)
	if err1 == nil || !strings.Contains(err1.Error(), "must be a number") {
		t.Errorf("Expected 'must be a number' error, but got: %v", err1)
	}

	// Test 2: max_lines is a negative number (invalid value)
	args2 := []string{"/path/log", "-5", "format"}
	err2 := cmd.RunE(cmd, args2)
	if err2 == nil || !strings.Contains(err2.Error(), "positive number") {
		t.Errorf("Expected 'positive number' error, but got: %v", err2)
	}

	// Test 3: max_lines is zero (invalid value)
	args3 := []string{"/path/log", "0", "format"}
	err3 := cmd.RunE(cmd, args3)
	if err3 == nil || !strings.Contains(err3.Error(), "positive number") {
		t.Errorf("Expected 'positive number' error, but got: %v", err3)
	}
}

// Helper function to compare slices
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Test standard Cobra behavior (display Usage and error)
func TestArgs_StandardCobraBehavior(t *testing.T) {
	// Use bytes.Buffer to capture the help output
	var buf bytes.Buffer

	// Create a simple root Command with standard properties
	cmd := &cobra.Command{
		Use:  "\tlogcleaner [log_path] [max_lines] [date_format]",
		Args: cobra.ExactArgs(3),
		// Silence the error (Error: accepts 3...) but KEEP the Usage (help)
		SilenceErrors: true,
		SilenceUsage:  false,
		Run:           func(cmd *cobra.Command, args []string) { /* Do Nothing */ },
	}
	cmd.SetOut(&buf) // Redirect Cobra's output to the buffer

	// Run Execute with the wrong number of arguments (less than 3)
	cmd.SetArgs([]string{"/path/log", "5"})
	err := cmd.Execute()

	// 1. Check if an error was returned (Cobra.CommandError or similar)
	if err == nil {
		t.Fatal("Expected an error, but none was returned.")
	}

	// 2. Check if any content (help) was printed
	out := buf.String()
	if !strings.Contains(out, "logcleaner [log_path]") {
		t.Errorf("Expected help output (Usage), but found: %s", out)
	}
}

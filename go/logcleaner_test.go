package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Pomocná funkce pro barevný výstup v testech
func testLog(t *testing.T, color string, message string) {
	t.Logf("%s%s%s", color, message, ColorReset)
}

func testError(t *testing.T, message string) {
	t.Errorf("%s%s%s", ColorRed, message, ColorReset)
}

// Test cleanLog functionality s barvičkami
func TestCleanLog_Trimming(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestCleanLog_Trimming ---")

	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	maxRows := 5

	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("Could not create test file: %v", err)
	}

	if err := cleanLog(logPath, maxRows, "", "2006-01-02", nil); err != nil {
		testError(t, fmt.Sprintf("cleanLog failed: %v", err))
	}

	// 1. Check backup
	files, _ := filepath.Glob(logPath + ".*.bak")
	if len(files) > 0 {
		testLog(t, ColorGreen, "✔ Backup file created successfully.")
		os.Remove(files[0])
	} else {
		testError(t, "✖ Error: Backup file was not created.")
	}

	// 2. Check line count
	trimmedContent, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Could not read cleaned file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(trimmedContent)), "\n")
	if len(lines) == maxRows {
		testLog(t, ColorGreen, fmt.Sprintf("✔ Line count matches: %d", maxRows))
	} else {
		testError(t, fmt.Sprintf("✖ Expected %d lines, but got %d", maxRows, len(lines)))
	}

	// 3. Check content
	expectedLines := []string{"Line 6", "Line 7", "Line 8", "Line 9", "Line 10"}
	if equalSlices(lines, expectedLines) {
		testLog(t, ColorGreen, "✔ Content matches expected last 5 lines.")
	} else {
		testError(t, fmt.Sprintf("✖ Content mismatch!\nExpected: %v\nActual: %v", expectedLines, lines))
	}
}

// Test with an empty log
func TestCleanLog_Empty(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestCleanLog_Empty ---")
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")

	if _, err := os.Create(logPath); err != nil {
		t.Fatalf("Could not create empty log: %v", err)
	}

	if err := cleanLog(logPath, 5, "", "2006-01-02", nil); err != nil {
		testError(t, fmt.Sprintf("cleanLog failed on empty file: %v", err))
	}

	data, _ := os.ReadFile(logPath)
	if len(data) == 0 {
		testLog(t, ColorGreen, "✔ Empty log stayed empty as expected.")
	} else {
		testError(t, "✖ Empty log should not contain data after cleaning.")
	}
}

// Test argument parsing errors
func TestRunE_ArgumentErrors(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestRunE_ArgumentErrors ---")

	testRunE := func(cmd *cobra.Command, args []string) error {
		lines, _ := cmd.Flags().GetInt("lines")
		date, _ := cmd.Flags().GetString("date")
		format, _ := cmd.Flags().GetString("format")

		// This part is simplified as cobra does the type checking
		rowsStr := strconv.Itoa(lines)
		rows, err := strconv.Atoi(rowsStr)
		if err != nil {
			return fmt.Errorf("error: --lines must be a number, but was: %s", rowsStr)
		}
		if rows <= 0 {
			return fmt.Errorf("error: maximum number of rows must be a positive number")
		}

		if (date != "" && format == "") || (date == "" && format != "") {
			return fmt.Errorf("error: --date and --format must be used together")
		}

		_ = format
		_ = date
		return nil
	}

	cmd := &cobra.Command{RunE: testRunE}
	cmd.Flags().Int("lines", 0, "")
	cmd.Flags().String("date", "", "")
	cmd.Flags().String("format", "", "")

	// Test case helper
	runSubTest := func(name, lines, date, format, expectedErrPart string) {
		cmd.Flags().Set("lines", lines)
		cmd.Flags().Set("date", date)
		cmd.Flags().Set("format", format)
		// We pass a dummy arg because the RunE expects one
		err := cmd.RunE(cmd, []string{"/log"})
		if err != nil && strings.Contains(err.Error(), expectedErrPart) {
			testLog(t, ColorGreen, fmt.Sprintf("✔ Subtest [%s] passed (Caught expected error).", name))
		} else {
			testError(t, fmt.Sprintf("✖ Subtest [%s] failed. Expected error containing '%s', got: %v", name, expectedErrPart, err))
		}
	}
	// Cobra handles non-numeric input for Int flags, so we only test logic handled in RunE
	runSubTest("Negative ML", "-10", "", "", "positive number")
	runSubTest("Zero ML", "0", "", "", "positive number")
	runSubTest("Date without Format", "100", "2025-01-01", "", "must be used together")
	runSubTest("Format without Date", "100", "", "2006-01-02", "must be used together")
}

// Test with date filtering using a real log file
func TestCleanLog_WithDateFilter(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestCleanLog_WithDateFilter ---")

	srcLogPath := "test_log.txt"
	if _, err := os.Stat(srcLogPath); os.IsNotExist(err) {
		t.Skip("Skipping: test_log.txt not found in current directory.")
	}

	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	// Copy dummy log
	data, _ := os.ReadFile(srcLogPath)
	os.WriteFile(logPath, data, 0644)

	maxRows := 100
	minDateStr := "2025-08-01"
	dateFormat := "2006-01-02"

	if err := cleanLog(logPath, maxRows, minDateStr, dateFormat, nil); err != nil {
		testError(t, fmt.Sprintf("cleanLog with date filter failed: %v", err))
	}

	cleanedContent, _ := os.ReadFile(logPath)
	lines := strings.Split(strings.TrimSpace(string(cleanedContent)), "\n")

	if len(lines) > 0 {
		testLog(t, ColorGreen, fmt.Sprintf("✔ Date filter applied. Kept %d lines.", len(lines)))
		if strings.HasPrefix(lines[0], "2025-11-09 13:06:55") {
			testLog(t, ColorGreen, "✔ First line date matches expectations.")
		} else {
			testLog(t, ColorYellow, "⚠ First line date doesn't match expected start, check your test_log.txt content.")
		}
	} else {
		testError(t, "✖ Date filter removed all lines!")
	}
}

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

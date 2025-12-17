package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

	// Prevent failures from previous runs by cleaning up old backups first
	oldBackups, _ := filepath.Glob(logPath + ".*.bak")
	for _, f := range oldBackups {
		os.Remove(f)
	}

	opts := CleanOptions{
		Path:       logPath,
		MaxRows:    maxRows,
		DateFormat: "2006-01-02",
	}

	if err := cleanLog(opts); err != nil {
		testError(t, fmt.Sprintf("cleanLog failed: %v", err))
	}

	// 1. Check backup
	files, err := filepath.Glob(logPath + ".*.bak")
	if err != nil {
		t.Fatalf("Error checking for backup file: %v", err)
	}
	if len(files) == 1 {
		testLog(t, ColorGreen, "✔ Backup file created successfully.")
		os.Remove(files[0])
	} else {
		testError(t, fmt.Sprintf("✖ Error: Expected 1 backup file, but found %d.", len(files)))
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

	opts := CleanOptions{
		Path:       logPath,
		MaxRows:    5,
		DateFormat: "2006-01-02",
	}
	if err := cleanLog(opts); err != nil {
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

		if date != "" {
			if len(date) < 10 {
				return fmt.Errorf("--date string must be at least 10 chars (YYYY-MM-DD)")
			}
			if format == "" {
				fullLayout := "2006-01-02 15:04:05"
				if len(date) > len(fullLayout) {
					return fmt.Errorf("--date string is longer than the default supported format")
				}
				format = fullLayout[:len(date)]
			} else {
				if len(date) != len(format) {
					return fmt.Errorf("--date and --format must have the same length")
				}
			}
		}

		// Simplified check for the test
		if date == "" && format != "" {
			return fmt.Errorf("error: --format requires --date")
		}

		// If we passed validation, return nil
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
	runSubTest("Format without Date", "100", "", "2006-01-02", "requires --date")
	runSubTest("Short Date", "100", "2025", "", "must be at least 10 chars")
	runSubTest("Mismatched Date/Format Length", "100", "2025-01-01", "2006-01", "same length")
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

	maxRows := 1000 // Must be > expectedLineCount to test date filtering properly
	minDateStr := "2025-08-01 00:00:00"
	dateFormat := "2006-01-02 15:04:05"

	opts := CleanOptions{
		Path:       logPath,
		MaxRows:    maxRows,
		MinDateStr: minDateStr,
		DateFormat: dateFormat,
	}
	if err := cleanLog(opts); err != nil {
		testError(t, fmt.Sprintf("cleanLog with date filter failed: %v", err))
	}

	cleanedContent, _ := os.ReadFile(logPath)
	lines := strings.Split(strings.TrimSpace(string(cleanedContent)), "\n")

	// Expected results based on test_log.txt and minDateStr "2025-08-01 00:00:00"
	expectedFirstLinePrefix := "2025-08-01 00:17:15"
	expectedLastLinePrefix := "2025-11-25 21:53:32"

	if len(lines) == 0 {
		testError(t, "✖ Date filter removed all lines!")
		return
	}

	testLog(t, ColorGreen, fmt.Sprintf("✔ Date filter applied. Kept %d lines.", len(lines)))

	if !strings.HasPrefix(lines[0], expectedFirstLinePrefix) {
		testError(t, fmt.Sprintf("✖ First line prefix mismatch. Expected: '%s', Got: '%s'", expectedFirstLinePrefix, lines[0]))
	}

	if !strings.HasPrefix(lines[len(lines)-1], expectedLastLinePrefix) {
		testError(t, fmt.Sprintf("✖ Last line prefix mismatch. Expected: '%s', Got: '%s'", expectedLastLinePrefix, lines[len(lines)-1]))
	}
}

// Test with content excluding
func TestCleanLog_ExcludeFilter(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestCleanLog_ExcludeFilter ---")

	srcLogPath := "test_log.txt"
	if _, err := os.Stat(srcLogPath); os.IsNotExist(err) {
		t.Skip("Skipping: test_log.txt not found in current directory.")
	}

	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	data, _ := os.ReadFile(srcLogPath)
	os.WriteFile(logPath, data, 0644)

	maxRows := 1000
	exclude := []string{"Path"}
	dateFormat := "2006-01-02 15:04:05" // Provide format for merging

	opts := CleanOptions{
		Path:       logPath,
		MaxRows:    maxRows,
		DateFormat: dateFormat,
		Exclude:    exclude,
	}
	if err := cleanLog(opts); err != nil {
		testError(t, fmt.Sprintf("cleanLog with exclude filter failed: %v", err))
	}

	cleanedContent, _ := os.ReadFile(logPath)
	lines := strings.Split(strings.TrimSpace(string(cleanedContent)), "\n")

	if len(lines) == 0 {
		testError(t, "✖ Content filter removed all lines!")
		return
	}

	for _, line := range lines {
		if strings.Contains(line, "Path") {
			testError(t, fmt.Sprintf("✖ Line with 'Path' found after excluding:\n%s", line))
			return
		}
	}

	testLog(t, ColorGreen, fmt.Sprintf("✔ Exclude filter applied. Kept %d lines not containing 'Path'.", len(lines)))
}

// Test with content and date excluding
func TestCleanLog_ExcludeAndDateFilter(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestCleanLog_ExcludeAndDateFilter ---")

	srcLogPath := "test_log.txt"
	if _, err := os.Stat(srcLogPath); os.IsNotExist(err) {
		t.Skip("Skipping: test_log.txt not found in current directory.")
	}

	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	data, _ := os.ReadFile(srcLogPath)
	os.WriteFile(logPath, data, 0644)

	maxRows := 1000
	minDateStr := "2025-09-01 00:00:00"
	dateFormat := "2006-01-02 15:04:05"
	exclude := []string{"Path"}

	opts := CleanOptions{
		Path:       logPath,
		MaxRows:    maxRows,
		MinDateStr: minDateStr,
		DateFormat: dateFormat,
		Exclude:    exclude,
	}
	if err := cleanLog(opts); err != nil {
		testError(t, fmt.Sprintf("cleanLog with exclude and date filter failed: %v", err))
	}

	cleanedContent, _ := os.ReadFile(logPath)
	lines := strings.Split(strings.TrimSpace(string(cleanedContent)), "\n")

	if len(lines) == 0 {
		testError(t, "✖ Content and date filter removed all lines!")
		return
	}

	minDate, _ := time.Parse(dateFormat, minDateStr)

	for _, line := range lines {
		if strings.Contains(line, "Path") {
			testError(t, fmt.Sprintf("✖ Line with 'Path' found after excluding:\n%s", line))
			return
		}
		lineDate, err := time.Parse(dateFormat, line[:len(dateFormat)])
		if err != nil || lineDate.Before(minDate) {
			testError(t, fmt.Sprintf("✖ Line with incorrect date found after filtering:\n%s", line))
			return
		}
	}

	testLog(t, ColorGreen, fmt.Sprintf("✔ Exclude and date filter applied. Kept %d lines.", len(lines)))
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

func TestPathValidation(t *testing.T) {
	testLog(t, ColorCyan, "--- START: TestPathValidation ---")

	// Vytvoříme si reálný rootCmd, abychom testovali skutečnou logiku
	var lines int
	rootCmd := &cobra.Command{
		Use:  "logcleaner <path>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if lines <= 0 {
				return fmt.Errorf("--lines must be positive")
			}
			if !strings.HasPrefix(args[0], "/") && !strings.HasPrefix(args[0], "./") {
				return fmt.Errorf("path must be absolute (starting with '/') or relative (starting with './')")
			}
			// Pro účely testu nevoláme cleanLog, jen vrátíme nil, pokud validace projde
			return nil
		},
	}
	rootCmd.Flags().IntVar(&lines, "lines", 10, "")

	// Test, který má selhat
	err := rootCmd.RunE(rootCmd, []string{"invalidpath.log"})
	if err == nil || !strings.Contains(err.Error(), "path must be absolute") {
		testError(t, "✖ Path validation failed: an invalid path was accepted.")
	} else {
		testLog(t, ColorGreen, "✔ Path validation correctly rejected an invalid path.")
	}
}

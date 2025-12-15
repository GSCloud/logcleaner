package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// CLEANLOG - CONTAINS MAIN LOGIC

// cleanLog performs the core log file trimming and filtering operations.
// path: file path to the log
// maxRows: max number of rows to keep
// dateFormat: date string for filtration. If invalid, date filtering is skipped.
func cleanLog(path string, maxRows int, dateFormat string) error {
	// 1. Create a backup file with a timestamp
	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("2006-01-02 15:04:05"))
	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("error backing up log %s to %s: %w", path, backupPath, err)
	}

	// 2. Open backup for reading
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("error opening backup file %s: %w", backupPath, err)
	}
	defer file.Close()

	// 3. Read all lines from the backup file
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading backup file: %w", err)
	}

	// 4. Handle empty log file
	if len(allLines) == 0 {
		fmt.Printf("Log %s is empty.\n", path)
		// Recreate the original file as an empty file
		if _, err := os.Create(path); err != nil {
			return fmt.Errorf("error creating empty log file: %w", err)
		}
		return nil
	}

	// 5. Date Filtering Logic
	const logDateFormat = "2006-01-02 15:04:05"
	minDate, dateParseErr := time.Parse(logDateFormat, dateFormat)

	var filteredLines []string
	lastKeptIndex := -1 // Index of the last element added to filteredLines

	if dateParseErr != nil {
		// If dateFormat is not a valid date (like "irrelevant"), skip date filtering.
		// No merging of lines is necessary when filtering is skipped, as all lines are kept.
		filteredLines = allLines
		fmt.Printf("Date filter skipped: '%s' is not a valid date format. Trimming only by line count.\n", dateFormat)
	} else {
		// Date filtering is active.
		for _, line := range allLines {
			var err error
			var lineDate time.Time
			isMainLogLine := false

			// Check if the line is long enough to contain the timestamp
			if len(line) >= len(logDateFormat) {
				lineDateStr := line[:len(logDateFormat)]
				lineDate, err = time.Parse(logDateFormat, lineDateStr)

				if err == nil {
					// Successfully parsed the date from the line.
					if !lineDate.Before(minDate) {
						// It's a main log line and it passed the date filter.
						isMainLogLine = true
					}
					// If the date is too old, isMainLogLine remains false, and it won't be kept.
				}
				// If parsing fails (err != nil), isMainLogLine remains false.
			}

			if isMainLogLine {
				// Case 1: Valid main log line, passed date filter.
				filteredLines = append(filteredLines, line)
				lastKeptIndex = len(filteredLines) - 1
			} else if lastKeptIndex != -1 {
				// Case 2: Continuation line (no valid date OR failed date check) that follows a kept line.
				// Append this line to the preceding kept line, using a single space as separator.
				// The index remains the same as we modify the last element, not add a new one.
				filteredLines[lastKeptIndex] += " " + line // <-- Changed "\n" to " "
			}
			// Case 3: Line that failed date filter and is not following a kept line is discarded.
		}

		// The number of "lines" reported here is the number of grouped entries.
		fmt.Printf("Date filter applied: keeping entries newer than %s. Original lines: %d, kept entries: %d.\n", dateFormat, len(allLines), len(filteredLines))
	}

	// 6. Trim the filtered entries (which may contain multiline content) to max rows
	var finalLines []string
	if len(filteredLines) > maxRows {
		// Keep only the last 'maxRows' entries (each entry can be multiline).
		finalLines = filteredLines[len(filteredLines)-maxRows:]
	} else {
		// No trimming needed if entries are within the limit
		finalLines = filteredLines
	}

	// 7. Write to a new temporary file (atomic move preparation)
	// Create temp file in the same directory as the original log
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	writer := bufio.NewWriter(tempFile)
	// IMPORTANT: Use Fprint/Fprintln. Fprintln appends the newline character automatically.
	// Note: Since entries now use space separation, Fprintln will output single-line log entries.
	for _, line := range finalLines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			tempFile.Close() // Close file before returning
			return fmt.Errorf("error writing to temporary file: %w", err)
		}
	}
	writer.Flush()   // Ensure all buffered data is written to the file
	tempFile.Close() // Close the file to release the handle

	// 8. Atomic move - temporary file replaces the original log
	if err = os.Rename(tempPath, path); err != nil {
		fmt.Printf("Error when renaming temporary file, restoring backup: %v\n", err)
		os.Rename(backupPath, path) // try to fix it
		return fmt.Errorf("atomic move failed: %v", err)
	}
	fmt.Printf("Log %s successfully purged. Original backup: %s. New log has %d grouped entries.\n", path, backupPath, len(finalLines))
	return nil
}

// HelpDisplayedError is an empty structure that we use to signal that help has been displayed and the program should exit with an exit code 0
type HelpDisplayedError struct{}

func (e *HelpDisplayedError) Error() string {
	return ""
}

// main entry point
func main() {
	var rootCmd = &cobra.Command{
		Short:   "Minimalistic tool for rotating and cleaning logs.",
		Long:    "LOGCLEANER is designed to maintain optimal log file size by precisely truncating a specified log file. It retains only the desired number of the most recent lines, allowing filtering up to a designated date in the past. This makes it easy to drop outdated log entries and ensure your logs remain current and manageable.\n",
		Use:     "\tlogcleaner <log_path> <max_lines> <date_format>",
		Example: "\tlogcleaner /path/messages.txt 500 \"2025-01-02 15:04:05\"",
		// silencing Cobra parameters
		SilenceErrors: true,
		SilenceUsage:  true,

		// manual arguments tests
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				cmd.Help()                   // show help
				return &HelpDisplayedError{} // our own bug
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			rowsStr := args[1]
			format := args[2]

			// Validate max_lines is a number
			rows, err := strconv.Atoi(rowsStr)
			if err != nil {
				// Error: Bad argument format
				return fmt.Errorf("error: second argument 'max_lines' must be a number, but was: %s", rowsStr)
			}
			// Validate max_lines is positive
			if rows <= 0 {
				// Error: Invalid argument value
				return fmt.Errorf("error: maximum number of rows must be a positive number")
			}

			// run main logic
			if err := cleanLog(path, rows, format); err != nil {
				return fmt.Errorf("Error while cleaning the log file: %w", err)
			}
			return nil
		},
	}

	// run Cobra
	if err := rootCmd.Execute(); err != nil {
		// Check if the error is our own signal error (HelpDisplayedError)
		if _, ok := err.(*HelpDisplayedError); ok {
			os.Exit(0) // Exit with success code 0
		}

		// Other errors (e.g., I/O error) exit with code 1.
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

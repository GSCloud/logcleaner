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

// path: file path to the log
// maxRows: max number of rows to export
// dateFormat: date format for filtration (TBD, just a placeholder now)
func cleanLog(path string, maxRows int, dateFormat string) error {
	backupPath := path + "." + time.Now().Format("2006-01-02 15:04:05") + ".bak"
	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("Error backing up log %s to %s: %w", path, backupPath, err)
	}

	// open backup for reading
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("Error opening backup file %s: %w", backupPath, err)
	}
	defer file.Close()

	// read all lines
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading backup file: %w", err)
	}

	// empty log
	if len(lines) == 0 {
		fmt.Printf("Log %s is empty.\n", path)
		_, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("Error creating empty log file: %w", err)
		}
		return nil
	}

	// Trim lines to max rows
	var trimmedLines []string
	if len(lines) > maxRows {
		trimmedLines = lines[len(lines)-maxRows:]
	} else {
		trimmedLines = lines // No trimming needed if lines are within the limit
	}

	// dateFormat is not used right now, just needed as a parameter

	// new temp file
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("Error creating temporary file: %w", err)
	}
	tempPath := tempFile.Name() // Storing path for the final rename

	// write the trimmed lines
	writer := bufio.NewWriter(tempFile)
	for _, line := range trimmedLines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			tempFile.Close() // Close file before returning
			return fmt.Errorf("Error writing to temporary file: %w", err)
		}
	}
	writer.Flush()   // Ensure all buffered data is written to the file
	tempFile.Close() // Close the file to release the handle

	// atomic move - temp to the origin
	if err = os.Rename(tempPath, path); err != nil {
		fmt.Printf("Error when renaming temporary file, restoring backup: %v\n", err)
		os.Rename(backupPath, path) // try to fix it
		return fmt.Errorf("atomic move failed: %v", err)
	}
	fmt.Printf("Log %s successfully purged. Original backup: %s. New log has %d lines. Format used: %s\n", path, backupPath, len(trimmedLines), dateFormat)
	return nil
}

// HelpDisplayedError is an empty structure that we use to signal that help has been displayed and the program should exit with an exit code 0
type HelpDisplayedError struct{}

func (e *HelpDisplayedError) Error() string {
	return ""
}

// main
func main() {
	var rootCmd = &cobra.Command{
		Short:   "Minimalistic tool for rotating and cleaning logs.",
		Long:    "LogCleaner truncates a given log file to the exact specified number of last lines up to a date in the past.\n",
		Use:     "\tlogcleaner [log_path] [max_lines] [date_format]",
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

			rows, err := strconv.Atoi(rowsStr)
			if err != nil {
				// Error: Bad argument format
				return fmt.Errorf("error: second argument 'max_lines' must be a number, but was: %s", rowsStr)
			}
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

		// CHECK IF THE ERROR IS NOT OUR OWN SIGNAL ERROR

		// If it is our signal (HelpDisplayedError), we exit with code 0 (success).
		// All other errors (e.g. I/O error in cleanLog or bad line format) exit with code 1.
		if _, ok := err.(*HelpDisplayedError); ok {
			os.Exit(0)
		}

		// other errors
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

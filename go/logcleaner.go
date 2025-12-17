package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// shell colors
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorReset  = "\033[0m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// app version
const (
	VERSION = "0.0.1"
)

// copyFile helper - duplicate a file safely
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// rollback ensures that the original log is restored safely using a double-backup strategy
func rollback(originalPath, backupPath string) {
	fmt.Printf("%sRollback initiated: Restoring %s from %s...%s\n", ColorYellow, originalPath, backupPath, ColorReset)

	doubleBak := backupPath + ".bak"

	// 1. Create .bak.bak (safety copy of our only good backup)
	if err := copyFile(backupPath, doubleBak); err != nil {
		fmt.Printf("%sRollback critical failure: Could not create safety copy %s: %v%s\n", ColorRed, doubleBak, err, ColorReset)
		return
	}

	// 2. Move .bak to original position (this might consume the .bak)
	if err := os.Rename(backupPath, originalPath); err != nil {
		fmt.Printf("%sRollback failed: Could not move backup back to %s: %v%s\n", ColorRed, originalPath, err, ColorReset)
		return
	}

	// 3. Move .bak.bak to .bak to satisfy the "always keep a backup" rule
	if err := os.Rename(doubleBak, backupPath); err != nil {
		fmt.Printf("%sRollback warning: Original restored, but failed to preserve .bak from .bak.bak: %v%s\n", ColorYellow, err, ColorReset)
	}
}

// CLEANLOG - CONTAINS MAIN LOGIC
func cleanLog(path string, maxRows int, dateFormat string) error {
	// 1. Create a backup file with a timestamp
	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("2006-01-02-15-04-05"))
	if err := copyFile(path, backupPath); err != nil {
		return fmt.Errorf("error creating backup of %s to %s: %w", path, backupPath, err)
	}

	// Flag to track success for potential rollback
	var operationFailed bool
	defer func() {
		if operationFailed {
			rollback(path, backupPath)
		}
	}()

	// 2. Open backup for reading
	file, err := os.Open(backupPath)
	if err != nil {
		operationFailed = true
		return fmt.Errorf("error opening backup file %s: %w", backupPath, err)
	}
	defer file.Close()

	// 3. Read all lines
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		operationFailed = true
		return fmt.Errorf("error reading backup file: %w", err)
	}

	if len(allLines) == 0 {
		fmt.Printf("Log %s is empty.\n", path)
		return nil
	}

	// 4. Date Filtering Logic
	const logDateFormat = "2006-01-02 15:04:05"
	minDate, dateParseErr := time.Parse(logDateFormat, dateFormat)

	var filteredLines []string
	lastKeptIndex := -1

	if dateParseErr != nil {
		filteredLines = allLines
		fmt.Printf("Date filter skipped: '%s' is not valid. Trimming only by line count.\n", dateFormat)
	} else {
		for _, line := range allLines {
			var err error
			var lineDate time.Time
			isMainLogLine := false

			if len(line) >= len(logDateFormat) {
				lineDateStr := line[:len(logDateFormat)]
				lineDate, err = time.Parse(logDateFormat, lineDateStr)
				if err == nil {
					if !lineDate.Before(minDate) {
						isMainLogLine = true
					}
				}
			}

			if isMainLogLine {
				filteredLines = append(filteredLines, line)
				lastKeptIndex = len(filteredLines) - 1
			} else if lastKeptIndex != -1 {
				filteredLines[lastKeptIndex] += " " + line
			}
		}
	}

	// 5. Trim to max_lines
	var finalLines []string
	if len(filteredLines) > maxRows {
		finalLines = filteredLines[len(filteredLines)-maxRows:]
	} else {
		finalLines = filteredLines
	}

	// 6. Write to temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		operationFailed = true
		return fmt.Errorf("error creating temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	writer := bufio.NewWriter(tempFile)
	for _, line := range finalLines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			tempFile.Close()
			operationFailed = true
			return fmt.Errorf("error writing to temporary file: %w", err)
		}
	}
	writer.Flush()
	tempFile.Close()

	// 7. Atomic move
	if err = os.Rename(tempPath, path); err != nil {
		operationFailed = true
		return fmt.Errorf("atomic move failed: %v", err)
	}

	fmt.Printf("%sLog %s purged. Backup kept at: %s. Entries: %d.%s\n", ColorGreen, path, backupPath, len(finalLines), ColorReset)
	return nil
}

type HelpDisplayedError struct{}

func (e *HelpDisplayedError) Error() string { return "" }

func main() {
	var rootCmd = &cobra.Command{
		Short:         "LOGCLEANER - a minimalistic tool for truncating and cleaning logs.",
		Long:          "LOGCLEANER is designed to maintain optimal log file size by precisely truncating a specified log file by lines, datetime stamp and content filtering.",
		Use:           ColorBold + "\nlogcleaner <log_path> <max_lines> <date_format>" + ColorReset,
		Example:       ColorBold + "\nlogcleaner ./messages.txt 5000 \"2025-01-15 15:04:05\"" + ColorReset,
		Version:       VERSION,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 3 {
				cmd.Help()
				return &HelpDisplayedError{}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			rowsStr := args[1]
			format := args[2]

			rows, err := strconv.Atoi(rowsStr)
			if err != nil {
				return fmt.Errorf("error: 'max_lines' must be a number")
			}
			if rows <= 0 {
				return fmt.Errorf("error: maximum number of lines must be positive")
			}

			return cleanLog(path, rows, format)
		},
	}

	if err := rootCmd.Execute(); err != nil {
		if _, ok := err.(*HelpDisplayedError); ok {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%s%v%s\n", ColorRed, err, ColorReset)
		os.Exit(1)
	}
}

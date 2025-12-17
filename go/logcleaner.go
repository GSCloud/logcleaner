package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

const VERSION = "0.1.29"

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

func rollback(originalPath, backupPath string) {
	fmt.Printf("%srollback: restoring %s%s\n", ColorBlue, originalPath, ColorReset)
	os.Rename(backupPath, originalPath)
}

func cleanLog(path string, maxRows int, minDateStr string, dateFormat string, filter []string) error {
	if minDateStr == "" {
		dateFormat = ""
	}

	fmt.Printf("%s[INFO] Cleaning log: %s (max lines: %d)%s\n", ColorBold, path, maxRows, ColorReset)

	// 1. Backup
	backupPath := fmt.Sprintf("%s.%s.bak", path, time.Now().Format("2006-01-02-15-04-05"))
	if err := copyFile(path, backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	var operationFailed bool
	defer func() {
		if operationFailed {
			rollback(path, backupPath)
		}
	}()

	file, err := os.Open(backupPath)
	if err != nil {
		operationFailed = true
		return err
	}
	defer file.Close()

	var rawLines []string
	scanner := bufio.NewScanner(file)
	// Support for long lines up to 10MB
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		rawLines = append(rawLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("%s[ERROR] Read failed: %v%s\n", ColorRed, err, ColorReset)
		operationFailed = true
		return err
	}

	if len(rawLines) == 0 {
		return nil
	}

	// 2. Multiline grouping
	var processedLines []string
	formatLen := len(dateFormat)

	if formatLen > 0 {
		for _, line := range rawLines {
			isNewEntry := false
			if len(line) >= formatLen {
				prefix := line[:formatLen]
				if _, err := time.Parse(dateFormat, prefix); err == nil {
					isNewEntry = true
				}
			}

			if isNewEntry || len(processedLines) == 0 {
				processedLines = append(processedLines, line)
			} else {
				lastIdx := len(processedLines) - 1
				// Join with space to keep it single-line in the file
				processedLines[lastIdx] = processedLines[lastIdx] + " " + line
			}
		}
	} else {
		processedLines = rawLines
	}

	// 3. Filter by content
	if len(filter) > 0 {
		var filtered []string
		for _, entry := range processedLines {
			match := false
			for _, f := range filter {
				if strings.Contains(entry, f) {
					match = true
					break
				}
			}
			if match {
				filtered = append(filtered, entry)
			}
		}
		processedLines = filtered
	}

	// 4. Filter by date threshold
	if minDateStr != "" && formatLen > 0 {
		minDate, parseErr := time.Parse(dateFormat, minDateStr)
		if parseErr != nil && len(minDateStr) >= 10 {
			minDate, _ = time.Parse("2006-01-02", minDateStr[:10])
		}

		var dateFiltered []string
		for _, entry := range processedLines {
			if len(entry) >= formatLen {
				prefix := entry[:formatLen]
				d, err := time.Parse(dateFormat, prefix)
				if err == nil {
					if !d.Before(minDate) {
						dateFiltered = append(dateFiltered, entry)
					}
				} else {
					dateFiltered = append(dateFiltered, entry)
				}
			} else {
				dateFiltered = append(dateFiltered, entry)
			}
		}
		processedLines = dateFiltered
	}

	// 5. Trimming (Applied AFTER date filtering)
	if len(processedLines) > maxRows {
		processedLines = processedLines[len(processedLines)-maxRows:]
	}

	// 6. Final Write
	tempPath := path + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		operationFailed = true
		return err
	}

	writer := bufio.NewWriter(tempFile)
	for _, entry := range processedLines {
		writer.WriteString(entry)
		writer.WriteString("\n")
	}
	writer.Flush()
	tempFile.Close()

	if err := os.Rename(tempPath, path); err != nil {
		operationFailed = true
		return err
	}

	fmt.Printf("%s[OK] Log updated. Entries: %d%s\n", ColorGreen, len(processedLines), ColorReset)
	return nil
}

func main() {
	var (
		lines  int
		date   string
		format string
		filter []string
	)

	var rootCmd = &cobra.Command{
		Short: ColorBold + "LOGCLEANER" + ColorReset + " - minimalistic log tool.",
		Use:   "logcleaner <path> --lines <n>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if lines <= 0 {
				return fmt.Errorf("--lines must be positive")
			}
			return cleanLog(args[0], lines, date, format, filter)
		},
	}

	rootCmd.Flags().IntVar(&lines, "lines", 0, "Max entries to keep")
	rootCmd.Flags().StringVar(&date, "date", "", "Start date threshold (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&format, "format", "", "Date layout in log")
	rootCmd.Flags().StringSliceVar(&filter, "filter", []string{}, "Keep only entries containing these strings")
	rootCmd.MarkFlagRequired("lines")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

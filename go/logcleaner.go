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

const VERSION = "0.1.31"

// custom help template
var helpTemplate = fmt.Sprintf(`
%s%s v%s%s
{{.Short}}

%sUsage:%s
  {{.UseLine}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if .HasAvailableLocalFlags}}

%sFlags:%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%sGlobal Flags:%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

%sExamples:%s
{{.Example}}{{end}}
`, ColorBold, "LOGCLEANER", VERSION, ColorReset, ColorYellow, ColorReset, ColorYellow, ColorReset, ColorYellow, ColorReset, ColorYellow, ColorReset)

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

type CleanOptions struct {
	Path       string
	MaxRows    int
	MinDateStr string
	DateFormat string
	Exclude    []string
}

func cleanLog(opts CleanOptions) error {
	if opts.MinDateStr == "" {
		opts.DateFormat = ""
	}

	fmt.Printf("%s[INFO] Cleaning log: %s (max lines: %d)%s\n", ColorBold, opts.Path, opts.MaxRows, ColorReset)

	// 1. Backup
	backupPath := fmt.Sprintf("%s.%s.bak", opts.Path, time.Now().Format("2006-01-02-15-04-05"))
	if err := copyFile(opts.Path, backupPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	var operationFailed bool
	defer func() {
		if operationFailed {
			rollback(opts.Path, backupPath)
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
	formatLen := len(opts.DateFormat)

	if formatLen > 0 {
		for _, line := range rawLines {
			isNewEntry := false
			if len(line) >= formatLen {
				prefix := line[:formatLen]
				if _, err := time.Parse(opts.DateFormat, prefix); err == nil {
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
	if len(opts.Exclude) > 0 {
		var filtered []string
		for _, entry := range processedLines {
			match := false
			for _, f := range opts.Exclude {
				if strings.Contains(entry, f) {
					match = true
					break
				}
			}
			if !match {
				filtered = append(filtered, entry)
			}
		}
		processedLines = filtered
	}

	// 4. Filter by date threshold
	if opts.MinDateStr != "" && formatLen > 0 {
		minDate, err := time.Parse(opts.DateFormat, opts.MinDateStr)
		if err != nil {
			return fmt.Errorf("could not parse --date '%s' with format '%s': %w", opts.MinDateStr, opts.DateFormat, err)
		}

		var dateFiltered []string
		for _, entry := range processedLines {
			if len(entry) >= formatLen {
				prefix := entry[:formatLen]
				d, err := time.Parse(opts.DateFormat, prefix)
				if err == nil {
					if !d.Before(minDate) {
						dateFiltered = append(dateFiltered, entry)
					}
				}
			}
			// If an entry doesn't start with a valid date, it's discarded when date filtering is active.
		}
		processedLines = dateFiltered
	}

	// 5. Trimming (Applied AFTER date filtering)
	if len(processedLines) > opts.MaxRows {
		processedLines = processedLines[len(processedLines)-opts.MaxRows:]
	}

	// 6. Final Write
	tempPath := opts.Path + ".tmp"
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

	if err := os.Rename(tempPath, opts.Path); err != nil {
		operationFailed = true
		return err
	}

	fmt.Printf("%s[OK] Log updated. Entries: %d%s\n", ColorGreen, len(processedLines), ColorReset)
	return nil
}

func main() {
	var (
		lines   int
		date    string
		format  string
		exclude []string
	)

	var rootCmd = &cobra.Command{
		Short:         ColorBold + "LOGCLEANER" + ColorReset + " - a fast log cleaner and optimizer",
		Long:          ColorBold + "LOGCLEANER" + ColorReset + " - a fast log cleaner and optimizer\n\nA specialized utility to truncate and filter text log files.\nIt merges multiline entries, filters by date and content, and keeps a specified number of the last lines.",
		Use:           "logcleaner <path> --lines <number> [--date <date> --format <layout>] [--exclude <string>]",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       VERSION,
		RunE: func(cmd *cobra.Command, args []string) error {
			if lines <= 0 {
				return fmt.Errorf("--lines must be positive")
			}
			if !strings.HasPrefix(args[0], "/") && !strings.HasPrefix(args[0], "./") {
				return fmt.Errorf("path must be absolute (starting with '/') or relative (starting with './')")
			}
			// If date is provided, it must be at least 10 chars long.
			// If format is not provided, derive it from date length.
			if date != "" {
				if len(date) < 10 {
					return fmt.Errorf("--date string must be at least 10 chars (YYYY-MM-DD)")
				}
				if format == "" {
					fullLayout := "2006-01-02 15:04:05"
					format = fullLayout[:len(date)]
				} else {
					if len(date) != len(format) {
						return fmt.Errorf("--date and --format must have the same length")
					}
				}
			}
			return cleanLog(CleanOptions{
				Path:       args[0],
				MaxRows:    lines,
				MinDateStr: date,
				DateFormat: format,
				Exclude:    exclude,
			})
		},
	}

	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.Flags().IntVar(&lines, "lines", 0, "max entries to keep")
	rootCmd.Flags().StringVar(&date, "date", "", "start date threshold (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&format, "format", "", "date layout in log")
	rootCmd.Flags().StringSliceVar(&exclude, "exclude", []string{}, "exclude entries containing these strings")
	rootCmd.MarkFlagRequired("lines")
	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(rootCmd.Short)
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", ColorRed, err, ColorReset)
		fmt.Fprintf(os.Stderr, "Use \"--help\" for more information.\n")
		os.Exit(1)
	}
}

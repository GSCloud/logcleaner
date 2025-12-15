# LOGCLEANER

## Your fast server logs cleaner and optimizer

LogCleaner truncates a given log file to the exact specified number of last lines up to a date in the past.

Usage:
        logcleaner [log_path] [max_lines] [date_format] [flags]

Examples:
        logcleaner /path/messages.txt 500 "2025-01-02 15:04:05"

Flags:
  -h, --help   help for         logcleaner
  
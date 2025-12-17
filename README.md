# 🧹 LOGCLEANER

## Fast log cleaner and optimizer for high-performance environments

**LOGCLEANER** is a specialized utility designed to truncate text log files to an exact number of last lines, synchronized with a specific point in time. It optimizes log readability by merging multiline entries (lines without a date) and provides powerful filtering capabilities.

## ✨ Key Features

**Precision Truncation**: Keeps exactly the last $N$ lines up to a specific date/time.

**Log Optimization**: Automatically appends date-less lines to the previous timestamped entry.

**Flexible Filtering**: Optional string stub filtering for focused analysis (e.g., keeping only "ERROR").

**Safety First**: Automatically creates a .bak file and attempts recovery in case of failure.

## 🚀 Usage

        logcleaner <log_path> ml=<max_lines> df=<date_format> [fs=<filter_stub> ...]

## Parameters

| Argument | Description | Requirement
| --- | --- | ---
| **log_path | Absolute or relative path to the log file. | Required
| **ml** | Maximum number of lines to retain. | Required
| **df** | Date Format: Golang style time format [time#Layout](https://pkg.go.dev/time#Layout). | Required
| **fs** | Filter Stub: Only keep lines containing this string (can be used multiple times). | Optional

## 💡 Examples

Standard cleanup (last 1500 lines):

        logcleaner /var/log/apache2/messages.txt ml=1500 df="2025-01-20"

Cleanup with specific timestamp:

        logcleaner /var/log/apache2/messages.txt ml=1500 df="2025-01-20 15:04:05"

Cleanup with ERROR filtering:

        logcleaner /var/log/apache2/messages.txt ml=1500 df="2025-01-20 15:04:05" fs="ERROR"

## 🛠 Commands & Flags

| Flag | Action
| --- | ---
| **-h, --help** | Display help message.
| **-v, --version** | Display version and build information.

## 📝 Technical Notes

**Input Validation**: The *log_path* must explicitly start with **/** (absolute) or **./** (relative).

**Format**: The *date_format* must strictly follow the Go reference layout.

**Data Integrity**: The tool always creates a backup. If a process is interrupted, it attempts to restore the original file from the .bak copy.

---

Author: Fred Brooker 💌 <git@gscloud.cz> ⛅️ GS Cloud Ltd. [https://gscloud.cz](https://gscloud.cz)

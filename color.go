package main

import (
	"fmt"
	"os"
	"runtime"
)

var (
	colorEnabled = isTerminal()
	verboseMode  = false
)

func isTerminal() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	bold    = "\033[1m"
)

func colorize(color, text string) string {
	if !colorEnabled {
		return text
	}
	return color + text + reset
}

func logInfo(format string, args ...any) {
	if !verboseMode {
		return
	}
	fmt.Fprintf(os.Stderr, colorize(cyan, "[*] ")+format+"\n", args...)
}

func logSuccess(format string, args ...any) {
	if !verboseMode {
		return
	}
	fmt.Fprintf(os.Stderr, colorize(green, "[+] ")+format+"\n", args...)
}


func logError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorize(red, "[-] ")+format+"\n", args...)
}

func logFatal(format string, args ...any) {
	logError(format, args...)
	os.Exit(1)
}

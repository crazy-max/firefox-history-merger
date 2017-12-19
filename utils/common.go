package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// CheckIfError should be used to natively panics if an error is not nil
func CheckIfError(err error) {
	if err == nil {
		return
	}
	ErrorExit(err.Error())
}

// ErrorExit should be used to natively panics
func ErrorExit(format string, args ...interface{}) {
	Error(format, args...)
	os.Exit(1)
}

// Error to describe an error
func Error(format string, args ...interface{}) {
	fmt.Printf("\n\x1b[31;1mERROR: %s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// Info should be used to describe the example commands that are about to run
func Info(format string, args ...interface{}) {
	fmt.Printf("\n\x1b[34;1mINFO: %s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// Warning should be used to display a warning
func Warning(format string, args ...interface{}) {
	fmt.Printf("\n\x1b[33;1mWARN: %s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// Debug to output additionnal information
func Debug(format string, args ...interface{}) {
	fmt.Printf("\n\x1b[36;1mDEBUG: %s\x1b[0m\n", fmt.Sprintf(format, args...))
}

// PrintPretty print of struct or slice
func PrintPretty(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

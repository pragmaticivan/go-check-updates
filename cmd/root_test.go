package cmd

import (
	"bytes"
	"testing"
)

func TestExecute_Help(t *testing.T) {
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"--help"})

	// Execute should not os.Exit on success.
	Execute()
}

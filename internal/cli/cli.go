package cli

import "github.com/coreyvan/chirp/internal/cli/commands"

// Version is the CLI version and can be overridden at build time.
var Version = "dev"

// Execute runs the root command.
func Execute() error {
	commands.Version = Version
	return commands.Execute()
}

// ExitCode returns the CLI process exit code for the provided error.
func ExitCode(err error) int {
	return commands.ExitCode(err)
}


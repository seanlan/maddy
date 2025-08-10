package mailcoincli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CobraContext provides a compatibility layer between cobra commands and the existing
// CLI functions that expect urfave/cli context
type CobraContext struct {
	cmd  *cobra.Command
	args []string
}

// NewCobraContext creates a new context adapter for cobra commands
func NewCobraContext(cmd *cobra.Command, args []string) *CobraContext {
	return &CobraContext{
		cmd:  cmd,
		args: args,
	}
}

// String returns the value of a string flag
func (c *CobraContext) String(name string) string {
	val, _ := c.cmd.Flags().GetString(name)
	return val
}

// StringSlice returns the value of a string slice flag
func (c *CobraContext) StringSlice(name string) []string {
	val, _ := c.cmd.Flags().GetStringSlice(name)
	return val
}

// Bool returns the value of a boolean flag
func (c *CobraContext) Bool(name string) bool {
	val, _ := c.cmd.Flags().GetBool(name)
	return val
}

// Int returns the value of an int flag
func (c *CobraContext) Int(name string) int {
	val, _ := c.cmd.Flags().GetInt(name)
	return val
}

// IsSet returns true if the flag was set
func (c *CobraContext) IsSet(name string) bool {
	return c.cmd.Flags().Changed(name)
}

// Args returns the command arguments
func (c *CobraContext) Args() Args {
	return &CobraArgs{args: c.args}
}

// NArg returns the number of arguments
func (c *CobraContext) NArg() int {
	return len(c.args)
}

// Path returns the value of a path flag (same as String for our purposes)
func (c *CobraContext) Path(name string) string {
	return c.String(name)
}

// CobraArgs provides argument access similar to cli.Context.Args()
type CobraArgs struct {
	args []string
}

// Args interface to match urfave/cli behavior
type Args interface {
	First() string
	Get(n int) string
	Len() int
}

func (a *CobraArgs) First() string {
	if len(a.args) > 0 {
		return a.args[0]
	}
	return ""
}

func (a *CobraArgs) Get(n int) string {
	if n < len(a.args) {
		return a.args[n]
	}
	return ""
}

func (a *CobraArgs) Len() int {
	return len(a.args)
}

// Exit simulates cli.Exit behavior for cobra commands
type ExitError struct {
	Message string
	Code    int
}

func (e *ExitError) Error() string {
	return e.Message
}

// Exit creates an ExitError to simulate cli.Exit
func Exit(message string, code int) error {
	fmt.Fprintln(os.Stderr, message)
	return &ExitError{Message: message, Code: code}
}
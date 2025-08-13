package mailchatcli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var stdinScanner = bufio.NewScanner(os.Stdin)

// Confirmation prompts the user for a yes/no confirmation
func Confirmation(prompt string, def bool) bool {
	selection := "y/N"
	if def {
		selection = "Y/n"
	}

	fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, selection)
	if !stdinScanner.Scan() {
		fmt.Fprintln(os.Stderr, stdinScanner.Err())
		return false
	}

	switch strings.ToLower(strings.TrimSpace(stdinScanner.Text())) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return def
	}
}

// ReadPassword prompts the user for a password with hidden input
func ReadPassword(prompt string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	
	// Check if stdin is a terminal
	if !term.IsTerminal(int(syscall.Stdin)) {
		// If not a terminal, read plaintext (for scripts/tests)
		if !stdinScanner.Scan() {
			return "", stdinScanner.Err()
		}
		return stdinScanner.Text(), nil
	}

	// Read password with hidden input
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr) // Print newline after password input
	
	if err != nil {
		return "", err
	}
	
	return string(password), nil
}
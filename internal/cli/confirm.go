package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmAction asks the user for confirmation with a y/N prompt
// Returns true if the user confirms with 'y' or 'yes', false otherwise
func ConfirmAction(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}


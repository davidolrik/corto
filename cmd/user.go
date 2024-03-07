package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/davidolrik/corto/internal/core"
	"github.com/davidolrik/corto/internal/services"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewUserCommand() *cobra.Command {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		Long:  "Manage users",
	}

	var password string
	createCmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a user",
		Long:  "Create a user with the given username",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				p, err := readPassword()
				if err != nil {
					return fmt.Errorf("reading password: %w", err)
				}
				password = p
			}

			db := core.NewDatabase()
			defer db.Close()

			userService := services.NewUserService(slog.Default(), db)
			user, err := userService.CreateUser(cmd.Context(), args[0], password)
			if err != nil {
				return err
			}

			fmt.Printf("Created user %q with public ID %s\n", user.Username, user.PublicID)
			return nil
		},
	}
	createCmd.Flags().StringVar(&password, "password", "", "Password for the new user (prompted when omitted)")
	userCmd.AddCommand(createCmd)

	return userCmd
}

// readPassword prompts for a password without echoing when stdin is a
// terminal, and reads a single line otherwise so passwords can be piped in.
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, "Password: ")
		password, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		return string(password), err
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

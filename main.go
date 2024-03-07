package main

import (
	"fmt"
	"os"

	"github.com/davidolrik/corto/cmd"
	"github.com/jedib0t/go-pretty/v6/text"
)

func main() {
	text.EnableColors()
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

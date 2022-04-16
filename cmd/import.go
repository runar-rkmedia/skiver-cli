/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"reflect"

	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import from local i18n-file",
	Run: func(cmd *cobra.Command, args []string) {
		if CLI.Import.Source == "" {
			l.Fatal().Msg("Source is required")
		}
		api := requireApi(true)
		l.Debug().Str("path", CLI.Import.Source).Msg("importing")
		source, exists := getFile(CLI.Import.Source)
		if !exists {
			l.Fatal().Str("path", CLI.Import.Source).Msg("File not found")
		}
		err := api.Import(CLI.Project, "i18n", CLI.Locale, source)
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to import")
		}
		l.Info().Msg("Successful import")
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	s := reflect.TypeOf(CLI.Import)
	for _, v := range []string{"Source"} {
		mustSetVar(s, v, importCmd, "import.")
	}
}

/*
Copyright © 2022 Runar Kristoffersen

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"io"
	"os"
	"reflect"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate files for the project",
	Run: func(cmd *cobra.Command, args []string) {
		api := requireApi(false)
		var w io.Writer
		format := CLI.Generate.Format
		locale := CLI.Locale
		if locale == "" {
			l.Fatal().Msg("Locale is required")
		}
		if CLI.Project == "" {
			l.Fatal().Msg("Project is required")
		}
		if CLI.Generate.Format == "" {
			l.Fatal().Msg("Format is required")
		}
		ll := l.Debug().Str("project", CLI.Project).
			Str("format", format)
		if CLI.Generate.Path != "" {
			// We ensure the file exists before start. We overwrite the file with data.
			// In case of an error, we should not write empty data.
			// This is what would happen if opening the file with os.O_WRONLY|os.O_CREATE|os.O_TRUNC
			err := touchFile(CLI.Generate.Path)
			if err != nil {
				l.Fatal().Err(err).Str("path", CLI.Generate.Path).Msg("Failed to open/create file")
				return
			}
			outfile, err := os.OpenFile(CLI.Generate.Path, os.O_WRONLY, os.ModePerm)
			if err != nil {
				l.Fatal().Err(err).Str("path", CLI.Generate.Path).Msg("Failed to open file for writing")
				return
			}
			defer outfile.Close()
			w = outfile
			ll = ll.Str("path", CLI.Generate.Path)
		}
		if w == nil {
			w = os.Stdout
			ll = ll.Bool("stdout", true)
		}

		if l.HasDebug() {
			ll.Msg("Generating file")
		}
		if format == "tKeys" {
			// TODO: make an alias for this format on the server
			// (don't have the time right now)
			format = "typescript"
		}
		err := api.Export(CLI.Project, format, locale, w)
		if err != nil {
			l.Fatal().Err(err).Msg("Failed export")
		}
		l.Debug().Msg("Export completed")
		if CLI.WithPrettier && CLI.Generate.Path != "" {
			out, err := runPrettier(CLI.Generate.Path, nil)
			if err != nil {
				l.Error().Err(err).Str("out", string(out)).Msg("Failed to run prettier on output")
			}
		}
		l.Info().Msg("Successful export")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	s := reflect.TypeOf(CLI.Generate)
	for _, v := range []string{"Format", "Path"} {
		mustSetVar(s, v, generateCmd, "generate.")
	}
}
func touchFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

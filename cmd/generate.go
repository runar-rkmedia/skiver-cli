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
	"bytes"
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
		// var buf io.Writer
		buf := &bytes.Buffer{}
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

		if l.HasDebug() {
			ll.Msg("Generating file")
		}
		if format == "tKeys" {
			// TODO: make an alias for this format on the server
			// (don't have the time right now)
			format = "typescript"
		}
		// fmt.Println("writer", writer)
		err := api.Export(CLI.Project, format, locale, buf)
		if err != nil {
			l.Fatal().Err(err).Msg("Failed export")
		}
		l.Debug().Msg("Export completed")
		if CLI.Generate.Path == "" {
			os.Stdout.Write(buf.Bytes())
			return
		}
		if buf.Len() == 0 {
			l.Fatal().Msg("No output generated")

		}
		err = os.WriteFile(CLI.Generate.Path, buf.Bytes(), 0644)
		if err != nil {
			l.Fatal().Err(err).
				Str("path", CLI.Generate.Path).
				Msg("Failed during write to file")
		}
		l.Info().Msg("Successful export")
		if CLI.WithPrettier && CLI.Generate.Path != "" {
			out, err := runPrettier(CLI.Generate.Path, nil)
			if err != nil {
				l.Error().Err(err).Str("out", string(out)).Msg("Failed to run prettier on output")
			}
		}
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

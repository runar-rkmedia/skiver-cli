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
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thediveo/enumflag"
)

var (
	format formatMode
)

type formatMode enumflag.Flag

const (
	formatToml formatMode = iota
	formatYaml
	formatJson
)

var formatMap = map[formatMode][]string{
	formatYaml: {"yaml"},
	formatToml: {"toml"},
	formatJson: {"json"},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:       "config",
	Short:     "Show information about the current configuration, or create a new one",
	ValidArgs: []string{"show", "active", "raw", "new"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			switch args[0] {
			case "active":
				write(configFile)
			case "raw":
				c := viper.AllSettings()
				marshalout(c, format)
			case "show":
				marshalout(CLI, format)
			case "new":
				fPath := "skiver-cli." + formatMap[format][0]
				_, err := os.Stat(fPath)
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						l.Fatal().Err(err).Str("filepath", fPath).Msg("Failed to stat path")
					}
				} else {
					l.Fatal().Str("filepath", fPath).Msg("File already exists")
				}
				b := marshal(CLI, format)
				err = os.WriteFile(fPath, b, 0677)
				if err != nil {
					l.Fatal().Err(err).Str("filepath", fPath).Msg("File already exists")
				}
			}
		}
	},
	Args: cobra.OnlyValidArgs,
}

func init() {
	rootCmd.AddCommand(configCmd)

	var formats []string
	for _, v := range formatMap {
		formats = append(formats, v[0])
	}
	configCmd.PersistentFlags().VarP(
		enumflag.New(&format, "format", formatMap, enumflag.EnumCaseInsensitive),
		"format", "f",
		"valid formats: "+strings.Join(formats, ","))
	viper.BindPFlag("config.format", configCmd.PersistentFlags().Lookup("format"))
	viper.SetDefault("config.format", "toml")

}

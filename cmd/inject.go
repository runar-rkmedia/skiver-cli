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
	"fmt"
	"reflect"
	"strings"

	"github.com/runar-rkmedia/skiver/utils"
	"github.com/spf13/cobra"
)

// injectCmd represents the inject command
var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject comments into source-code for locale-usage, with rich descriptions",
	Run: func(cmd *cobra.Command, args []string) {
		api := requireApi(false)
		m := BuildTranslationKeyFromApi(*api, l, CLI.Project, CLI.Locale)
		sorted := utils.SortedMapKeys(m)
		regex := buildTranslationKeyRegexFromMap(sorted)
		replacementFunc := func(groups []string) (replacement string, changed bool) {
			if len(groups) < 3 {
				return "", false

			}
			prefix := groups[0]
			// If the line is a comment, we dont care about replacing it
			if strings.HasPrefix(strings.TrimSpace(prefix), "//") {
				return "", false
			}
			key := groups[1]
			suffix := groups[2]
			rest := strings.Join(groups[3:], "")
			skiverComment := "// skiver: "
			var prevSkiverComment string
			if i := strings.Index(rest, skiverComment); i >= 0 {
				prevSkiverComment = rest[i:]
				rest = rest[0:i]
			}

			found, ok := m[key]
			if !ok {
				panic("Not found")
			}
			var ts string
			if len(found) == 0 {
				return "", false
			}
			f := utils.SortedMapKeys(found)
			for _, k := range f {
				if found[k] == "" {
					continue
				}
				ts += fmt.Sprintf("(%s) %s; ", k, found[k])

			}
			if ts == "" {
				return "", false
			}

			ts = skiverComment + ts
			ts = newLineReplacer.Replace(ts)
			ts = strings.TrimSuffix(ts, " ")
			if prevSkiverComment != "" {
				if prevSkiverComment == ts {
					return "", false
				}
			}

			if strings.TrimSpace(rest) == "," {
				return prefix + key + suffix + ", " + ts, true
			}

			return prefix + key + suffix + ts + "\n" + rest, true
		}
		filter := []string{"ts", "tsx"}

		in := NewInjector(l, CLI.Inject.Dir, CLI.Inject.DryRun, CLI.Inject.OnReplace, CLI.IgnoreFilter, filter, regex, replacementFunc)
		err := in.Inject()
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to inject")
		}
		l.Info().Msg("Done")
	},
}

func init() {
	rootCmd.AddCommand(injectCmd)
	s := reflect.TypeOf(CLI.Inject)
	for _, v := range []string{"DryRun", "Dir", "OnReplace"} {
		mustSetVar(s, v, injectCmd, "inject.")
	}
}

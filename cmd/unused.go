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
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// unusedCmd represents the unused command
var unusedCmd = &cobra.Command{
	Use:   "unused",
	Short: "Find unused translations",
	Run: func(cmd *cobra.Command, args []string) {
		if CLI.Unused.Dir == "" {
			l.Fatal().Msg("Dir is required")
		}
		api := requireApi(true)
		// TODO: also check if translations are used within other translations.
		// For instance, a translation may only be used by refereance.
		source, _ := getFile(CLI.Unused.Source)
		translationKeys, regex := buildTranslationMapWithRegex(l, source, *api, CLI.Project, CLI.Locale)
		found := map[string]bool{}
		foundCh := make(chan string)
		quitCh := make(chan struct{})
		replacementFunc := func(groups []string) (replacement string, changed bool) {
			foundCh <- groups[1]
			return "", false
		}

		go func() {
			for {
				select {
				case <-quitCh:
					return
				case f := <-foundCh:
					found[f] = true
				}
			}
		}()
		filter := []string{"ts", "tsx"}

		in := NewInjector(l, CLI.Unused.Dir, true, "", CLI.IgnoreFilter, filter, regex, replacementFunc, nil)
		err := in.Inject()
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to inject")
		}
		quitCh <- struct{}{}
		var unused []string
		for k := range translationKeys {
			if found[k] {
				continue
			}
			unused = append(unused, k)
		}
		sort.Strings(unused)

		count := len(unused)
		fmt.Println(strings.Join(unused, "\n"))
		if count > 0 {
			l.Info().Int("count-unused", len(unused)).Msg("Found some possibly unused translation-keys")
		} else {
			l.Info().Msg("Found no unused translation-keys")
		}
	},
}

func init() {
	rootCmd.AddCommand(unusedCmd)
	s := reflect.TypeOf(CLI.Unused)
	// viper has some trouble with nested keys it seems
	// https://github.com/spf13/viper/issues/368
	// In my case, registering primitives work, but not complex types, like []string
	for _, v := range []string{"Source", "Dir"} {
		mustSetVar(s, v, unusedCmd, "unused.")
	}
}

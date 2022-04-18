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
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/runar-rkmedia/go-common/logger"
	"github.com/runar-rkmedia/skiver/utils"
	"github.com/spf13/cobra"
)

// injectCmd represents the inject command
var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject comments into source-code for locale-usage, with rich descriptions",
	Run: func(cmd *cobra.Command, args []string) {
		if CLI.Inject.Dir == "" {
			l.Fatal().Msg("Inject.Dir is required")
		}
		switch CLI.Inject.Type {
		case "comment", "tKeys":
		case "":
			l.Fatal().Msg("Inject.Type is required")
		default:
			l.Fatal().Msg("Inject.Type must be one of 'comment', 'tKeys'")
		}

		if _, err := os.Stat(CLI.Inject.Dir); err != nil {
			l.Fatal().Err(err).Msg("Error locating Inject.Dir")
		}
		api := requireApi(false)
		m := BuildTranslationKeyFromApi(*api, l, CLI.Project, CLI.Locale)
		sorted := utils.SortedMapKeys(m)
		filter := []string{"ts", "tsx"}
		regex := buildTranslationKeyRegexFromMap(sorted)

		importPath := findFile(CLI.Inject.Dir, "tKeys.ts")
		if importPath == "" {
			l.Fatal().Msg("Failed to find the tKeys.ts-file. You can generate it with 'skiver generate --format typescript --path src/tKeys.ts'")
		}

		// importPath := path.Join(CLI.Inject.Dir, "tKeys.ts")

		var replacementFunc ReplacementFunc
		var traverserFunc TraverserFunc
		switch CLI.Inject.Type {
		case "comment":
			replacementFunc = commentReplacementFunc(m)
		case "tKeys":
			traverserFunc = tKeysTraverserFunc(l, m, importPath)
		default:
			l.Fatal().Msg("Inject.Type must be one of 'comment', 'tKeys'")
		}

		CLI.IgnoreFilter = append(CLI.IgnoreFilter, importPath)
		in := NewInjector(l, CLI.Inject.Dir, CLI.Inject.DryRun, CLI.Inject.OnReplace, CLI.IgnoreFilter, filter, regex, replacementFunc, traverserFunc)
		err := in.Inject()
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to inject")
		}
		l.Info().
			Str("dir", CLI.Inject.Dir).
			Str("on-replace", CLI.Inject.OnReplace).
			Bool("dry-run", CLI.Inject.DryRun).
			Msg("Done")
	},
}

func findFile(dir string, name string) string {
	var errFound = errors.New("Found")
	var match string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == name {
			match = path

			return errFound
		}
		return nil
	})
	if err != nil && err != errFound {
		l.Fatal().Err(err).Msg("Failed in filepath.WalkDir")
	}
	return match
}

func isString(t chroma.TokenType) bool {
	return t >= 3100 && t < 3200
}
func trimStringToken(t chroma.Token) string {
	switch t.Type {
	case chroma.LiteralString:
	case chroma.LiteralStringAffix:
	case chroma.LiteralStringAtom:
	case chroma.LiteralStringBacktick:
	case chroma.LiteralStringBoolean:
	case chroma.LiteralStringChar:
	case chroma.LiteralStringDelimiter:
	case chroma.LiteralStringDoc:
	case chroma.LiteralStringDouble:
		return strings.Trim(t.Value, `"`)
	case chroma.LiteralStringEscape:
	case chroma.LiteralStringHeredoc:
	case chroma.LiteralStringInterpol:
	case chroma.LiteralStringName:
	case chroma.LiteralStringOther:
	case chroma.LiteralStringRegex:
	case chroma.LiteralStringSingle:
		return strings.Trim(t.Value, `'`)
	case chroma.LiteralStringSymbol:
	}
	return strings.Trim(t.Value, "'\"`")
}

func tKeysTraverserFunc(l logger.AppLogger, m map[string]map[string]string, importPath string) TraverserFunc {
	debug := l.HasDebug() || isDev
	restrictions := [][]*TokenRestriction{
		{
			// Typescript, matches code like:
			//   t("foo.bar")
			//   t("foo.bar",
			//   t("foo.bar" as
			NewTokenRestriction(-2).AddType(chroma.NameOther).AddValue("t", "tt"),
			NewTokenRestriction(-1).AddType(chroma.Punctuation).AddValue("("),
			NewTokenRestriction(1).AddType(chroma.Punctuation).AddValue(")", ",").Or(
				// in typescript, sometimes keys will be added with `as any` as suffix to fit the typings.
				NewTokenRestriction(1).AddType(chroma.Text), //typically whitespace
				NewTokenRestriction(2).AddType(chroma.KeywordReserved).AddValue("as"),
			),
		},
		{
			// Typescript, matches code like:
			//   tKey: "foo.bar"
			NewTokenRestriction(-3).AddType(chroma.NameOther).AddValue("tKey"),
			NewTokenRestriction(-2).AddType(chroma.Operator).AddValue(":"),
			NewTokenRestriction(-1).AddType(chroma.Text).AddValue(" "),
		},

		// TODO: add matcher for
		//   t(IsBar ? "foo.bar" : "foo.baz")
		//   t(
		//     isBar
		//     ? "foo.bar"
		//     ? "foo.baz"
	}
	return func(tokenizer *Tokenizer) {
		tokens := tokenizer.Tokens()
		length := len(tokens)
		at := func(index int) int {
			if index < 0 {
				return 0
			}
			if index >= length {
				return length - 1
			}
			return index
		}
		shouldReplace := false
		for i, t := range tokens {
			if !isString(t.Type) {
				continue
			}
			value := trimStringToken(t)
			_, ok := m[value]
			if ok {
				// fmt.Println("found IT", t.Type, value)
				ll := l.Logger
				slice := tokens[at(i-8):at(i+3)]
				ll = l.With().
					Str("filePath", tokenizer.FilePath).
					Str("lexer", tokenizer.Lexer.Name).
					Interface("slice", slice).
					Interface("matched-token", t).
					Logger()
				match := false
				var mismatches []TokenRestriction
				for _, rSet := range restrictions {
					// All within a set must match

					setMatch := true
					for _, res := range rSet {
						if !res.Matches(i, tokens) {
							setMatch = false
							mismatches = append(mismatches, *res)
							if debug {
								ll.Debug().
									Interface("non-matched-restriction", res).
									Msg("Restriction-check failed")

							}
							break
						}
					}
					if !setMatch {
						continue
					}
					match = true
					if debug {
						ll.Debug().
							Interface("matched-set", rSet).
							Msg("Matched set")
						break
					}
				}
				if !match {
					ll.Warn().
						Interface("mismatches", mismatches).
						Msg("Found key, but restrictions-check did not match.")
					if debug {
						if isDev {
							logger.Debug("Found key, but restriction-check did not match", map[string]interface{}{
								"slice":         slice,
								"value":         value,
								"matched-token": t,
								"fileName":      tokenizer.FilePath,
								"mismatches":    mismatches,
							})
						}
					}
					continue
				}

				// We have a match!
				// We should do replacement / here
				// I dont think it matters much if if add "invalid" token-values here, as the tokens are only
				// concatinated afterwards.
				t.Value = fmt.Sprintf("tKeys.%s", value)
				tokens[i] = t
				shouldReplace = true
			}
		}
		if !shouldReplace {
			return
		}
		// add import-statement
		relativePath, err := filepath.Rel(filepath.Dir(tokenizer.FilePath), importPath)
		if err != nil {
			l.Fatal().Err(err).Msg("Failed to calculate relative importPath")
		}
		relativePath = stripExtension(relativePath)
		relativePath = filepath.ToSlash(relativePath)
		importToken := chroma.Token{Value: fmt.Sprintf(`import tKeys from "%s"`, relativePath) + "\n"}
		tokens = append([]chroma.Token{importToken}, tokens...)
		tokenizer.SetTokens(tokens)
	}
}

func stripExtension(filePath string) string {
	return strings.TrimSuffix(filePath, path.Ext(filePath))
}

type TokenRestriction struct {
	Offset        int
	AllowedTypes  []chroma.TokenType
	AllowedValues []string
	OrSet         []*TokenRestriction
}

func (t *TokenRestriction) AddType(type_ ...chroma.TokenType) *TokenRestriction {
	t.AllowedTypes = append(t.AllowedTypes, type_...)
	return t
}
func (t *TokenRestriction) AddValue(v ...string) *TokenRestriction {
	t.AllowedValues = append(t.AllowedValues, v...)
	return t
}
func (t *TokenRestriction) Or(v ...*TokenRestriction) *TokenRestriction {
	t.OrSet = append(t.OrSet, v...)
	return t
}
func (t TokenRestriction) Matches(i int, tokens []chroma.Token) bool {
	index := i + t.Offset
	if index >= 0 && index < len(tokens) {
		tok := tokens[index]

		matchesType := false
		if len(t.AllowedTypes) > 0 {
			for _, ty := range t.AllowedTypes {
				if ty == tok.Type {
					matchesType = true
					break
				}
			}
		}
		if !matchesType && len(t.OrSet) == 0 {
			return false
		}
		matchesValue := false
		if len(t.AllowedValues) > 0 {
			for _, ty := range t.AllowedValues {
				if ty == tok.Value {
					matchesValue = true
					break
				}
			}
		}
		if matchesValue {
			return true
		}
	}

	for _, or := range t.OrSet {
		if or.Matches(i, tokens) {
			return true
		}
	}
	return false
}
func NewTokenRestriction(offset int) *TokenRestriction {
	return &TokenRestriction{
		Offset:        offset,
		AllowedTypes:  []chroma.TokenType{},
		AllowedValues: []string{},
		OrSet:         []*TokenRestriction{},
	}

}

func isToken(t chroma.Token, type_ chroma.TokenType, values ...string) bool {
	if t.Type != type_ {
		return false
	}
	if len(values) == 0 {
		return true
	}
	for _, v := range values {
		if v == t.Value {
			return true
		}
	}
	return false
}

// Injects comments after the translation-key
func commentReplacementFunc(m map[string]map[string]string) ReplacementFunc {
	return func(groups []string) (replacement string, changed bool) {
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
}

func init() {
	rootCmd.AddCommand(injectCmd)
	s := reflect.TypeOf(CLI.Inject)
	for _, v := range []string{"DryRun", "Dir", "OnReplace", "Type"} {
		mustSetVar(s, v, injectCmd, "inject.")
	}
}

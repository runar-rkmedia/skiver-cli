//go:build !nopretty

package cmd

/*

Even though this looks nice, its add about 4MB to the binary-size with stripping.
Currently, I dont think that the feature is that important.

Also, there would need to be a smarter selection of styles, as many of the styles will
not fit with the users terminal colors.

For replacement, we would probably also want to hightlight the replacements.

However, the lexers are very interesting for various uses, and could probaly improve the replacer a lot,
considering that running regex-replacements over source-code is both brittle and prone to error.

The lexer.Tokenize returns tokens which looks very promising, at least for typescript.
there is also a Contcatonator available. Using this instead of regexes looks awesome.

*/

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

func TokenizeSourceFileContent(filepath string, content string) (chroma.Iterator, error) {
	lexer := lexers.Match(filepath)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		return nil, fmt.Errorf("failed to identify content-type for file '%s'", filepath)
	}
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize content: %w", err)
	}
	return iterator, nil
}

func ConcatTokens(iterator chroma.Iterator) string {
	return chroma.Stringify(iterator.Tokens()...)
}

func PrettyPrinttFile(filepath string, content string) (string, error) {
	if CLI.NoColor || !isInteractive {
		return content, nil
	}
	lexer := lexers.Match(filepath)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		return content, fmt.Errorf("failed to identify content-type for file '%s'", filepath)
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content, fmt.Errorf("failed to tokenize content: %w", err)
	}
	style := styles.Fallback
	// TODO: we could use https://github.com/muesli/termenv to help pick a nice theme automatically
	if CLI.HighlightStyle != "" {
		style = styles.Get(CLI.HighlightStyle)
		if style == nil {
			style = styles.Fallback
		}

	}
	formatter := formatters.Get("terminal")
	if formatter == nil {
		formatter = formatters.Fallback
	}
	w := new(strings.Builder)
	err = formatter.Format(w, style, iterator)
	if err != nil {
		return content, fmt.Errorf("failed to format content: %w", err)
	}
	return w.String(), nil

}

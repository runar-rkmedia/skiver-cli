package cmd

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

type Tokenizer struct {
	iterator chroma.Iterator
	tokens   *[]chroma.Token
	index    int
	FilePath string
	Lexer    chroma.Config
}

func TokenizeSourceFileContent(filepath string, content string) (Tokenizer, error) {
	lexer := lexers.Match(filepath)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		return Tokenizer{}, fmt.Errorf("failed to identify content-type for file '%s'", filepath)
	}
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return Tokenizer{}, fmt.Errorf("failed to tokenize content: %w", err)
	}
	t := Tokenizer{
		iterator: iterator,
		index:    0,
		FilePath: filepath,
		Lexer:    *lexer.Config(),
	}
	return t, nil
}

func (t Tokenizer) Tokens() []chroma.Token {
	if t.tokens != nil && len(*t.tokens) != 0 {
		return *t.tokens
	}
	return t.iterator.Tokens()
}
func (t Tokenizer) IsChanged() bool {
	return t.tokens != nil
}
func (t *Tokenizer) SetTokens(tokens []chroma.Token) {
	t.tokens = &tokens
}
func (t Tokenizer) Concat() string {
	return chroma.Stringify(t.Tokens()...)
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

package syntax

import (
	"io"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func WithStyle(style string) Option {
	return func(opt *options) {
		opt.style = &style
	}
}

func WithFallbackLanguage(lang string) Option {
	return func(opt *options) {
		opt.fallback = &lang
	}
}

func WithLinkPrefix(prefix string) Option {
	return func(opt *options) {
		opt.prefix = &prefix
	}
}

var (
	defaultStyle      = "github-dark"
	defaultLinkPrefix = "script"
)

func FormatHTML(w io.Writer, script string, opts ...Option) error {
	opt := &options{
		style:  &defaultStyle,
		prefix: &defaultLinkPrefix,
	}
	for _, o := range opts {
		o(opt)
	}

	lexer := lexers.Analyse(script)
	if lexer == nil {
		if opt.fallback != nil {
			lexer = lexers.Get(*opt.fallback)
		}
		if lexer == nil {
			lexer = lexers.Fallback
		}
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get(*opt.style)
	formatter := html.New(
		html.BaseLineNumber(0),
		html.WithLineNumbers(true),
		html.WithClasses(false),
		html.LineNumbersInTable(true),
		html.WithLinkableLineNumbers(true, *opt.prefix),
		html.TabWidth(4),
		html.WithAllClasses(true),
		html.WrapLongLines(true),
	)

	iterator, err := lexer.Tokenise(nil, script)
	if err != nil {
		return err
	}

	return formatter.Format(w, style, iterator)
}

type options struct {
	fallback *string
	style    *string
	prefix   *string
}

type Option func(*options)

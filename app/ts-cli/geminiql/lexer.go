/*
Copyright 2022 Huawei Cloud Computing Technologies Co., Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package geminiql

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	EOF rune = 0
)

const (
	EOF_TOKEN int = iota
	WS_TOKEN
	BAD_STRING
	BAD_ESCAPE
	BAD_RAW
	ILLEGAL_TOKEN
)

func isChar(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentChar(ch rune) bool {
	return isChar(ch) || unicode.IsDigit(ch)
}

type Tokenizer struct {
	r        io.RuneScanner
	keywords map[string]int
	tokens   []int
}

func NewTokenizer(rd io.Reader) *Tokenizer {
	t := &Tokenizer{
		r:        bufio.NewReader(rd),
		keywords: make(map[string]int),
		tokens:   nil,
	}

	for i, name := range QLToknames {
		if i-3 < 0 {
			continue
		}
		t.keywords[strings.ToUpper(name)] = INSERT + i - 3
	}

	return t
}

func (t *Tokenizer) updateTokens(tok *int) {
	t.tokens = append(t.tokens, *tok)
}

func (t *Tokenizer) firstToken() int {
	if len(t.tokens) == 0 {
		return EOF_TOKEN
	}
	return t.tokens[0]
}

func (t *Tokenizer) lastToken() int {
	if len(t.tokens) == 0 {
		return EOF_TOKEN
	}
	return t.tokens[len(t.tokens)-1]
}

func (t *Tokenizer) Scan() (tok int, val string) {
	defer t.updateTokens(&tok)

	if t.firstToken() == INSERT && t.lastToken() == EQ {
		return t.scanRaw()
	}

	ch := t.Lookahead()

	if unicode.IsSpace(ch) {
		return t.scanWhiteSpace()
	} else if isChar(ch) {
		return t.scanIdentifier()
	} else if unicode.IsDigit(ch) {
		return t.scanDigit()
	}

	switch ch {
	case EOF:
		return EOF_TOKEN, ""
	case '\'':
		return t.scanString()
	case '"':
		return t.scanString()
	case '.':
		ch = t.read()
		return DOT, string(ch)
	case '=':
		ch = t.read()
		return EQ, string(ch)
	case ',':
		ch = t.read()
		return COMMA, string(ch)
	}

	return ILLEGAL_TOKEN, ""
}

func (t *Tokenizer) scanIdentifier() (int, string) {
	var buf bytes.Buffer

	for {
		ch := t.read()
		if ch == EOF {
			break
		} else if !isIdentChar(ch) {
			t.unRead()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	tok := t.scanKeywords(buf.String())
	return tok, buf.String()
}

func (t *Tokenizer) scanKeywords(s string) int {
	if tok, ok := t.keywords[strings.ToUpper(s)]; ok {
		return tok
	}
	return IDENT
}

func (t *Tokenizer) scanRaw() (int, string) {
	var buf bytes.Buffer
	for {
		ch := t.read()
		if ch == ',' || ch == ' ' {
			t.unRead()
			return RAW, buf.String()
		} else if ch == EOF {
			return RAW, buf.String()
		} else {
			buf.WriteRune(ch)
		}
	}
}

func (t *Tokenizer) scanString() (int, string) {
	end := t.read()

	var buf bytes.Buffer
	for {
		ch := t.read()
		if ch == end {
			return STRING, buf.String()
		} else if ch == EOF || ch == '\n' {
			return BAD_STRING, buf.String()
		} else if ch == '\\' {
			nch := t.read()
			if nch == 'n' {
				buf.WriteRune('\n')
			} else if nch == '\\' {
				buf.WriteRune('\\')
			} else if nch == '"' {
				buf.WriteRune('"')
			} else if nch == '\'' {
				buf.WriteRune('\'')
			} else {
				return BAD_ESCAPE, string(ch) + string(nch)
			}
		} else {
			buf.WriteRune(ch)
		}
	}
}

func (t *Tokenizer) scanDigit() (int, string) {
	var buf bytes.Buffer

	typ := INTEGER
	for {
		ch := t.read()

		if ch == EOF {
			break
		} else if ch == '.' {
			nch := t.Lookahead()
			if !unicode.IsDigit(nch) {
				t.unRead()
				break
			} else {
				buf.WriteRune(ch)
				typ = DECIMAL
			}
		} else if !unicode.IsDigit(ch) {
			t.unRead()
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	return typ, buf.String()
}

func (t *Tokenizer) scanWhiteSpace() (int, string) {
	var buf bytes.Buffer

	for {
		ch := t.read()
		if ch == EOF {
			break
		} else if !unicode.IsSpace(ch) {
			t.unRead()
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	return WS_TOKEN, buf.String()
}

func (t *Tokenizer) Lookahead() rune {
	ch, _, err := t.r.ReadRune()
	defer t.unRead()

	if err != nil {
		ch = EOF
	}
	return ch
}

func (t *Tokenizer) read() rune {
	ch, _, err := t.r.ReadRune()

	if err != nil {
		ch = EOF
	}
	return ch
}

func (t *Tokenizer) unRead() error {
	return t.r.UnreadRune()
}

type QLLexerImpl struct {
	tokenizer *Tokenizer
	ast       *QLAst
}

func QLNewLexer(tokenizer *Tokenizer, ast *QLAst) QLLexer {
	l := &QLLexerImpl{
		tokenizer: tokenizer,
		ast:       ast,
	}

	return l
}

func (l *QLLexerImpl) Lex(lval *QLSymType) int {
	var tok int
	var val string

	for {
		tok, val = l.tokenizer.Scan()
		switch tok {
		case ILLEGAL_TOKEN:
			l.Error(fmt.Sprintf("illegal token %s", val))
		case EOF_TOKEN:
			return 0
		case BAD_STRING:
			l.Error(fmt.Sprintf("bad string %s", val))
		case BAD_ESCAPE:
			l.Error(fmt.Sprintf("bad escape %s", val))
		case INTEGER:
			var err error
			lval.integer, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				l.Error(err.Error())
			}
		case DECIMAL:
			var err error
			lval.decimal, err = strconv.ParseFloat(val, 64)
			if err != nil {
				l.Error(err.Error())
			}
		default:
			lval.str = val
		}

		if tok != WS_TOKEN {
			break
		}
	}

	return tok
}

func (l *QLLexerImpl) Error(s string) {
	l.ast.Error = fmt.Errorf("ERR: %s", s)
}

func (l *QLLexerImpl) UpdateStmt(stmt Statement) {
	l.ast.Stmt = stmt
}

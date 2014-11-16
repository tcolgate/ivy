// Copyright 2014 Rob Pike. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse // import "robpike.io/ivy/parse"

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"robpike.io/ivy/scan"
	"robpike.io/ivy/value"
)

var debugFlags = []string{
	"panic",
	"parse",
	"tokens",
	"types",
}

func (p *Parser) need(want ...scan.Type) scan.Token {
	tok := p.Next()
	for _, w := range want {
		if tok.Type == w {
			return tok
		}
	}
	p.errorf("expected %s, got %s", want, tok)
	panic("not reached")
}

func (p *Parser) special() {
Switch:
	switch text := p.need(scan.Identifier).Text; text {
	case "debug":
		if p.Peek().Type == scan.Newline {
			for _, f := range debugFlags {
				fmt.Println(f)
			}
			break Switch
		}
		name := p.need(scan.Identifier).Text
		found := false
		for _, f := range debugFlags {
			if f == name {
				found = true
				break
			}
		}
		if !found {
			fmt.Println("no such debug flag:", name)
			break Switch
		}
		if p.Peek().Type != scan.Number {
			// Toggle the value
			p.config.SetDebug(name, !p.config.Debug(name))
			if p.config.Debug(name) {
				fmt.Println("1")
			} else {
				fmt.Println("0")
			}
			break
		}
		number, err := value.ValueString(p.need(scan.Number).Text)
		if err != nil {
			p.errorf("%s", err)
		}
		v, ok := number.(value.Int)
		p.config.SetDebug(name, ok && v.ToBool())
	case "format":
		if p.Peek().Type == scan.Newline {
			fmt.Printf("%q\n", p.config.Format())
			break
		}
		p.config.SetFormat(p.getString())
	case "get":
		p.runFromFile(p.getString())
	case "origin":
		if p.Peek().Type != scan.Number {
			fmt.Println(p.config.Origin())
			break
		}
		origin, err := strconv.Atoi(p.Next().Text)
		if err != nil {
			p.errorf("%s", err)
		}
		if origin != 0 && origin != 1 {
			p.errorf("illegal origin", err)
		}
		p.config.SetOrigin(origin)
	case "prompt":
		if p.Peek().Type == scan.Newline {
			fmt.Printf("%q\n", p.config.Format())
			break
		}
		p.config.SetPrompt(p.getString())
	case "seed":
		if p.Peek().Type != scan.Number {
			fmt.Println(p.config.Origin())
			break
		}
		seed, err := strconv.Atoi(p.Next().Text)
		if err != nil {
			p.errorf("%s", err)
		}
		p.config.RandomSeed(int64(seed))
	default:
		p.errorf(")%s: not recognized", text)
	}
	p.need(scan.Newline)
}

// getString returns the value of the string or raw string
// that must be next in the input.
func (p *Parser) getString() string {
	str := p.need(scan.String, scan.RawString).Text
	str, err := strconv.Unquote(str)
	if err != nil {
		p.errorf("%s", err)
	}
	return str
}

var runDepth = 0

// runFromFile executes the contents of the named file.
func (p *Parser) runFromFile(name string) {
	runDepth++
	if runDepth > 10 {
		p.errorf("get %q nested too deep", name)
	}
	defer func() {
		runDepth--
		err := recover()
		if err == nil {
			return
		}
		if err, ok := err.(value.Error); ok {
			fmt.Fprintf(os.Stderr, "%s: %s\n", p.Loc(), err)
			return
		}
		panic(err)
	}()
	fd, err := os.Open(name)
	if err != nil {
		p.errorf("%s", err)
	}
	scanner := scan.New(p.config, name, bufio.NewReader(fd))
	parser := NewParser(p.config, name, scanner)
	for {
		value, ok := parser.Line()
		if value != nil {
			fmt.Fprintln(os.Stdout, value)
		}
		if !ok {
			return
		}
	}
}

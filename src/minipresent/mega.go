// Copyright (2014) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	log "minilog"
	"path/filepath"
	"strings"
)

type Mega struct {
	Text     template.HTML
	Filename string
}

func init() {
	Register("mega", parseMega)
}

func (c Mega) TemplateName() string { return "mega" }

func executable(m Mega) bool {
	return *f_exec
}

func parseMega(ctx *Context, sourceFile string, sourceLine int, cmd string) (Elem, error) {
	cmd = strings.TrimSpace(cmd)
	log.Debug("parseMega cmd: %v", cmd)

	f := strings.Fields(cmd)
	if len(f) != 2 {
		return nil, fmt.Errorf("invalid .mega directive: %v", cmd)
	}

	filename := filepath.Join(filepath.Dir(sourceFile), f[1])
	log.Debug("filename: %v", filename)

	text, err := ctx.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s:%d: %v", sourceFile, sourceLine, err)
	}

	data := &megaTemplateData{
		Body: string(text),
	}

	var buf bytes.Buffer
	if err := megaTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}

	return Mega{
		Text:     template.HTML(buf.String()),
		Filename: filepath.Base(filename),
	}, nil
}

type megaTemplateData struct {
	Body string
}

var megaTemplate = template.Must(template.New("code").Parse(megaTemplateHTML))

const megaTemplateHTML = `
<pre contenteditable="true" spellcheck="false">
{{.Body}}
</pre>
`
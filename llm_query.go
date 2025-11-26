package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

var (
	//go:embed prompt.template
	PromptTmplText string
	PromptTmpl = template.Must(template.New("prompt").Parse(PromptTmplText))
)

type PromptTmplOptions struct {
	Context string
	UserQuery string
}

func GenerateLLMPrompt(userQuery string) (string, error) {
	vector, err := EmbedQuery(userQuery)
	if err != nil {
		return "", err
	}

	rows, err := SearchWithQueryVector(vector)
	if err != nil {
		return "", err
	}

	var context strings.Builder
	for _, row := range rows {
		context.WriteString(row.Text)
	}

	var output bytes.Buffer
	err = PromptTmpl.Execute(&output, PromptTmplOptions{
		Context: context.String(),
		UserQuery: userQuery,
	})

	return output.String(), err
}

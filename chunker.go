package main

import (
	"strings"

	"github.com/neurosnap/sentences/english"
)

func ChunkTextFile(contents string) ([]string, error) {
	tokenizer, err := english.NewSentenceTokenizer(nil)
    if err != nil {
		return []string{}, err
    }

    sentences := tokenizer.Tokenize(contents)
	var chunks []string
	for i, sentence := range sentences {
		var context strings.Builder
		// Let's include the previous and next sentence for some overlap
		if i - 1 >= 0 {
			context.WriteString(sentences[i - 1].Text)
		}
		context.WriteString(sentence.Text)
		if i + 1 < len(sentences) {
			context.WriteString(sentences[i + 1].Text)
		}

		chunks = append(chunks, context.String())
	}

	return chunks, nil
}

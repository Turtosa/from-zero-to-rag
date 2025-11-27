---
title: "From Zero to RAG"
author: "Kevin Roleke"
publishedAt: "2025-11-27"
excerpt: "Roll your own RAG! Learn about chunking, embedding, vector search, and more."
tags: ["rag", "nlp", "tutorial"]
---

In 2025, it seems nearly tech company is introducing RAG as part of their new AI-enhanced product lineup. Technobabble zingers like "hybrid semantic-lexical retrieval" or "state-of-the-art multi-modal knowledge ingestion" often plague marketing material, leading the uninitiated to assume building RAG pipelines is a difficult task best left to the wizards.

This is the first part of a series on building a RAG system from scratch (save for the embedding model itself). Part one will cover the following:
- Basic overview of RAG
- Setting up a vector database and embedding model
- Basic chunking of text files
- Embedding and inserting chunks into our database
- Vector query searching
- Prompt generation

Full code is available in the [From Zero to RAG repo](https://github.com/Turtosa/from-zero-to-rag)
## Overview
RAG can be thought of as two systems, parsing and retrieval, each containing several steps depending on how complex your project is.

Parsing takes your soup of documents and converts it into neat vectors:
1. \*Normalization: conversion of various documents into simple text (e.g. OCR, parsing of DOCX, PDF, PPT, MD, HTML, etc)
2. Chunking: splitting the text into small sections such as sentences or paragraphs
3. Embedding: generate high-dimension vectors from text chunks
4. Insertion: placing these vectors in a database for future querying

Retrieval takes a user query and turns it into actionable context for the LLM:
1. \*Pre-processing of user query
	1. \*Decide whether or not to bypass the RAG for this request (e.g. if user asks what time of day it is, there's no point in searching the knowledge base)
	2. \*Break up the request into sub-queries (e.g. if user query is complex and contains many possible queries, we break it into logical parts and perform search for each of these).
	3. \*Generate metadata filters for the results (e.g. if user query is looking for data from 2023, we will only search files created in 2023).
2. Query embedding: take the plain text queries and convert to vectors.
3. Search: find the N most similar vectors to the user query
4. \*Post-processing of results
	1. \*Re rank results if multiple/alternative search algorithms were used (e.g. BM25, full text).
	2. \*Rearrange results: LLMs tend to prefer data at the beginning and end of the context[<sup>[1]</sup>](https://arxiv.org/abs/2307.03172), so place low confidence chunks in the middle, flanked on either end by the high confidence chunks.
	3. \*Drop anything that doesn't match our metadata filters.
5. Prompt: Generate a new prompt combining the user's query with the context from our search.

In this first part, we will create a very simple setup, and skip the steps marked with an asterisk above. Let's dive into it.
## Prerequisites
As previously mentioned, creating an embedding model is way out-of-scope for this series: we will rely on external open source tooling for this aspect. Similarly, we will be using an existing vector database, although it is not necessary to use one at all for this simple example.

After briefly comparing the most popular vector databases, I settled on [Infinity](https://github.com/infiniflow/infinity/) due to its performance and relative simplicity.  For the embedding model, we will use [bge-small-en-v1.5](https://huggingface.co/BAAI/bge-small-en-v1.5) (due to its performance on my lowly RTX 4070 mobile GPU); but you are of course free to switch this up.

For future compatibility's sake, we're going to wrap this model in a server that matches the OpenAI API spec. There are several options we can pick from, but my favorite, unfortunately for you, is also named [Infinity](https://github.com/michaelfeil/infinity); so from now on, I will refer to the vector database as "Infinity (DB)" and the embedding model server as "Infinity (Embed)".
### Installation
Getting an instance of Infinity (DB) running on your local machine is as easy as pulling the docker container.
```bash
sudo mkdir -p /var/infinity && sudo chown -R $USER /var/infinity
docker pull infiniflow/infinity:nightly
docker run -d --name infinity -v /var/infinity/:/var/infinity --ulimit nofile=500000:500000 --network=host infiniflow/infinity:nightly
```

The bge-small-en-v1.5 model and therefor our Infinity (Embed) requires GPU acceleration, so we will need to setup nvidia-docker (and your card's NVIDIA drivers).
```bash
sudo pacman -S nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker
```

Finally, let's setup Infinity (Embed) with docker.
```bash
port=7997
model1=michaelfeil/bge-small-en-v1.5 # fork
volume=$PWD/data

docker run -it --gpus all \
 -v $volume:/app/.cache \
 -p $port:$port \
 michaelf34/infinity:latest \
 v2 \
 --model-id $model1 \
 --port $port
```

This will pull the specified model from [HuggingFace](https://huggingface.co/) and start the API server on port 7997.
### Configuration
Start by creating a new database in Infinity (DB). I called it "rfs" here:
```bash
curl --request POST \
     --url http://localhost:23820/databases/rfs \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{"create_option": "ignore_if_exists"}'
```

Now, let's create a table under that database. We will want columns for vectors and text, as well as for filename and chunk index (these latter two will be useful in future parts).
```bash
curl --request POST \
     --url http://localhost:23820/databases/rfs/tables/data \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '{
         "create_option": "ignore_if_exists",
         "fields": [
             {
                  "name": "name",
                  "type": "varchar"
             },
             {
                  "name": "index",
                  "type": "int",
                  "default": 0
             },
             {
                  "name": "dense_column",
                  "type": "vector,384,float"
             },
             {
                  "name": "fulltext_column",
                  "type": "varchar",
                  "default": ""
             }
        ]
    }'
```

Finally, we add an index for full text search:
```bash
curl --request POST \
     --url http://localhost:23820/databases/rfs/tables/data/indexes/fts \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '
     {
          "fields":
          [
              "fulltext_column"
          ],
          "index":
          {
              "type": "fulltext",
              "analyzer": "standard"
          },
          "create_option": "ignore_if_exists"
     } '

```

Now that setup is done, we're ready to write some code!
## Chunking
There are several ways we can opt to split our data into chunks. A naive approach would be to use a fixed string length (e.g. turn a 1000 character text file into 100 sequential chunks of 10 characters); the problem here is that we're dealing with *natural language*: we'd be splitting up a paragraph into nonsensical sentence fragments.

A slightly better approach would be to split according to paragraphs or sections (like starting a new chunk every time we run into a markdown heading or a double newline). However, this can cause issues when paragraphs are not used, or are unable to be identified in some documents (some chunks will be absolutely massive while others are paragraph sized).

For this project, I've opted to use sentence-based chunking. For determining sentence boundaries, the first thought that comes to mind is just to split on common delimiters like period, semi-colon, colon, and em-dash. Unfortunately though, there are other use-cases for these characters in the English language (e.g. "e.g."); "e" and "g" are not sentences, so we need a way to filter these out. Thankfully there exists an abundance of NLP tools to help us with sentence tokenization.

Here's the first iteration of our chunking function, using neurosnap's [Sentences](https://github.com/neurosnap/sentences) module.
```go
package main

import (
	"github.com/neurosnap/sentences/english"
)

func ChunkText(contents string) ([]string, error) {
	tokenizer, err := english.NewSentenceTokenizer(nil)
    if err != nil {
		return []string{}, err
    }

    sentences := tokenizer.Tokenize(contents)
	var chunks []string
	for _, sentence := range sentences {
		chunks = append(chunks, sentence.Text)
	}

	return chunks, nil
}
```

We will also want to pad the chunk with a couple surrounding sentences, in order to bolster the coherence of context embedded within each chunk.
```go
package main

import (
	"strings"

	"github.com/neurosnap/sentences/english"
)

func ChunkText(contents string) ([]string, error) {
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
```
## Embedding and Inserting
Begin by defining a data structure to represent our database table schema as defined above.
```go
type VectorRow struct {
	Name string `json:"name"`
	Text string `json:"fulltext_column"`
	Index int `json:"index"`
	Vector []float64 `json:"dense_column"`
}
```

Our embedding server is quite simple. It takes in a list of strings (our chunks) and spits out a list of vectors (which are represented with a list of floats).
```bash
$ curl --request POST \
     --url http://localhost:7997/embeddings \
     --header 'accept: application/json' \
     --header 'content-type: application/json' \
     --data '
     {
          "model": "michaelfeil/bge-small-en-v1.5",
          "input": [
              "chunk 1",
              "chunk 2"
          ]
     } ' | jq

{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [
        -0.033600203692913055,
        -0.0808352679014206,
        0.04066111519932747,
        -0.05867861211299896,
        -0.012052246369421482,
 ...
```

We'll define two functions, the first of which simply calls the embeddings endpoint on our model server, and the latter of which inserts the result into our table.
```go
func GetEmbeddings(input []string, fname string) ([]VectorRow, error)
func InsertEmbeddings(input []VectorRow) error
```

Throwing everything together:
```go
func ChunkAndEmbedFile(filename string) error {
	// Just dealing with text files for now. We will add parsing for other document types later.
	fs := strings.Split(filename, ".")
	if fs[len(fs)-1] != "txt" {
		return fmt.Errorf("Unsupported filetype: %s", fs[len(fs)-1])
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	chunks, err := ChunkText(string(contents))
	if err != nil {
		return err
	}

	rows, err := GetEmbeddings(chunks, filename)
	if err != nil {
		return err
	}

	err = InsertEmbeddings(rows)
	return err
}
```
## Retrieval
Our RAG process kicks off with a query from the user. We then need to embed this query into a vector, at which point we can query the database for relative semantic matches in `dense_column`. Vector search is great because it relates the context and meaning of queries, rather than directly looking for keyword matches.

After embedding the user query with the same exact model and endpoint from above, we perform a search query on Infinity (DB). The search endpoint was a bit annoying, with the documentation not quite matching up with reality, and some ungraceful segfaults occurring when I failed to conform to the implied spec.

Following some trial and error, I ended up with the following (which works great and is stable for now):
```go
type InfinityResponse struct {
	ErrorCode int `json:"error_code"`
	ErrorMsg *string `json:"error_msg"`
}

type SearchResponse struct {
	InfinityResponse
	Output [][]map[string]any `json:"output"`
	Rows *[]VectorRow `json:"rows"`
	Total int `json:"total_hits_count"`
}

type MatchRequest struct {
    MatchMethod   string      `json:"match_method"`
    Field        string      `json:"fields"`
    QueryVector   any `json:"query_vector,omitempty"`
    FDE           *FDE        `json:"fde,omitempty"`
    MatchingText  *string     `json:"matching_text,omitempty"`
    ElementType   *string      `json:"element_type,omitempty"`
    MetricType    *string      `json:"metric_type,omitempty"`
    FusionMethod  *string     `json:"fusion_method,omitempty"`
    TopN          int         `json:"topn"`
	Params *map[string]string `json:"params"`
}

type FDE struct {
    QueryTensor     [][]float64 `json:"query_tensor"`
    TargetDimension int         `json:"target_dimension"`
}

type SearchRequest struct {
	Output []string `json:"output"`
	Highlight *[]string `json:"highlight,omitempty"`
	Filter *string `json:"filter,omitempty"`
	Search []MatchRequest `json:"search"`
}

func SearchWithQueryVector(queryVector []float64) ([]VectorRow, error) {
	l2 := "l2" // Ah Golang, why cant you have std::option?
	fl := "float"
	return SearchVectors([]MatchRequest{
		{
			MatchMethod: "dense",
			TopN: 2,
			Field: "dense_column",
			QueryVector: queryVector,
			ElementType: &fl,
			MetricType: &l2,
			Params: &map[string]string{},
		},
	})

}

func SearchVectors(matches []MatchRequest) ([]VectorRow, error) {
    jsonData, err := json.Marshal(SearchRequest{
		Search: matches,
		Output: []string{"name", "index", "fulltext_column"},
	})
    if err != nil {
        return []VectorRow{}, fmt.Errorf("Error marshaling JSON: %v\n", err)
    }

	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:23820/databases/rfs/tables/data/docs", bytes.NewReader(jsonData))
	if err != nil {
		return []VectorRow{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return []VectorRow{}, err
	}
	defer resp.Body.Close()

	var res SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
    if err != nil {
        return []VectorRow{}, fmt.Errorf("Error making request: %v\n", err)
    }
	if res.ErrorCode != 0 {
		return []VectorRow{}, fmt.Errorf("Error searching: %s", *res.ErrorMsg)
	}

	// This is horrible, but Infinity outputs in this format
	// [[{"name":"TODO: filename"},{"index":0},{"fulltext_column":"Saturday"}]]
	// This is also not reflected in their documentation.
	var rows []VectorRow
	for _, result := range res.Output {
		var row VectorRow
		a := map[string]any{}
		for _, column := range result {
			maps.Copy(a, column)
		}
		row.Name = a["name"].(string)
		row.Index = int(a["index"].(float64))
		row.Text = a["fulltext_column"].(string)
		rows = append(rows, row)
	}

	return rows, nil
}
```

## The Prompt
We want to generate a prompt that supplies the LLM with both the retrieved context and the original user query. Let's not over-complicate this just yet:
```Markdown
Use the following context to answer the question below.

# Context
{{.Context}}

# Question
Question: {{.UserQuery}}
```

Simple enough. We'll use Go's builtin template engine and embed the prompt straight in the binary:
```go
var (
	//go:embed prompt.template
	PromptTmplText string
	PromptTmpl = template.Must(template.New("prompt").Parse(PromptTmplText))
)

type PromptTmplOptions struct {
	Context string
	UserQuery string
}
```

Finally we generate the full prompt:
```go
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
```

## Data
We want a simple, but realistic dataset for testing our rudimentary RAG. I opted for Aesop's Fables, the full text of which has been conveniently split into 311 text files on this [repo](https://github.com/barbarabai/aesop).

The last piece of our RAG pie is this function to recursively ingest all files in a directory:
```go
func ProcessDirectory(dirname string) error {
    err := filepath.WalkDir(dirname, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        if strings.HasPrefix(d.Name(), ".") {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        if !d.IsDir() {
			log.Printf("Processing file: %s\n", path)
			err := ChunkAndEmbedFile(path)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Success!")
			}
        }

        return nil
    })
	return err
}
```

Time for a quick test! Embed and insert Aesop's Fables into the database like so:
```go
err := ProcessDirectory("../data/aesop")
if err != nil {
	log.Fatalln(err)
}
```

And for our first query:
```go
prompt, err := GenerateLLMPrompt("What happened to the oak?")
if err != nil {
	log.Fatalln(err)
}
log.Println(prompt)
```

The resultant prompt:
```
2025/11/26 23:16:59 Use the following context to answer the question below.

# Context
 An old oak, lamenting when too late the
destruction of his companions, said to a neighboring cedar, �The first
step has lost us all. If we had not given up the rights of the ash, we
might yet have retained our own privileges and have stood for ages.�The Oak and the Reeds

A VERY LARGE OAK was uprooted by the wind and thrown across a stream. It
fell among some Reeds, which it thus addressed: �I wonder how you, who
are so light and weak, are not entirely crushed by these strong winds.�
 They replied, �You fight and contend with the wind, and consequently you
are destroyed; while we on the contrary bend before the least breath of
air, and therefore remain unbroken, and escape.�

Stoop to conquer.

# Question
Question: What happened to the oak?
```

Pasting that into Opus 4.1, I got this response:
```
Based on the context provided, two different things happened to the oak in these two separate fables:

1. **In "The Oak and the Reeds":** The oak was uprooted by strong winds and thrown across a stream, where it fell among some reeds. This happened because the oak tried to fight and contend with the wind rather than bending with it.
2. **In the first fable:** The old oak survived but was lamenting the destruction of his companion trees. He regretted that they had given up the rights of the ash tree, believing this first concession led to the loss of all their privileges and the destruction of the other trees.

The main oak that met its demise was the one in "The Oak and the Reeds" - it was destroyed by trying to resist the wind instead of yielding to it like the flexible reeds did.
```

Not bad, eh?

Definitely some room for improvement, though. Stay tuned for part 2 of Zero to RAG for a dive into pre-processing techniques.
## References
1. https://arxiv.org/abs/2307.03172
2. https://github.com/Turtosa/from-zero-to-rag
3. https://github.com/infiniflow/infinity/
4. https://infiniflow.org/docs/http_api_reference
5. https://github.com/michaelfeil/infinity
6. https://github.com/neurosnap/sentences
7. https://docs.aws.amazon.com/bedrock/latest/userguide/kb-chunking.html
8. https://github.com/isaacus-dev/semchunk
9. https://github.com/barbarabai/aesop

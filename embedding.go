package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"fmt"
)

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input []string `json:"input"`
}

type EmbeddingData struct {
	Object string `json:"object"`
	Embedding []float64 `json:"embedding"`
}

type EmbeddingResponse struct {
	Object string `json:"object"`
	Data []EmbeddingData `json:"data"`
}

func GetEmbeddings(input []string) ([][]float64, error) {
	var embeddings [][]float64

    payload := EmbeddingRequest{
		Model: "michaelfeil/bge-small-en-v1.5",
		Input: input,
    }

    jsonData, err := json.Marshal(payload)
    if err != nil {
        return embeddings, fmt.Errorf("Error marshaling JSON: %v\n", err)
    }

    resp, err := http.Post(
		"http://localhost:7997/embeddings",
        "application/json",
        bytes.NewReader(jsonData),
    )
    if err != nil {
        return embeddings, fmt.Errorf("Error making request: %v\n", err)
    }
    defer resp.Body.Close()

	var res EmbeddingResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
    if err != nil {
        return embeddings, fmt.Errorf("Error making request: %v\n", err)
    }

	for _, obj := range res.Data {
		embeddings = append(embeddings, obj.Embedding)
	}

	return embeddings, nil
}

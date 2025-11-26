package main

import (
	"net/http"
	"fmt"
	"bytes"
	"encoding/json"
)

type InfinityResponse struct {
	ErrorCode int `json:"error_code"`
	ErrorMsg *string `json:"error_msg"`
}

func InsertEmbeddings(input []VectorRow) error {
    jsonData, err := json.Marshal(input)
    if err != nil {
        return fmt.Errorf("Error marshaling JSON: %v\n", err)
    }

    resp, err := http.Post(
		"http://localhost:23820/databases/rfs/tables/data/docs",
        "application/json",
        bytes.NewReader(jsonData),
    )
    if err != nil {
        return fmt.Errorf("Error making request: %v\n", err)
    }
    defer resp.Body.Close()

	var res InfinityResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
    if err != nil {
        return fmt.Errorf("Error making request: %v\n", err)
    }
	if res.ErrorCode != 0 {
		return fmt.Errorf("Error inserting embeddings: %s", *res.ErrorMsg)
	}

	return nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
)

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

func SearchVectors(queryVector []float64, queryText string) ([]VectorRow, error) {
	//fm := "rrf"
	//l2 := "l2"
	//fl := "float"
    jsonData, err := json.Marshal(SearchRequest{
		Search: []MatchRequest{
			{
				MatchMethod: "text",
				TopN: 2,
				Field: "fulltext_column",
				MatchingText: &queryText,
				Params: &map[string]string{},
			},
		},
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

	// This is really horrible, but Infinity outputs in this format
	// [[{"name":"TODO: filename"},{"index":0},{"fulltext_column":"Saturday"}]]
	// This is not reflected in their documentation. I will make a PR... at some point
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

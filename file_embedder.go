package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ChunkAndEmbedFile(filename string) error {
	// Just dealing with text files for now. We will add OCR for other document types later
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

package main

import "log"

func main() {
	embeddings, err := GetEmbeddings([]string{"Test embedding", "Saturday"})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Got embeddings")
	err = InsertEmbeddings(embeddings)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Inserted embeddings")
}

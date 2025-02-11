package main

import (
	"fmt"
	"log"
	"sermersys/googlesearch"
)

func main() {
	testRequest := googlesearch.RequestData{
		HotelName:    "Ameron Abion",
		Address:      "",
		City:         "Berlin",
		Country:      "Germany",
		PlatformsFile: "platform2.txt",
	}

	filename, results, err := googlesearch.FetchData(testRequest)
	if err != nil {
		log.Fatalf("FetchData failed: %v", err)
	}

	fmt.Println("Generated file:", filename)
	fmt.Println("Results:", results)
}

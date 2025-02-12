package main

import (
	"fmt"
	"log"
	"sermersys/googlesearch"
	"sermersys/mapsearchg"
)

func main() {
	// Исходные входные данные
	initialRequest := mapsearchg.RequestData{
		ObjectName:    "Ameron Abion",
		Address:       "",
		City:          "Berlin",
		Country:       "Germany",
		PlatformsFile: "platform2.txt",
	}

	// Запрашиваем уточненные данные у mapsearchg
	filename, refinedResults, err := mapsearchg.FetchData(initialRequest)
	if err != nil {
		log.Fatalf("Ошибка в mapsearchg.FetchData: %v", err)
	}

	if len(refinedResults) == 0 {
		log.Fatalf("mapsearchg не нашел результатов для %s", initialRequest.ObjectName)
	}

	// Берем первый найденный результат
	refinedAddress := refinedResults[0]["formatted_address"]

	fmt.Println("Уточненный адрес:", refinedAddress)
	fmt.Println("Результаты сохранены в файл:", filename)

	// Формируем новый запрос для googlesearch
	searchRequest := googlesearch.RequestData{
		HotelName:     initialRequest.ObjectName,
		Address:       refinedAddress,
		City:          initialRequest.City,
		Country:       initialRequest.Country,
		PlatformsFile: initialRequest.PlatformsFile,
	}

	// Запускаем поиск в googlesearch с уточненными данными
	filename, searchResults, err := googlesearch.FetchData(searchRequest)
	if err != nil {
		log.Fatalf("Ошибка в googlesearch.FetchData: %v", err)
	}

	fmt.Println("Результаты поиска сохранены в файл:", filename)
	fmt.Println("Результаты поиска:", searchResults)
}

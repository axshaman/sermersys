package search

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

// RequestData - структура данных для поиска
type RequestData struct {
	HotelName    string `json:"hotel_name"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Country      string `json:"country"`
	PlatformsFile string `json:"platforms_file"`
}

// FetchData - функция для вызова без сети
func FetchData(data RequestData) (string, error) {
	// Загружаем платформы из файла
	platforms, err := loadPlatforms(data.PlatformsFile)
	if err != nil {
		return "", fmt.Errorf("Ошибка загрузки платформ: %v", err)
	}

	query := buildQuery(data.HotelName, data.City, data.Country)

	// Создание имени файла
	timestamp := time.Now().Format("150405020106")
	filename := fmt.Sprintf("%s-%s.csv", timestamp, strings.ReplaceAll(data.HotelName, " ", "_"))

	// Открываем CSV-файл
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("Ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заполняем CSV заголовками
	writer.Write([]string{"Platform", "Title", "Link"})

	// Эмулируем поиск на 3 страницах и анализируем платформы по 5 штук
	for i := 0; i < len(platforms); i += 5 {
		end := i + 5
		if end > len(platforms) {
			end = len(platforms)
		}

		platformSubset := platforms[i:end]
		links := findLinksMock(query, platformSubset, data.HotelName, 3) // Мок-функция вместо Google API

		for platform, item := range links {
			writer.Write([]string{platform, item.Title, item.Link})
		}
	}

	log.Println("Данные успешно сохранены в файл:", filename)
	return filename, nil
}

// loadPlatforms загружает список платформ из файла
func loadPlatforms(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines, nil
}

// buildQuery формирует поисковый запрос
func buildQuery(hotelName, city, country string) string {
	return fmt.Sprintf("%s %s %s", hotelName, city, country)
}

// findLinksMock - эмуляция поиска ссылок (заменяет Google API)
func findLinksMock(query string, platforms []string, hotelName string, maxPages int) map[string]struct{ Title, Link string } {
	links := make(map[string]struct{ Title, Link string })
	hotelWords := strings.Fields(strings.ToLower(hotelName))

	// Эмулируем 3 страницы выдачи
	for start := 1; start <= maxPages*10; start += 10 {
		for _, platform := range platforms {
			title := fmt.Sprintf("%s review on %s", hotelName, platform)
			if checkTitle(strings.ToLower(title), hotelWords) {
				links[platform] = struct{ Title, Link string }{
					Title: title,
					Link:  fmt.Sprintf("https://www.%s/review/%s", platform, strings.ReplaceAll(hotelName, " ", "-")),
				}
			}
		}
	}

	return links
}

// checkTitle проверяет, содержит ли заголовок хотя бы одно слово (если одно) или любую пару слов (если два и более)
func checkTitle(title string, words []string) bool {
	if len(words) == 1 {
		return strings.Contains(title, words[0])
	}

	wordPairs := generateWordPairs(words)

	for _, pair := range wordPairs {
		if strings.Contains(title, pair[0]) && strings.Contains(title, pair[1]) {
			return true
		}
	}

	return false
}

// generateWordPairs создает все возможные комбинации по 2 слова
func generateWordPairs(words []string) [][]string {
	var pairs [][]string
	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words); j++ {
			pairs = append(pairs, []string{words[i], words[j]})
		}
	}
	return pairs
}

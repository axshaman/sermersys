package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// CustomSearchResponse - структура для парсинга ответа от Google Custom Search API
type CustomSearchResponse struct {
	Items []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	} `json:"items"`
}

// PlaceDetails - структура для данных из Google Places API
type PlaceDetails struct {
	Result struct {
		Name             string `json:"name"`
		Rating           float64 `json:"rating"`
		UserRatingsTotal int    `json:"user_ratings_total"`
		Reviews          []struct {
			AuthorName string `json:"author_name"`
			Rating     int    `json:"rating"`
			Text       string `json:"text"`
		} `json:"reviews"`
	} `json:"result"`
}

// Config - структура для загрузки API-ключей из файла
type Config struct {
	GoogleAPIKey string `json:"google_api_key"`
	GoogleCX     string `json:"google_cx"`
}

// Список платформ для поиска
var platforms = []string{
	"tripadvisor.com",
	"booking.com",
	"expedia.com",
	"hotels.com",
	"agoda.com",
	"kayak.com",
	"priceline.com",
	"skyscanner.com",
	"trip.com",
}

func main() {
	// Загружаем конфиг
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Данные об отеле
	anName := "Abion Ameron Berlin"
	city := "Berlin"
	country := "Germany"

	query := buildQuery(anName, city, country)

	// Создание имени файла
	timestamp := time.Now().Format("150405020106")
	filename := fmt.Sprintf("%s-%s.csv", timestamp, strings.ReplaceAll(anName, " ", "_"))

	// Открываем CSV-файл
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Получаем данные из Google Places API
	details, err := getPlaceDetails(config.GoogleAPIKey, anName, city)
	if err != nil {
		log.Println("Ошибка при получении данных из Google Places API:", err)
	}

	writer.Write([]string{"Platform", "Title", "Link", "Rating", "User Ratings Total", "Review Author", "Review Rating", "Review Text"})

	for i := 0; i < len(platforms); i += 5 {
		end := i + 5
		if end > len(platforms) {
			end = len(platforms)
		}

		platformSubset := platforms[i:end]
		searchQuery := buildSearchQuery(query, platformSubset)
		links := findLinksGoogleCSE(searchQuery, config.GoogleAPIKey, config.GoogleCX, platformSubset, anName, 3)

		for platform, item := range links {
			rating := ""
			userRatingsTotal := ""
			reviewAuthor := ""
			reviewRating := ""
			reviewText := ""

			if details != nil {
				rating = fmt.Sprintf("%.1f", details.Result.Rating)
				userRatingsTotal = fmt.Sprintf("%d", details.Result.UserRatingsTotal)

				if len(details.Result.Reviews) > 0 {
					review := details.Result.Reviews[0]
					reviewAuthor = review.AuthorName
					reviewRating = fmt.Sprintf("%d", review.Rating)
					reviewText = review.Text
				}
			}

			writer.Write([]string{platform, item.Title, item.Link, rating, userRatingsTotal, reviewAuthor, reviewRating, reviewText})
		}
	}

	log.Println("Данные успешно сохранены в файл:", filename)
}

// loadConfig загружает API-ключи из config.json
func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// buildQuery формирует строку запроса
func buildQuery(anName, city, country string) string {
	return fmt.Sprintf("%s %s %s", anName, city, country)
}

// buildSearchQuery формирует строку запроса с site:
func buildSearchQuery(query string, platforms []string) string {
	var siteFilters []string
	for _, platform := range platforms {
		siteFilters = append(siteFilters, fmt.Sprintf("site:%s", platform))
	}
	return strings.Join(siteFilters, " OR ") + " " + query
}

// findLinksGoogleCSE ищет ссылки через Google Custom Search API
func findLinksGoogleCSE(query, apiKey, cx string, platforms []string, anName string, maxPages int) map[string]struct{ Title, Link string } {
	baseURL := "https://www.googleapis.com/customsearch/v1"
	links := make(map[string]struct{ Title, Link string })

	hotelWords := strings.Fields(strings.ToLower(anName))

	for start := 1; start <= maxPages*10; start += 10 {
		u, _ := url.Parse(baseURL)
		q := u.Query()
		q.Set("key", apiKey)
		q.Set("cx", cx)
		q.Set("q", query)
		q.Set("start", fmt.Sprintf("%d", start))
		u.RawQuery = q.Encode()

		resp, err := http.Get(u.String())
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		defer resp.Body.Close()

		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		var result CustomSearchResponse
		json.Unmarshal(bodyBytes, &result)

		for _, item := range result.Items {
			titleLower := strings.ToLower(item.Title)
			if checkTitle(titleLower, hotelWords) {
				for _, platform := range platforms {
					if strings.Contains(item.Link, platform) {
						links[platform] = struct{ Title, Link string }{Title: item.Title, Link: item.Link}
					}
				}
			}
		}
	}
	return links
}

// checkTitle проверяет, есть ли ключевые слова в заголовке
func checkTitle(title string, words []string) bool {
	if len(words) == 1 {
		return strings.Contains(title, words[0])
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })

	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words); j++ {
			if strings.Contains(title, words[i]) && strings.Contains(title, words[j]) {
				return true
			}
		}
	}
	return false
}

// getPlaceDetails получает рейтинг и отзывы из Google Places API
func getPlaceDetails(apiKey, hotelName, city string) (*PlaceDetails, error) {
	placeID := getPlaceID(apiKey, hotelName, city)
	if placeID == "" {
		return nil, fmt.Errorf("Place ID not found")
	}

	u, _ := url.Parse("https://maps.googleapis.com/maps/api/place/details/json")
	q := u.Query()
	q.Set("key", apiKey)
	q.Set("place_id", placeID)
	q.Set("fields", "name,rating,user_ratings_total,reviews")
	u.RawQuery = q.Encode()

	resp, _ := http.Get(u.String())
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var placeDetails PlaceDetails
	json.Unmarshal(body, &placeDetails)

	return &placeDetails, nil
}

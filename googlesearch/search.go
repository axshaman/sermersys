package googlesearch

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// RequestData - структура входных данных
type RequestData struct {
	HotelName    string `json:"hotel_name"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city"`
	Country      string `json:"country"`
	PlatformsFile string `json:"platforms_file"`
}

// CustomSearchResponse - структура ответа от Google CSE
type CustomSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

// PlaceDetails - структура ответа от Google Places API
type PlaceDetails struct {
	Result struct {
		Name             string  `json:"name"`
		Rating           float64 `json:"rating"`
		UserRatingsTotal int     `json:"user_ratings_total"`
		Reviews          []struct {
			AuthorName string `json:"author_name"`
			Rating     int    `json:"rating"`
			Text       string `json:"text"`
		} `json:"reviews"`
	} `json:"result"`
}

// Config - структура конфигурации
type Config struct {
	GoogleAPIKey string `json:"google_api_key"`
	GoogleCX     string `json:"google_cx"`
}

// FetchData - выполняет поиск, записывает CSV и возвращает JSON
func FetchData(data RequestData) (string, []map[string]string, error) {
	config, err := loadConfig("./googlesearch/config.json")
	if err != nil {
		return "", nil, fmt.Errorf("Ошибка загрузки конфигурации: %v", err)
	}

	platforms, err := loadPlatforms("./googlesearch/" + data.PlatformsFile)
	if err != nil {
		return "", nil, fmt.Errorf("Ошибка загрузки платформ: %v", err)
	}

	query := buildQuery(data.HotelName, data.City, data.Country)

	// Создаём CSV-файл
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s-%s.csv", timestamp, strings.ReplaceAll(data.HotelName, " ", "_"))
	file, err := os.Create(filename)
	if err != nil {
		return "", nil, fmt.Errorf("Ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	writer.Write([]string{"Platform", "Title", "Link", "Rating", "User Ratings", "Review Author", "Review Rating", "Review Text"})

	// Получаем рейтинг и отзывы из Google Places API
	details, err := getPlaceDetails(config.GoogleAPIKey, data.HotelName, data.City)
	if err != nil {
		log.Println("Ошибка при получении данных из Google Places API:", err)
	}

	// Запускаем поиск по 3 страницам
	results := []map[string]string{}
	for i := 0; i < len(platforms); i += 5 {
		end := i + 5
		if end > len(platforms) {
			end = len(platforms)
		}

		platformSubset := platforms[i:end]
		searchQuery := buildSearchQuery(query, platformSubset)
		links := findLinksGoogleCSE(searchQuery, config.GoogleAPIKey, config.GoogleCX, platformSubset, data.HotelName, 3)

		for platform, item := range links {
			rating, userRatingsTotal, reviewAuthor, reviewRating, reviewText := "", "", "", "", ""

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

			entry := map[string]string{
				"platform":      platform,
				"title":         item.Title,
				"link":          item.Link,
				"rating":        rating,
				"user_ratings":  userRatingsTotal,
				"review_author": reviewAuthor,
				"review_rating": reviewRating,
				"review_text":   reviewText,
			}
			results = append(results, entry)
			writer.Write([]string{platform, item.Title, item.Link, rating, userRatingsTotal, reviewAuthor, reviewRating, reviewText})
		}
	}

	log.Println("Данные успешно сохранены в файл:", filename)
	return filename, results, nil
}

// **Функция поиска Google CSE с поддержкой проверки заголовков**
func findLinksGoogleCSE(query, apiKey, cx string, platforms []string, hotelName string, maxPages int) map[string]struct{ Title, Link string } {
	baseURL := "https://www.googleapis.com/customsearch/v1"
	links := make(map[string]struct{ Title, Link string })
	hotelWords := strings.Fields(strings.ToLower(hotelName))

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

// **Функция проверки заголовков**
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

// **Генерация комбинаций слов (2 слова)**
func generateWordPairs(words []string) [][]string {
	var pairs [][]string
	for i := 0; i < len(words); i++ {
		for j := i + 1; j < len(words); j++ {
			pairs = append(pairs, []string{words[i], words[j]})
		}
	}
	return pairs
}

// **Формирование поискового запроса**
func buildQuery(hotelName, city, country string) string {
	return fmt.Sprintf("%s %s %s", hotelName, city, country)
}

// **Функция загрузки конфигурации**
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// **Функция загрузки списка платформ**
func loadPlatforms(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines, nil
}

// **Функция формирования поискового запроса для Google Custom Search**
func buildSearchQuery(query string, platforms []string) string {
	var siteFilters []string
	for _, platform := range platforms {
		siteFilters = append(siteFilters, fmt.Sprintf("site:%s", platform))
	}
	return strings.Join(siteFilters, " OR ") + " " + query
}
// getPlaceDetails получает информацию о месте из Google Places API
func getPlaceDetails(apiKey, hotelName, city string) (*PlaceDetails, error) {
	baseURL := "https://maps.googleapis.com/maps/api/place/findplacefromtext/json"
	u, _ := url.Parse(baseURL)

	// Формируем поисковый запрос
	query := fmt.Sprintf("%s, %s", hotelName, city)
	q := u.Query()
	q.Set("input", query)
	q.Set("inputtype", "textquery")
	q.Set("fields", "name,rating,user_ratings_total,reviews")
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()

	// Отправляем GET-запрос
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса к Google Places API: %v", err)
	}
	defer resp.Body.Close()

	// Читаем и парсим ответ
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа от Google Places API: %v", err)
	}

	var placeDetails PlaceDetails
	err = json.Unmarshal(bodyBytes, &placeDetails)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON от Google Places API: %v", err)
	}

	return &placeDetails, nil
}

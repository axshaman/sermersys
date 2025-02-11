package search

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

// FetchData - выполняет поиск, записывает CSV и возвращает JSON-ответ
func FetchData(data RequestData) (string, []map[string]string, error) {
	config, err := loadConfig("config.json")
	if err != nil {
		return "", nil, fmt.Errorf("Ошибка загрузки конфигурации: %v", err)
	}

	platforms, err := loadPlatforms(data.PlatformsFile)
	if err != nil {
		return "", nil, fmt.Errorf("Ошибка загрузки платформ: %v", err)
	}

	query := buildQuery(data.HotelName, data.City, data.Country)

	// Создаём CSV-файл
	timestamp := time.Now().Format("150405020106")
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
				"platform":        platform,
				"title":           item.Title,
				"link":            item.Link,
				"rating":          rating,
				"user_ratings":    userRatingsTotal,
				"review_author":   reviewAuthor,
				"review_rating":   reviewRating,
				"review_text":     reviewText,
			}
			results = append(results, entry)
			writer.Write([]string{platform, item.Title, item.Link, rating, userRatingsTotal, reviewAuthor, reviewRating, reviewText})
		}
	}

	log.Println("Данные успешно сохранены в файл:", filename)
	return filename, results, nil
}

// findLinksGoogleCSE - выполняет поиск через Google CSE
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

// getPlaceDetails получает данные из Google Places API
func getPlaceDetails(apiKey, hotelName, city string) (*PlaceDetails, error) {
	u, _ := url.Parse("https://maps.googleapis.com/maps/api/place/details/json")
	q := u.Query()
	q.Set("key", apiKey)
	q.Set("input", fmt.Sprintf("%s, %s", hotelName, city))
	q.Set("inputtype", "textquery")
	q.Set("fields", "place_id")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ошибка запроса к Google Places API: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var details PlaceDetails
	json.Unmarshal(body, &details)

	return &details, nil
}

// Дополнительные функции
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

func loadPlatforms(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	return lines, nil
}

func buildSearchQuery(query string, platforms []string) string {
	var siteFilters []string
	for _, platform := range platforms {
		siteFilters = append(siteFilters, fmt.Sprintf("site:%s", platform))
	}
	return strings.Join(siteFilters, " OR ") + " " + query
}

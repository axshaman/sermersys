// sermersys/mapsearchg/mapsearchg.go
package mapsearchg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// =================== Структуры ===================

// Config - структура конфигурации
type Config struct {
	GoogleAPIKey string `json:"google_api_key"`
}

// RequestData - структура входных данных
type RequestData struct {
	ObjectName    string `json:"object_name"`
	Address       string `json:"address,omitempty"`
	City          string `json:"city"`
	Country       string `json:"country"`
	PlatformsFile string `json:"platforms_file"`
}

// APIResponse - структура ответа API
type APIResponse struct {
	Results []FinalData `json:"results"`
	Error   string      `json:"error,omitempty"`
}

// Структуры для Text Search API
type TextSearchResponse struct {
	Results []TextSearchResult `json:"results"`
	Status  string             `json:"status"`
}

type TextSearchResult struct {
	Name             string `json:"name"`
	FormattedAddress string `json:"formatted_address"`
	PlaceID          string `json:"place_id"`
	Geometry         struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
	Rating           float64 `json:"rating,omitempty"`
	UserRatingsTotal int     `json:"user_ratings_total,omitempty"`
}

// Структуры для Place Details API
type PlaceDetailsResponse struct {
	Result PlaceDetailsResult `json:"result"`
	Status string             `json:"status"`
}

type PlaceDetailsResult struct {
	Name                 string  `json:"name"`
	FormattedAddress     string  `json:"formatted_address"`
	PlaceID              string  `json:"place_id"`
	FormattedPhoneNumber string  `json:"formatted_phone_number"`
	Website              string  `json:"website"`
	Rating               float64 `json:"rating"`
	UserRatingsTotal     int     `json:"user_ratings_total"`
	Geometry             struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
	} `json:"geometry"`
}

// Итоговые данные
type FinalData struct {
	Timestamp        string  `json:"timestamp"`
	Name             string  `json:"name"`
	FormattedAddress string  `json:"formatted_address"`
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	PlaceID          string  `json:"place_id"`
	Website          string  `json:"website"`
	Phone            string  `json:"phone"`
	Rating           float64 `json:"rating"`
	UserRatingsTotal int     `json:"user_ratings_total"`
}

// =================== Функция загрузки API-ключа ===================

// loadConfig загружает конфигурацию из файла
func loadConfig(filename string) (*Config, error) {
	// Определяем абсолютный путь к конфигурационному файлу
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка определения абсолютного пути: %v", err)
	}

	data, err := ioutil.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения конфигурационного файла: %v", err)
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурационного файла: %v", err)
	}
	return &config, nil
}

// =================== Text Search API ===================

// doTextSearch вызывает Places Text Search API и возвращает срез результатов
func doTextSearch(apiKey, query string) ([]TextSearchResult, error) {
	baseURL := "https://maps.googleapis.com/maps/api/place/textsearch/json"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("невалидный baseURL: %w", err)
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("ошибка GET запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения Body: %w", err)
	}

	var tsr TextSearchResponse
	err = json.Unmarshal(body, &tsr)
	if err != nil {
		return nil, fmt.Errorf("ошибка Unmarshal: %w", err)
	}

	// Если статус не OK, вернём пустой массив
	if tsr.Status != "OK" {
		log.Printf("TextSearch status=%s, возможно ZERO_RESULTS или другая проблема\n", tsr.Status)
		return nil, nil
	}
	return tsr.Results, nil
}

// =================== Place Details API ===================

// doPlaceDetails вызывает Places Details API и возвращает результат
func doPlaceDetails(apiKey, placeID string) (*PlaceDetailsResult, error) {
	baseURL := "https://maps.googleapis.com/maps/api/place/details/json"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка разбора URL: %v", err)
	}
	q := u.Query()
	q.Set("place_id", placeID)
	q.Set("fields", "formatted_address,name,geometry,place_id,website,formatted_phone_number,rating,user_ratings_total")
	q.Set("key", apiKey)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("ошибка GET запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения Body: %w", err)
	}

	var pdr PlaceDetailsResponse
	err = json.Unmarshal(body, &pdr)
	if err != nil {
		return nil, fmt.Errorf("ошибка Unmarshal: %w", err)
	}

	if pdr.Status != "OK" {
		return nil, fmt.Errorf("Google API вернул статус: %s", pdr.Status)
	}

	return &pdr.Result, nil
}

// =================== Функция поиска ===================

// SearchGooglePlaces выполняет поиск через Google Places API и возвращает итоговые данные
func SearchGooglePlaces(data RequestData) ([]FinalData, error) {
	// Формируем поисковый запрос
	query := fmt.Sprintf("%s, %s, %s", data.ObjectName, data.City, data.Country)

	// Загружаем конфигурацию
	config, err := loadConfig("./config.json")
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфигурации: %v", err)
	}

	// Выполняем текстовый поиск
	textResults, err := doTextSearch(config.GoogleAPIKey, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка doTextSearch: %v", err)
	}

	if len(textResults) == 0 {
		return nil, fmt.Errorf("нет результатов для запроса: %s", query)
	}

	var finalResults []FinalData
	for _, r := range textResults {
		details, err := doPlaceDetails(config.GoogleAPIKey, r.PlaceID)
		if err != nil {
			log.Printf("Не удалось получить детали для place_id=%s: %v", r.PlaceID, err)
			continue
		}

		finalResults = append(finalResults, FinalData{
			Timestamp:        time.Now().Format(time.RFC3339),
			Name:             details.Name,
			FormattedAddress: details.FormattedAddress,
			Lat:              details.Geometry.Location.Lat,
			Lng:              details.Geometry.Location.Lng,
			PlaceID:          details.PlaceID,
			Website:          details.Website,
			Phone:            details.FormattedPhoneNumber,
			Rating:           details.Rating,
			UserRatingsTotal: details.UserRatingsTotal,
		})
	}

	return finalResults, nil
}

// =================== Сохранение результатов ===================

// saveToCSV сохраняет результаты в CSV-файл и возвращает имя файла
func saveToCSV(data []FinalData) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("нет данных для сохранения в CSV")
	}

	// Создаём директорию для результатов, если не существует
	dir := "./results/"
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("ошибка создания директории: %v", err)
	}

	// Создаём CSV-файл
	timestamp := time.Now().Format("20060102150405")
	nameSafe := strings.ReplaceAll(data[0].Name, " ", "_")
	filename := fmt.Sprintf("%sresults_%s_%s.csv", dir, timestamp, nameSafe)
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовки
	headers := []string{
		"Timestamp",
		"Name",
		"FormattedAddress",
		"Lat",
		"Lng",
		"PlaceID",
		"Website",
		"Phone",
		"Rating",
		"UserRatingsTotal",
	}
	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("ошибка записи заголовков: %v", err)
	}

	// Записываем данные
	for _, row := range data {
		record := []string{
			row.Timestamp,
			row.Name,
			row.FormattedAddress,
			fmt.Sprintf("%f", row.Lat),
			fmt.Sprintf("%f", row.Lng),
			row.PlaceID,
			row.Website,
			row.Phone,
			fmt.Sprintf("%.2f", row.Rating),
			strconv.Itoa(row.UserRatingsTotal),
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("ошибка записи записи: %v", err)
		}
	}

	return filename, nil
}

// =================== Функция FetchData ===================

// FetchData выполняет поиск, сохраняет результаты в CSV и возвращает имя файла и результаты
func FetchData(data RequestData) (string, []map[string]string, error) {
	// Выполняем поиск
	finalResults, err := SearchGooglePlaces(data)
	if err != nil {
		return "", nil, fmt.Errorf("ошибка SearchGooglePlaces: %v", err)
	}

	// Сохраняем результаты в CSV
	filename, err := saveToCSV(finalResults)
	if err != nil {
		return "", nil, fmt.Errorf("ошибка saveToCSV: %v", err)
	}

	// Преобразуем результаты в []map[string]string для совместимости с main.go
	var searchResults []map[string]string
	for _, fr := range finalResults {
		entry := map[string]string{
			"timestamp":          fr.Timestamp,
			"name":               fr.Name,
			"formatted_address":  fr.FormattedAddress,
			"lat":                fmt.Sprintf("%f", fr.Lat),
			"lng":                fmt.Sprintf("%f", fr.Lng),
			"place_id":           fr.PlaceID,
			"website":            fr.Website,
			"phone":              fr.Phone,
			"rating":             fmt.Sprintf("%.2f", fr.Rating),
			"user_ratings_total": strconv.Itoa(fr.UserRatingsTotal),
		}
		searchResults = append(searchResults, entry)
	}

	return filename, searchResults, nil
}

// =================== HTTP-Обработчик ===================

// Handler обрабатывает HTTP-запросы и возвращает результаты поиска
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("Received request for mapsearchg: %+v", requestData)

	// Выполняем FetchData
	filename, results, err := FetchData(requestData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем JSON-ответ
	response := APIResponse{
		Results: convertFinalDataToFinalData(results),
	}

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"filename": filename,
		"results":  response,
	})
}

// convertFinalDataToFinalData преобразует []map[string]string в []FinalData
func convertFinalDataToFinalData(data []map[string]string) []FinalData {
	var finalData []FinalData
	for _, entry := range data {
		lat, _ := strconv.ParseFloat(entry["lat"], 64)
		lng, _ := strconv.ParseFloat(entry["lng"], 64)
		rating, _ := strconv.ParseFloat(entry["rating"], 64)
		userRatingsTotal, _ := strconv.Atoi(entry["user_ratings_total"])

		finalData = append(finalData, FinalData{
			Timestamp:        entry["timestamp"],
			Name:             entry["name"],
			FormattedAddress: entry["formatted_address"],
			Lat:              lat,
			Lng:              lng,
			PlaceID:          entry["place_id"],
			Website:          entry["website"],
			Phone:            entry["phone"],
			Rating:           rating,
			UserRatingsTotal: userRatingsTotal,
		})
	}
	return finalData
}

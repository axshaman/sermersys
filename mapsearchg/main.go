package main

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

// ------------ Структуры для Text Search ответа ------------
type TextSearchResponse struct {
    Results []TextSearchResult `json:"results"`
    Status  string             `json:"status"`
}

type TextSearchResult struct {
    Name            string `json:"name"`
    FormattedAddress string `json:"formatted_address"`
    PlaceID         string `json:"place_id"`
    Geometry        struct {
        Location struct {
            Lat float64 `json:"lat"`
            Lng float64 `json:"lng"`
        } `json:"location"`
    } `json:"geometry"`
    Rating           float64 `json:"rating,omitempty"`
    UserRatingsTotal int     `json:"user_ratings_total,omitempty"`
}

// ------------ Структуры для Place Details ответа ------------
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

// ------------ Основная логика ------------
func main() {
    // -- 1) Пример входных данных (можно читать из флагов или окружения) --
    hotelName := "Fletcher Hotel"
    street := "Schepenbergweg"
    house := "50"
    city := "Amsterdam"
    country := "Netherlands"

    // Ваш API-ключ для Places API (не храните в коде в продакшне!)
    apiKey := "AIzaSyA3mv4-qynFTK4LPs3s4eIOiVXhNlWWX7Y"

    // Склеим строку для Text Search query
    queryStr := buildQueryString(hotelName, street, house, city, country)

    // -- 2) Делаем Text Search запрос, получаем список place_id и прочее --
    textResults, err := doTextSearch(apiKey, queryStr)
    if err != nil {
        log.Fatal("Ошибка Text Search:", err)
    }
    if len(textResults) == 0 {
        log.Printf("Ничего не найдено (или статус != OK) для запроса: %s\n", queryStr)
        return
    }

    // -- 3) Для каждого результата вызываем Place Details и собираем финальные данные --
    var finalRows []FinalData

    for _, r := range textResults {
        details, err := doPlaceDetails(apiKey, r.PlaceID)
        if err != nil {
            // Если не получилось получить детали для конкретного места — пишем в лог и пропускаем
            log.Printf("Ошибка Place Details для place_id=%s: %v\n", r.PlaceID, err)
            continue
        }

        // Формируем итоговую запись (берём данные из Place Details, fallback на TextSearch если нужно)
        row := FinalData{
            AnalysisTimestamp: time.Now().Format(time.RFC3339),
            Name:              details.Name,
            FormattedAddress:  details.FormattedAddress,
            Lat:               details.Geometry.Location.Lat,
            Lng:               details.Geometry.Location.Lng,
            PlaceID:           details.PlaceID,
            Website:           details.Website,
            Phone:             details.FormattedPhoneNumber,
            Rating:            details.Rating,
            UserRatingsTotal:  details.UserRatingsTotal,
        }

        // Если в Details не вернулся адрес (бывает, Google скрывает), возьмём из TextSearch
        if row.FormattedAddress == "" {
            row.FormattedAddress = r.FormattedAddress
        }
        // Аналогично Lat/Lng, Rating, UserRatingsTotal можно сравнивать
        // (но чаще данные из Details точнее/свежее)

        finalRows = append(finalRows, row)
    }

    // -- 4) Сохраняем результаты в CSV --
    if err := saveToCSV(finalRows, "results.csv"); err != nil {
        log.Fatal("Ошибка сохранения в CSV:", err)
    }

    fmt.Printf("Успешно сохранено %d записей в results.csv\n", len(finalRows))
}

// ========== Вспомогательные функции ==========

// buildQueryString собирает строку вида
// "Fletcher Hotel, Schepenbergweg 50, Amsterdam, Netherlands"
func buildQueryString(hotelName, street, house, city, country string) string {
    var sb strings.Builder
    sb.WriteString(hotelName)
    if street != "" || house != "" {
        sb.WriteString(", ")
        sb.WriteString(street)
        if house != "" {
            sb.WriteString(" ")
            sb.WriteString(house)
        }
    }
    if city != "" {
        sb.WriteString(", ")
        sb.WriteString(city)
    }
    if country != "" {
        sb.WriteString(", ")
        sb.WriteString(country)
    }
    return sb.String()
}

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
    if err := json.Unmarshal(body, &tsr); err != nil {
        return nil, fmt.Errorf("ошибка Unmarshal: %w", err)
    }

    // Если статус не OK, вернём пустой массив
    if tsr.Status != "OK" {
        log.Printf("TextSearch status=%s, возможно ZERO_RESULTS или другая проблема\n", tsr.Status)
        return nil, nil
    }
    return tsr.Results, nil
}

// doPlaceDetails вызывает Place Details API для получения website, phone, rating и т.д.
func doPlaceDetails(apiKey, placeID string) (*PlaceDetailsResult, error) {
    baseURL := "https://maps.googleapis.com/maps/api/place/details/json"
    u, err := url.Parse(baseURL)
    if err != nil {
        return nil, err
    }
    // Перечисляем нужные поля через запятую (не запрашиваем лишнее для экономии)
    fields := "formatted_address,name,geometry,place_id,website,formatted_phone_number,rating,user_ratings_total"

    q := u.Query()
    q.Set("place_id", placeID)
    q.Set("fields", fields)
    q.Set("key", apiKey)
    u.RawQuery = q.Encode()

    resp, err := http.Get(u.String())
    if err != nil {
        return nil, fmt.Errorf("ошибка GET запроса place details: %w", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("ошибка чтения Body place details: %w", err)
    }

    var pdr PlaceDetailsResponse
    if err := json.Unmarshal(body, &pdr); err != nil {
        return nil, fmt.Errorf("ошибка Unmarshal place details: %w", err)
    }

    if pdr.Status != "OK" {
        return nil, fmt.Errorf("Place Details статус=%s", pdr.Status)
    }

    return &pdr.Result, nil
}

// FinalData описывает итоговую строку для CSV
type FinalData struct {
    AnalysisTimestamp string  // Дата-время выгрузки
    Name              string
    FormattedAddress  string
    Lat               float64
    Lng               float64
    PlaceID           string
    Website           string
    Phone             string
    Rating            float64
    UserRatingsTotal  int
}

// saveToCSV сохраняет все записи в CSV-файл
func saveToCSV(data []FinalData, filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    // Заголовок
    headers := []string{
        "AnalysisTimestamp",
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
        return fmt.Errorf("ошибка записи заголовка CSV: %w", err)
    }

    // Строки
    for _, row := range data {
        record := []string{
            row.AnalysisTimestamp,
            row.Name,
            row.FormattedAddress,
            fmt.Sprintf("%f", row.Lat),
            fmt.Sprintf("%f", row.Lng),
            row.PlaceID,
            row.Website,
            row.Phone,
            fmt.Sprintf("%.2f", row.Rating),
            fmt.Sprintf("%d", row.UserRatingsTotal),
        }
        if err := writer.Write(record); err != nil {
            return fmt.Errorf("ошибка записи строки CSV: %w", err)
        }
    }

    return nil
}

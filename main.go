package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sermersys/googlesearch"
	"sermersys/mapsearchg"
	"time"
	"path/filepath"
)

// Структура ответа API
type APIResponse struct {
	RefinedHotelName string              `json:"refined_hotel_name"` // Добавлено уточнённое имя
	RefinedAddress   string              `json:"refined_address"`
	SearchResults    []map[string]string `json:"search_results"`
	ExecutionSteps   []string            `json:"execution_steps"`
	Error            string              `json:"error,omitempty"`
}


// =================== API-Обработчик ===================
func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Читаем JSON-запрос
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения тела запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Используем структуру RequestData из пакета mapsearchg
	var requestData mapsearchg.RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Получен запрос: %+v", requestData)

	// Логирование времени начала обработки
	startTime := time.Now()

	// 1️⃣ **Запрашиваем данные у `mapsearchg`**
	refinedData, err := mapsearchg.SearchGooglePlaces(requestData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка в mapsearchg.SearchGooglePlaces: %v", err), http.StatusInternalServerError)
		return
	}

	if len(refinedData) == 0 {
		http.Error(w, "Нет результатов в mapsearchg", http.StatusNotFound)
		return
	}

	// Обновляем запрос с уточнённым адресом
	updatedRequest := googlesearch.RequestData{
		HotelName:     refinedData[0].Name,
		Address:       refinedData[0].FormattedAddress,
		City:          requestData.City,
		Country:       requestData.Country,
		PlatformsFile: requestData.PlatformsFile,
	}

	log.Printf("Уточнённое имя из mapsearchg: %s", updatedRequest.HotelName)
	log.Printf("Уточнённый адрес из mapsearchg: %s", updatedRequest.Address)

	// 2️⃣ **Запускаем `googlesearch.FetchData` с уточнёнными данными**
	filename, searchResults, err := googlesearch.FetchData(updatedRequest)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка в googlesearch.FetchData: %v", err), http.StatusInternalServerError)
		return
	}

	// Логирование времени окончания обработки
	executionTime := time.Since(startTime)

	// 3️⃣ **Формируем финальный ответ**
	response := APIResponse{
		RefinedHotelName: updatedRequest.HotelName, // Добавляем уточнённое имя
		RefinedAddress:   updatedRequest.Address,
		SearchResults:    searchResults,
		ExecutionSteps: []string{
			"1️⃣ Запрос в mapsearchg для получения точного имени и адреса",
			"2️⃣ Уточнённые данные получены от mapsearchg",
			"3️⃣ Запрос в googlesearch.FetchData с уточнёнными данными",
			"4️⃣ Итоговый анализ завершён",
			fmt.Sprintf("⏳ Время выполнения: %v", executionTime),
		},
	}
	

	// Логирование итогового результата
	log.Printf("Итоговое время выполнения: %v", executionTime)
	log.Printf("Результаты поиска сохранены в файл: %s", filename)

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Ошибка при отправке ответа: %v", err)
	}
}

// =================== Обработчик HTML ===================
func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

// =================== Обработчик скачивания ===================
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	// Определяем абсолютный путь к файлу
	filePath, err := filepath.Abs(file)
	if err != nil {
		http.Error(w, "Invalid file path", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для скачивания
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, filePath)
}

// =================== Запуск сервера ===================
func main() {
	http.HandleFunc("/", homeHandler)    // Загружаем HTML-страницу
	http.HandleFunc("/process", handler) // API-обработчик
	http.HandleFunc("/download", downloadHandler) // Новый маршрут для скачивания

	log.Println("Server running on port 7001")
	err := http.ListenAndServe(":7001", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

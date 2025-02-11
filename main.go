package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"sermersys/googlesearch" // Импортируем наш модуль поиска
)

// RequestData - структура данных для API-запроса
type RequestData struct {
	HotelName    string `json:"hotel_name"`
	Address      string `json:"address,omitempty"`
	City         string `json:"city"`
	Country      string `json:"country"`
	PlatformsFile string `json:"platforms_file"`
}

// API-обработчик, принимает JSON-запрос, запускает поиск и возвращает результат
func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Разбираем JSON из запроса
	var requestData RequestData
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Ошибка разбора JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Запущен анализ: %+v\n", requestData)

	// Вызываем `FetchData` из `search.go`
	filename, result, err := search.FetchData(requestData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка анализа: %v", err), http.StatusInternalServerError)
		return
	}

	// Возвращаем JSON-ответ
	response := map[string]interface{}{
		"filename": filename,
		"results":  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Веб-страница для запроса данных у пользователя
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
	<html>
		<head><title>Анализатор отзывов</title></head>
		<body>
			<h2>Введите данные для анализа:</h2>
			<form action="/search" method="POST">
				<label>Тип объекта:</label>
				<select name="platforms_file">
					<option value="platforms1.txt">Кафе</option>
					<option value="platforms2.txt">Отель</option>
				</select><br><br>
				<label>Название объекта:</label> <input type="text" name="hotel_name" required><br><br>
				<label>Адрес (необязательно):</label> <input type="text" name="address"><br><br>
				<label>Город:</label> <input type="text" name="city" required><br><br>
				<label>Страна:</label> <input type="text" name="country" required><br><br>
				<input type="submit" value="Запустить анализ">
			</form>
		</body>
	</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	http.HandleFunc("/", homeHandler)  // Страница ввода
	http.HandleFunc("/search", searchHandler) // Обработчик API

	log.Println("Сервер запущен на порту 7001")
	log.Fatal(http.ListenAndServe(":7001", nil))
}

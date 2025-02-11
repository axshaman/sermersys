package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"sermersys/googlesearch" // Importing the search module
)

// API handler that accepts a JSON request, starts the search, and returns the result
func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the raw request body
	body, err := io.ReadAll(r.Body) // Replaced ioutil.ReadAll with io.ReadAll
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	log.Println("Raw JSON received:", string(body)) // Log raw JSON for debugging

	// Parse JSON from request
	var requestData googlesearch.RequestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		log.Println("JSON Parsing Error:", err) // Log the parsing error
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Parsed Request Data: %+v\n", requestData)

	// Call FetchData
	filename, result, err := googlesearch.FetchData(requestData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Analysis error: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	response := map[string]interface{}{
		"filename": filename,
		"results":  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Web page for user data input
func homeHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<html>
	<head><title>Review Analyzer</title></head>
	<body>
		<h2>Enter data for analysis:</h2>
		<form id="analyzeForm">
			<label>Object Type:</label>
			<select id="platforms_file">
				<option value="platform1.txt">Cafe</option>
				<option value="platform2.txt">Hotel</option>
			</select><br><br>
			<label>Object Name:</label> <input type="text" id="hotel_name" required><br><br>
			<label>Address (optional):</label> <input type="text" id="address"><br><br>
			<label>City:</label> <input type="text" id="city" required><br><br>
			<label>Country:</label> <input type="text" id="country" required><br><br>
			<button type="button" onclick="sendRequest()">Start Analysis</button>
		</form>

		<script>
			function sendRequest() {
				const data = {
					platforms_file: document.getElementById("platforms_file").value,
					hotel_name: document.getElementById("hotel_name").value,
					address: document.getElementById("address").value,
					city: document.getElementById("city").value,
					country: document.getElementById("country").value
				};

				fetch('/search', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(data)
				}).then(response => response.json())
				  .then(result => console.log(result))
				  .catch(error => console.error('Error:', error));
			}
		</script>
	</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func main() {
	http.HandleFunc("/", homeHandler)  // Input page
	http.HandleFunc("/search", searchHandler) // API handler

	log.Println("Server started on port 7001")
	log.Fatal(http.ListenAndServe(":7001", nil))
}

# 📌 Sermersys Search Tools

This repository contains two Go-based search tools:

1. **Map Searching** (`sermersys/mapsearching/main.go`) – Handles location-based searches and mapping services.
2. **Google Searching** (`sermersys/googlesearch/main.go`) – Performs Google Custom Search API queries to extract relevant results.

## 🚀 Features

### **1. Map Searching (`sermersys/mapsearching/main.go`)**
- Performs searches for locations based on input criteria.
- Retrieves geolocation data and maps relevant results.
- Ideal for integration with mapping APIs (Google Maps, OpenStreetMap, etc.).

### **2. Google Searching (`sermersys/googlesearch/main.go`)**
- Uses the Google Custom Search API to fetch search results from specific sources.
- Supports filtering and ranking of results.
- Extracts structured information from the search output.

## 🛠 Installation

Ensure you have **Go installed** on your system. Then, clone the repository:

```sh
git clone https://github.com/your-repo/sermersys.git
cd sermersys
```

## 🔧 Usage

### Run Map Searching
```sh
cd mapsearching
go run main.go
```

### Run Google Searching
```sh
cd googlesearch
go run main.go
```

## 📜 Configuration
Both scripts may require an API key for external services like Google Maps or Google Custom Search. Ensure you have a `config.json` file with the appropriate credentials:

```json
{
  "google_api_key": "YOUR_API_KEY",
  "google_cx": "YOUR_CUSTOM_SEARCH_ENGINE_ID"
}
```

## 📄 License
This project is licensed under the MIT License.

## 👨‍💻 Contributing
Feel

# sermersys

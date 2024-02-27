/*
**
kaushik
**
*/
package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Total        string `json:"total"`
	Items        []Item `json:"items"`
}

type Response struct {
	ID string `json:"id"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

var receipts = make(map[string]Receipt)
var mu sync.Mutex

func ProcessReceiptsHandler(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt

	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()

	mu.Lock()
	receipts[id] = receipt
	mu.Unlock()

	response := Response{ID: id}
	jsonResponse(w, response)
}

func GetPointsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	receiptID := params["id"]

	mu.Lock()
	receipt, found := receipts[receiptID]
	mu.Unlock()

	if !found {
		http.NotFound(w, r)
		return
	}
	points := calculatePoints(receipt)

	response := PointsResponse{Points: points}
	jsonResponse(w, response)
}

func jsonResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func calculatePoints(receipt Receipt) int {
	points := 0

	// Retailer Name Points
	for _, r := range receipt.Retailer {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			points++
		}
	}

	// Round Dollar Total Points and Multiple of 0.25 Points
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil {
		if total == math.Floor(total) {
			points += 50
		} else if math.Mod(total*100, 25) == 0 {
			points += 25
		}
	}

	points += (len(receipt.Items) / 2) * 5

	// Item Description Points
	for _, item := range receipt.Items {
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				points += int(math.Ceil(price * 0.2))
			}
		}
	}

	// Odd Day Points
	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err == nil && purchaseDate.Day()%2 != 0 {
		points += 6
	}
    // Specific Time Points
	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err == nil {
		if purchaseTime.Hour() > 14 && purchaseTime.Hour() < 16 {
			points += 10
		}
	}

	return points
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/receipts/process", ProcessReceiptsHandler).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", GetPointsHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

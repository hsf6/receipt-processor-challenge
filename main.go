package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Receipt struct {
	ID           string `json:"id,omitempty"`
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
	Points       int    `json:"-"`
	Breakdown    []string
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

var receipts = make(map[string]Receipt)
var mutex = &sync.Mutex{}

func main() {
	log.Println("Starting Receipt Processor server...")
	http.HandleFunc("/receipts/process", logRequest(processReceipt))
	http.HandleFunc("/receipts/", logRequest(handleRequests))

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Middleware to log incoming requests
func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request for %s", r.Method, r.URL.Path)
		handler(w, r)
	}
}

func processReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		log.Printf("Invalid method: %s. Only POST allowed.", r.Method)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		log.Printf("Error decoding JSON: %v", err)
		return
	}

	// Validate receipt
	if err := validateReceipt(receipt); err != nil {
		http.Error(w, fmt.Sprintf("Invalid receipt: %v", err), http.StatusBadRequest)
		log.Printf("Validation failed: %v", err)
		return
	}

	// Generate a unique ID and calculate points with breakdown
	receipt.ID = uuid.NewString()
	receipt.Points, receipt.Breakdown = calculatePoints(receipt)

	// Store in memory
	mutex.Lock()
	receipts[receipt.ID] = receipt
	mutex.Unlock()

	log.Printf("Receipt processed successfully. ID: %s, Points: %d", receipt.ID, receipt.Points)

	// Respond with ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": receipt.ID})
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		log.Printf("Invalid method: %s. Only GET allowed.", r.Method)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/receipts/")
	if strings.HasSuffix(id, "/points") {
		id = strings.TrimSuffix(id, "/points")
		getPoints(w, id)
	} else if strings.HasSuffix(id, "/breakdown") {
		id = strings.TrimSuffix(id, "/breakdown")
		getBreakdown(w, id)
	} else {
		http.Error(w, "Invalid endpoint", http.StatusNotFound)
		log.Printf("Invalid endpoint: %s", r.URL.Path)
	}
}

func getPoints(w http.ResponseWriter, id string) {
	if !isValidUUID(id) {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		log.Printf("Invalid UUID format: %s", id)
		return
	}

	mutex.Lock()
	receipt, found := receipts[id]
	mutex.Unlock()

	if !found {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		log.Printf("Receipt not found for ID: %s", id)
		return
	}

	log.Printf("Points retrieved for Receipt ID: %s, Points: %d", id, receipt.Points)

	// Respond with points
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"points": receipt.Points})
}

func getBreakdown(w http.ResponseWriter, id string) {
	if !isValidUUID(id) {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		log.Printf("Invalid UUID format: %s", id)
		return
	}

	mutex.Lock()
	receipt, found := receipts[id]
	mutex.Unlock()

	if !found {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		log.Printf("Receipt not found for ID: %s", id)
		return
	}

	log.Printf("Breakdown retrieved for Receipt ID: %s", id)

	// Respond with breakdown
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"points":    receipt.Points,
		"breakdown": receipt.Breakdown,
	}
	json.NewEncoder(w).Encode(response)
}

func calculatePoints(receipt Receipt) (int, []string) {
	points := 0
	breakdown := []string{}

	// Rule 1: Alphanumeric characters in retailer name
	retailerPoints := countAlphanumeric(receipt.Retailer)
	points += retailerPoints
	breakdown = append(breakdown, fmt.Sprintf("%d points - retailer name (%s) has %d alphanumeric characters", retailerPoints, receipt.Retailer, retailerPoints))

	// Rule 2: Total is a round dollar amount
	total, _ := strconv.ParseFloat(receipt.Total, 64)
	if total == float64(int(total)) {
		points += 50
		breakdown = append(breakdown, "50 points - total is a round dollar amount with no cents")
	}

	// Rule 3: Total is a multiple of 0.25
	if math.Mod(total, 0.25) == 0 {
		points += 25
		breakdown = append(breakdown, "25 points - total is a multiple of 0.25")
	}

	// Rule 4: 5 points for every two items
	itemPoints := (len(receipt.Items) / 2) * 5
	points += itemPoints
	breakdown = append(breakdown, fmt.Sprintf("%d points - %d items (%d pairs @ 5 points each)", itemPoints, len(receipt.Items), len(receipt.Items)/2))

	// Rule 5: Description length and price points
	for _, item := range receipt.Items {
		price, _ := strconv.ParseFloat(item.Price, 64)
		descLength := len(strings.TrimSpace(item.ShortDescription))
		if descLength%3 == 0 {
			totalPrice := price * 0.2
			itemPoints := int(math.Ceil(totalPrice))
			points += itemPoints
			breakdown = append(breakdown, fmt.Sprintf("%d points - \"%s\" is %d characters (a multiple of 3), item price %.2f * 0.2 = %.2f which is rounded to: %d points", itemPoints, strings.TrimSpace(item.ShortDescription), descLength, price, totalPrice, itemPoints))
		}
	}

	// Rule 6: Day of purchase is odd
	date, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if date.Day()%2 != 0 {
		points += 6
		breakdown = append(breakdown, "6 points - purchase day is odd")
	}

	// Rule 7: Purchase time between 2:00pm and 4:00pm
	time, _ := time.Parse("15:04", receipt.PurchaseTime)
	if time.Hour() == 14 || time.Hour() == 15 {
		points += 10
		breakdown = append(breakdown, "10 points - purchase time is between 2:00pm and 4:00pm")
	}

	log.Printf("Points calculated for receipt: %d", points)

	return points, breakdown
}

func validateReceipt(receipt Receipt) error {
	// Validate Retailer
	if receipt.Retailer == "" {
		log.Println("Validation failed: Retailer name is empty")
		return errors.New("retailer name is invalid")
	}
	if !regexp.MustCompile(`^[\w\s\-\&]+$`).MatchString(receipt.Retailer) {
		log.Printf("Validation failed: Retailer name '%s' contains invalid characters", receipt.Retailer)
		return errors.New("retailer name is invalid")
	}

	// Validate PurchaseDate
	if _, err := time.Parse("2006-01-02", receipt.PurchaseDate); err != nil {
		log.Printf("Validation failed: PurchaseDate '%s' is not in YYYY-MM-DD format", receipt.PurchaseDate)
		return errors.New("purchaseDate must be in YYYY-MM-DD format")
	}

	// Validate PurchaseTime
	if _, err := time.Parse("15:04", receipt.PurchaseTime); err != nil {
		log.Printf("Validation failed: PurchaseTime '%s' is not in HH:mm 24-hour format", receipt.PurchaseTime)
		return errors.New("purchaseTime must be in HH:mm 24-hour format")
	}

	// Validate Items
	if len(receipt.Items) < 1 {
		log.Println("Validation failed: Items array is empty")
		return errors.New("items array must have at least one item")
	}
	for index, item := range receipt.Items {
		// Validate ShortDescription
		if item.ShortDescription == "" {
			log.Printf("Validation failed: Item at index %d has an empty shortDescription", index)
			return errors.New("item shortDescription is invalid")
		}
		if !regexp.MustCompile(`^[\w\s\-]+$`).MatchString(item.ShortDescription) {
			log.Printf("Validation failed: Item at index %d has invalid characters in shortDescription '%s'", index, item.ShortDescription)
			return errors.New("item shortDescription is invalid")
		}

		// Validate Price
		if !regexp.MustCompile(`^\d+\.\d{2}$`).MatchString(item.Price) {
			log.Printf("Validation failed: Item at index %d has an invalid price '%s'", index, item.Price)
			return errors.New("item price must be a valid decimal number")
		}
	}

	// Validate Total
	if !regexp.MustCompile(`^\d+\.\d{2}$`).MatchString(receipt.Total) {
		log.Printf("Validation failed: Total '%s' is not a valid decimal number", receipt.Total)
		return errors.New("total must be a valid decimal number")
	}

	// Log success if all validations pass
	log.Println("Validation successful for receipt")
	return nil
}

func countAlphanumeric(s string) int {
	count := 0
	for _, char := range s {
		// Check if the character is alphanumeric
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			count++
		}
	}
	return count
}

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

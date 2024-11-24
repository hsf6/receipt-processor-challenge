package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	log.Println("Client started. Processing receipt...")

	// Read the JSON payload from a file
	payloadFile := "payload.json"
	log.Printf("Reading JSON payload from file: %s", payloadFile)
	payloadBytes, err := ioutil.ReadFile(payloadFile)
	if err != nil {
		log.Fatalf("Error reading payload file '%s': %v", payloadFile, err)
	}

	// Process a Receipt (POST request)
	postURL := "http://localhost:8080/receipts/process"
	log.Printf("Sending POST request to: %s", postURL)
	resp, err := http.Post(postURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Fatalf("Error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	// Handle Non-200 Response Status
	if resp.StatusCode != http.StatusOK {
		log.Printf("POST request failed with status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Server response: %s", string(body))
		fmt.Printf("\nError: %s\n", string(body))
		return
	}

	log.Printf("POST request sent successfully. Reading response...")
	// Parse the POST response
	var postResponse map[string]string
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading POST response: %v", err)
	}
	if err := json.Unmarshal(body, &postResponse); err != nil {
		log.Fatalf("Error parsing POST response: %v", err)
	}

	// Retrieve the receipt ID
	receiptID, exists := postResponse["id"]
	if !exists {
		log.Fatalf("POST response does not contain 'id'")
	}
	log.Printf("Receipt processed successfully. Received ID: %s\n", receiptID)
	fmt.Printf("\nReceipt Processed. ID: %s\n\n", receiptID)

	// Get Breakdown (GET request)
	breakdownURL := fmt.Sprintf("http://localhost:8080/receipts/%s/breakdown", receiptID)
	log.Printf("Sending GET request to: %s", breakdownURL)
	getResp, err := http.Get(breakdownURL)
	if err != nil {
		log.Fatalf("Error sending GET request for breakdown: %v", err)
	}
	defer getResp.Body.Close()

	// Handle Non-200 Response Status for Breakdown
	if getResp.StatusCode != http.StatusOK {
		log.Printf("GET request failed with status: %d %s", getResp.StatusCode, http.StatusText(getResp.StatusCode))
		breakdownBody, _ := ioutil.ReadAll(getResp.Body)
		log.Printf("Server response: %s", string(breakdownBody))
		fmt.Printf("\nError: %s\n", string(breakdownBody))
		return
	}

	log.Printf("GET request sent successfully. Reading response...")
	// Parse the GET response for breakdown
	var breakdownResponse map[string]interface{}
	breakdownBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		log.Fatalf("Error reading GET response for breakdown: %v", err)
	}
	if err := json.Unmarshal(breakdownBody, &breakdownResponse); err != nil {
		log.Fatalf("Error parsing GET response for breakdown: %v", err)
	}

	// Output Total Points
	totalPoints := breakdownResponse["points"]
	log.Printf("Total Points received: %v", totalPoints)
	fmt.Printf("Total Points: %v\n\n", totalPoints)

	// Output Breakdown
	log.Println("Printing breakdown of points...")
	fmt.Printf("Breakdown of Points:\n")
	for _, line := range breakdownResponse["breakdown"].([]interface{}) {
		fmt.Printf("- %s\n", line)
	}
	log.Println("Client processing completed successfully.")
}

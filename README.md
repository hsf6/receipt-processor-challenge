
# Receipt Processor

Receipt Processor is a Go-based web service that calculates points from receipts using predefined rules. It provides a RESTful API for submitting receipts, retrieving points earned, and viewing a detailed breakdown of the points calculation.

## Features
- **Submit Receipts**: Parse receipt data and calculate points.
- **Retrieve Points**: Get the total points earned for a submitted receipt.
- **Breakdown of Points**: View the detailed breakdown of how points are calculated.

## Installation
1. Clone the repository:  
   ```bash
   git clone https://github.com/hsf6/receipt-processor-challenge.git
   cd receipt-processor-challenge
   ```
2. Install dependencies:  
   Ensure Go is installed (`v1.23+` recommended).  

3. Add Payload
   Ensure payload.json is present in the project directory.

4. Run the service:  
   ```bash
   go run main.go
   ```
5. Run the client:  
   ```bash
   go run client.go
   ```

## API Endpoints

- **POST /receipts/process**  
  Submit a receipt and calculate points.  
  - Request:  
    ```json

    {
      "retailer": "Target",
      "purchaseDate": "2022-01-01",
      "purchaseTime": "13:01",
        "items": [
          {
            "shortDescription": "Mountain Dew 12PK",
            "price": "6.49"
          },{
            "shortDescription": "Emils Cheese Pizza",
            "price": "12.25"
          },{
            "shortDescription": "Knorr Creamy Soup",
            "price": "1.26"
          },{
            "shortDescription": "Doritos Nacho Cheese",
            "price": "3.35"
          },{
            "shortDescription": "   Klarbrunn 12-PK 12 FL OZ  ",
            "price": "12.00"
          }
        ],
        "total": "35.35"
    }

    ```
  - Response:  
    ```json
    { "id": "cb445f45-21e3-48b6-acd9-3150c9ed429c" }
    ```

- **GET /receipts/{id}/points**  
  Retrieve the total points for a submitted receipt.  
  - Response:  
    ```json
    { "points": 28 }
    ```

- **GET /receipts/{id}/breakdown**  (Additional Endpoint)
  Retrieve the breakdown of points earned for a receipt, showing how points are calculated.  
  - Response:  
    ```json
    {
      "breakdown": [
        "6 points - retailer name (Target) has 6 alphanumeric characters",
        "10 points - 5 items (2 pairs @ 5 points each)",
        "3 points - \"Emils Cheese Pizza\" is 18 characters (a multiple of 3), item price 12.25 * 0.2 = 2.45 which is rounded to: 3 points",
        "3 points - \"Klarbrunn 12-PK 12 FL OZ\" is 24 characters (a multiple of 3), item price 12.00 * 0.2 = 2.40 which is rounded to: 3 points",
        "6 points - purchase day is odd"
      ],
      "points": 28
    }
    ```
## Documentation
A detailed documentation for this project is available in the receipt-challenge.pdf file, which provides further insights into the implementation and steps to run the Receipt Processor service.

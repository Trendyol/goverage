package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/Trendyol/goverage" // Import the coverage server
)

// Simple web server for demonstration
func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/calculate", calculateHandler)

	log.Println("Starting example server on :8080")
	log.Println("Coverage server will automatically start on :7777")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Example Application - Coverage Testing\n")
	fmt.Fprintf(w, "Try these endpoints:\n")
	fmt.Fprintf(w, "  GET /api/health\n")
	fmt.Fprintf(w, "  GET /api/calculate?a=5&b=3\n\n")
	fmt.Fprintf(w, "Get coverage: POST http://localhost:7777/v1/cover/profile\n")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"%s","timestamp":"%s","uptime":"%s"}`,
		response["status"], response["timestamp"], response["uptime"])
}

func calculateHandler(w http.ResponseWriter, r *http.Request) {
	aStr := r.URL.Query().Get("a")
	bStr := r.URL.Query().Get("b")

	if aStr == "" || bStr == "" {
		http.Error(w, "Missing parameters a and b", http.StatusBadRequest)
		return
	}

	// Simple calculation for coverage demonstration
	var a, b int
	fmt.Sscanf(aStr, "%d", &a)
	fmt.Sscanf(bStr, "%d", &b)

	result := calculate(a, b)
	fmt.Fprintf(w, `{"a":%d,"b":%d,"sum":%d,"product":%d}`, a, b, result.sum, result.product)
}

type CalculationResult struct {
	sum     int
	product int
}

func calculate(a, b int) CalculationResult {
	result := CalculationResult{}

	// Add some branching for coverage demonstration
	if a > 0 && b > 0 {
		result.sum = a + b
		result.product = a * b
	} else if a == 0 || b == 0 {
		result.sum = a + b
		result.product = 0
	} else {
		// Negative numbers
		result.sum = a + b
		if a < 0 && b < 0 {
			result.product = a * b // This will be positive
		} else {
			result.product = a * b // This will be negative
		}
	}

	return result
}

var startTime = time.Now()

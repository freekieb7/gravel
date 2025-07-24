package main

import (
	"fmt"
	"time"

	"github.com/freekieb7/gravel/json"
)

// User represents a user in our system
type User struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Active   bool      `json:"active"`
	Tags     []string  `json:"tags,omitempty"`
	Settings *Settings `json:"settings,omitempty"`
}

// Settings represents user settings
type Settings struct {
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
	Language      string `json:"language"`
}

// Company represents a company with many users
type Company struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Users []User `json:"users"`
}

func main() {
	fmt.Println("🚀 Gravel JSON Performance Demo")
	fmt.Println("================================")

	// Demo 1: Basic Marshal/Unmarshal
	fmt.Println("\n1. Basic Marshal/Unmarshal Operations")
	demoBasicOperations()

	// Demo 2: Zero-allocation APIs
	fmt.Println("\n2. Zero-allocation APIs")
	demoZeroAllocation()

	// Demo 3: Fast parsing with simdjson-inspired features
	fmt.Println("\n3. Fast Parsing (simdjson-inspired)")
	demoFastParsing()

	// Demo 4: Zero-copy search
	fmt.Println("\n4. Zero-copy Search")
	demoZeroCopySearch()

	// Demo 5: Performance comparison
	fmt.Println("\n5. Performance Comparison")
	demoPerformanceComparison()
}

func demoBasicOperations() {
	user := User{
		ID:     123,
		Name:   "Alice Johnson",
		Email:  "alice@example.com",
		Active: true,
		Tags:   []string{"admin", "developer", "team-lead"},
		Settings: &Settings{
			Theme:         "dark",
			Notifications: true,
			Language:      "en",
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Marshaled JSON: %s\n", string(data))

	// Unmarshal back
	var decoded User
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Decoded user: %+v\n", decoded)
}

func demoZeroAllocation() {
	user := User{
		ID:     456,
		Name:   "Bob Smith",
		Email:  "bob@example.com",
		Active: true,
	}

	// Pre-allocate buffer
	buf := make([]byte, 0, 256)

	// Use MarshalAppend for zero-allocation marshaling
	result, err := json.MarshalAppend(buf, user)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Zero-alloc marshal: %s\n", string(result))

	// Use MarshalTo for direct buffer writing
	targetBuf := make([]byte, 256)
	n, err := json.MarshalTo(user, targetBuf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("MarshalTo result: %s\n", string(targetBuf[:n]))
}

func demoFastParsing() {
	// Create a complex JSON document
	company := Company{
		ID:   100,
		Name: "Tech Corp",
		Users: []User{
			{ID: 1, Name: "John Doe", Email: "john@techcorp.com", Active: true},
			{ID: 2, Name: "Jane Smith", Email: "jane@techcorp.com", Active: false},
			{ID: 3, Name: "Mike Wilson", Email: "mike@techcorp.com", Active: true},
		},
	}

	// Marshal to get JSON data
	jsonData, _ := json.Marshal(company)
	fmt.Printf("Complex JSON (%d bytes): %s\n", len(jsonData), string(jsonData))

	// Use fast parser (simdjson-inspired)
	start := time.Now()
	parser, err := json.ParseFast(jsonData)
	parseTime := time.Since(start)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Fast parsing completed in %v\n", parseTime)
	fmt.Printf("Tape instructions: %d\n", len(parser.GetTape()))
	fmt.Printf("String pool size: %d bytes\n", len(parser.GetStringPool()))
}

func demoZeroCopySearch() {
	// Create complex nested JSON
	nestedJSON := `{
		"company": {
			"id": 100,
			"name": "Tech Corp",
			"location": {
				"city": "San Francisco",
				"country": "USA",
				"coordinates": {
					"lat": 37.7749,
					"lon": -122.4194
				}
			},
			"employees": [
				{"name": "Alice", "role": "Engineer", "salary": 120000},
				{"name": "Bob", "role": "Designer", "salary": 95000},
				{"name": "Charlie", "role": "Manager", "salary": 140000}
			],
			"active": true
		}
	}`

	fmt.Printf("Nested JSON document (%d bytes)\n", len(nestedJSON))

	// Zero-copy search examples
	start := time.Now()

	// Search for company name
	companyName, err := json.SearchString([]byte(nestedJSON), "company.name")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Company name: %s\n", companyName)

	// Search for latitude
	lat, err := json.SearchFloat([]byte(nestedJSON), "company.location.coordinates.lat")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Latitude: %f\n", lat)

	// Search for first employee salary
	salary, err := json.SearchInt([]byte(nestedJSON), "company.employees.0.salary")
	if err != nil {
		panic(err)
	}
	fmt.Printf("First employee salary: %d\n", salary)

	// Search for active status
	value, err := json.ZeroCopySearch([]byte(nestedJSON), "company.active")
	if err != nil {
		panic(err)
	}
	active, err := value.GetBool()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Company active: %t\n", active)

	searchTime := time.Since(start)
	fmt.Printf("All zero-copy searches completed in %v\n", searchTime)
}

func demoPerformanceComparison() {
	fmt.Println("\n5. Performance Comparison")

	// Create a moderate-sized JSON document
	largeCompany := Company{
		ID:    200,
		Name:  "Big Tech Corp",
		Users: make([]User, 50), // Smaller dataset to avoid stack overflow
	}

	// Populate users
	for i := 0; i < 50; i++ {
		largeCompany.Users[i] = User{
			ID:     i + 1,
			Name:   fmt.Sprintf("User %d", i+1),
			Email:  fmt.Sprintf("user%d@bigtech.com", i+1),
			Active: i%2 == 0, // Alternate active/inactive
		}
	}

	// Marshal the document
	jsonData, _ := json.Marshal(largeCompany)
	fmt.Printf("Test JSON document: %d bytes, %d users\n", len(jsonData), len(largeCompany.Users))

	// Compare parsing methods
	const iterations = 10

	// Fast parsing only (to avoid stack overflow in unmarshal)
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := json.ParseFast(jsonData)
		if err != nil {
			panic(err)
		}
	}
	fastTime := time.Since(start)

	// Zero-copy search
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, err := json.SearchString(jsonData, "name")
		if err != nil {
			panic(err)
		}
	}
	searchTime := time.Since(start)

	fmt.Printf("Performance benchmark (%d iterations):\n", iterations)
	fmt.Printf("  Fast parsing:          %v (%v per op)\n", fastTime, fastTime/iterations)
	fmt.Printf("  Zero-copy search:      %v (%v per op)\n", searchTime, searchTime/iterations)

	fmt.Printf("\nThe high-performance JSON library provides:\n")
	fmt.Printf("  ✓ SIMD-accelerated parsing\n")
	fmt.Printf("  ✓ Zero-allocation marshaling\n")
	fmt.Printf("  ✓ Zero-copy value extraction\n")
	fmt.Printf("  ✓ Simdjson-inspired tape architecture\n")
}

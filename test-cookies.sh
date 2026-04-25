#!/bin/bash

# Test script to verify cookie loading

# Simulate the workflow's cookie setup
CHATURBATE_COOKIES="cf_clearance=test123; __cf_bm=test456; csrftoken=test789"

mkdir -p ./conf

cat > ./conf/settings.json << EOF
{
  "cookies": "${CHATURBATE_COOKIES}",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
}
EOF

echo "Created settings.json:"
cat ./conf/settings.json

echo ""
echo "Testing if Go can read it:"
cat > test.go << 'GOEOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type settings struct {
	Cookies   string `json:"cookies"`
	UserAgent string `json:"user_agent"`
}

func main() {
	data, err := os.ReadFile("./conf/settings.json")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	
	var s settings
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}
	
	fmt.Printf("Cookies loaded: %s\n", s.Cookies[:50])
	fmt.Printf("User-Agent loaded: %s\n", s.UserAgent[:50])
}
GOEOF

go run test.go
rm test.go

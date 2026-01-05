#!/bin/bash

# Script de pruebas para ROJUDGER API
# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URL de la API
BASE_URL="${API_URL:-http://localhost:8080}"

echo "🧪 Testing ROJUDGER API"
echo "Base URL: $BASE_URL"
echo ""

# Función para imprimir resultados
print_test() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        echo -e "${RED}  Error: $3${NC}"
    fi
}

# Test 1: Health Check
echo "Test 1: Health Check"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "200" ]; then
    print_test 0 "Health check passed"
    echo "  Response: $body"
else
    print_test 1 "Health check failed" "HTTP $http_code"
fi
echo ""

# Test 2: Get Languages
echo "Test 2: Get Available Languages"
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/languages")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "200" ]; then
    print_test 0 "Languages endpoint working"
    echo "  Languages available: $(echo "$body" | grep -o '"id"' | wc -l)"
else
    print_test 1 "Languages endpoint failed" "HTTP $http_code"
fi
echo ""

# Test 3: Simple Python Hello World (Synchronous)
echo "Test 3: Python Hello World (Synchronous)"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Hello, ROJUDGER!\")",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    status=$(echo "$body" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    stdout=$(echo "$body" | grep -o '"stdout":"[^"]*"' | cut -d'"' -f4)

    if [ "$status" == "completed" ] && [[ "$stdout" == *"Hello, ROJUDGER!"* ]]; then
        print_test 0 "Python Hello World executed successfully"
        echo "  Output: $stdout"
    else
        print_test 1 "Python execution failed" "Status: $status, Output: $stdout"
    fi
else
    print_test 1 "Python submission failed" "HTTP $http_code"
fi
echo ""

# Test 4: Python with Input
echo "Test 4: Python with stdin"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "name = input()\nprint(f\"Hello, {name}!\")",
    "stdin": "Alice"
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    stdout=$(echo "$body" | grep -o '"stdout":"[^"]*"' | cut -d'"' -f4)

    if [[ "$stdout" == *"Hello, Alice!"* ]]; then
        print_test 0 "Python with stdin works correctly"
        echo "  Output: $stdout"
    else
        print_test 1 "Python stdin test failed" "Output: $stdout"
    fi
else
    print_test 1 "Python stdin submission failed" "HTTP $http_code"
fi
echo ""

# Test 5: JavaScript Hello World
echo "Test 5: JavaScript Hello World"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 63,
    "source_code": "console.log(\"Hello from Node.js!\")",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    status=$(echo "$body" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    stdout=$(echo "$body" | grep -o '"stdout":"[^"]*"' | cut -d'"' -f4)

    if [ "$status" == "completed" ] && [[ "$stdout" == *"Hello from Node.js!"* ]]; then
        print_test 0 "JavaScript executed successfully"
        echo "  Output: $stdout"
    else
        print_test 1 "JavaScript execution failed" "Status: $status"
    fi
else
    print_test 1 "JavaScript submission failed" "HTTP $http_code"
fi
echo ""

# Test 6: Compilation Error (C++)
echo "Test 6: Compilation Error Handling (C++)"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 54,
    "source_code": "#include <iostream>\nint main() {\n  std::cout << \"Missing semicolon\"\n  return 0;\n}",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    exit_code=$(echo "$body" | grep -o '"exit_code":[0-9]*' | cut -d':' -f2)

    if [ "$exit_code" != "0" ]; then
        print_test 0 "Compilation error detected correctly"
        echo "  Exit code: $exit_code"
    else
        print_test 1 "Compilation error not detected" "Exit code was 0"
    fi
else
    print_test 1 "C++ submission failed" "HTTP $http_code"
fi
echo ""

# Test 7: Runtime Error
echo "Test 7: Runtime Error Handling (Python)"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "x = 1 / 0",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    exit_code=$(echo "$body" | grep -o '"exit_code":[0-9]*' | cut -d':' -f2)
    stderr=$(echo "$body" | grep -o '"stderr":"[^"]*"' | cut -d'"' -f4)

    if [ "$exit_code" != "0" ] && [[ "$stderr" == *"ZeroDivisionError"* ]]; then
        print_test 0 "Runtime error captured correctly"
        echo "  Exit code: $exit_code"
    else
        print_test 1 "Runtime error not handled properly" "Exit: $exit_code"
    fi
else
    print_test 1 "Python error submission failed" "HTTP $http_code"
fi
echo ""

# Test 8: Asynchronous Submission
echo "Test 8: Asynchronous Submission"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 71,
    "source_code": "print(\"Async test\")",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

if [ "$http_code" == "201" ]; then
    submission_id=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    if [ ! -z "$submission_id" ]; then
        print_test 0 "Async submission created"
        echo "  Submission ID: $submission_id"

        # Test 9: Get Submission Result
        echo ""
        echo "Test 9: Get Submission by ID"
        sleep 2 # Wait for execution

        response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/submissions/$submission_id")
        http_code=$(echo "$response" | tail -n1)
        body=$(echo "$response" | head -n-1)

        if [ "$http_code" == "200" ]; then
            status=$(echo "$body" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
            print_test 0 "Submission retrieved successfully"
            echo "  Status: $status"
        else
            print_test 1 "Failed to retrieve submission" "HTTP $http_code"
        fi
    else
        print_test 1 "No submission ID returned" ""
    fi
else
    print_test 1 "Async submission failed" "HTTP $http_code"
fi
echo ""

# Test 10: Invalid Language ID
echo "Test 10: Invalid Language ID"
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/submissions?wait=true" \
  -H "Content-Type: application/json" \
  -d '{
    "language_id": 9999,
    "source_code": "print(\"test\")",
    "stdin": ""
  }')
http_code=$(echo "$response" | tail -n1)

if [ "$http_code" == "400" ]; then
    print_test 0 "Invalid language ID rejected correctly"
else
    print_test 1 "Invalid language ID should return 400" "Got HTTP $http_code"
fi
echo ""

# Summary
echo "================================"
echo "🎉 Test suite completed!"
echo "================================"
echo ""
echo "To run individual tests, you can use curl directly:"
echo ""
echo "Example 1 - Simple Python:"
echo "curl -X POST \"$BASE_URL/api/v1/submissions?wait=true\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"language_id\": 71, \"source_code\": \"print(\\\"Hello!\\\")\"}'"
echo ""
echo "Example 2 - Get all languages:"
echo "curl \"$BASE_URL/api/v1/languages\""
echo ""

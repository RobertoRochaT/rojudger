#!/bin/bash

set -e

echo "=========================================="
echo "  ROJUDGER - All Languages Test Suite"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080/api/v1/submissions?wait=true"

# Function to test a submission
test_submission() {
    local name="$1"
    local language_id="$2"
    local source_code="$3"
    local stdin="$4"
    local expected_status="$5"
    local check_stdout="$6"

    echo -e "${BLUE}Testing: ${name}${NC}"

    # Create JSON payload
    json_payload=$(cat <<EOF
{
    "language_id": ${language_id},
    "source_code": ${source_code},
    "stdin": "${stdin}"
}
EOF
)

    # Make request
    response=$(curl -s -X POST "${API_URL}" \
        -H "Content-Type: application/json" \
        -d "${json_payload}")

    # Extract fields
    status=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', 'unknown'))")
    exit_code=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('exit_code', -1))")
    stdout=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stdout', ''))")
    stderr=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('stderr', ''))")
    compile_output=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('compile_output', ''))")

    # Check status
    if [ "$status" = "$expected_status" ]; then
        if [ -n "$check_stdout" ] && [[ "$stdout" == *"$check_stdout"* ]]; then
            echo -e "${GREEN}✓ PASSED${NC} - Status: $status, Output: OK"
            return 0
        elif [ -z "$check_stdout" ]; then
            echo -e "${GREEN}✓ PASSED${NC} - Status: $status"
            return 0
        else
            echo -e "${YELLOW}⚠ PARTIAL${NC} - Status OK but output mismatch"
            echo "  Expected in output: '$check_stdout'"
            echo "  Got: '$stdout'"
            return 1
        fi
    else
        echo -e "${RED}✗ FAILED${NC} - Expected status: $expected_status, Got: $status"
        echo "  Exit Code: $exit_code"
        if [ -n "$stdout" ]; then
            echo "  Stdout: $stdout"
        fi
        if [ -n "$stderr" ]; then
            echo "  Stderr: $stderr"
        fi
        if [ -n "$compile_output" ]; then
            echo "  Compile Output: $compile_output"
        fi
        return 1
    fi
}

# Test counters
TOTAL=0
PASSED=0
FAILED=0

echo -e "${BLUE}Checking API health...${NC}"
health=$(curl -s http://localhost:8080/health)
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ API is running${NC}"
    echo ""
else
    echo -e "${RED}✗ API is not responding${NC}"
    echo "Please start the API with: go run cmd/api/main.go"
    exit 1
fi

echo "=========================================="
echo "  Python 3 Tests"
echo "=========================================="

# Python 3 - Hello World
TOTAL=$((TOTAL + 1))
if test_submission \
    "Python Hello World" \
    71 \
    '"print(\"Hello, World!\")"' \
    "" \
    "completed" \
    "Hello, World!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# Python 3 - With Input
TOTAL=$((TOTAL + 1))
if test_submission \
    "Python with stdin" \
    71 \
    '"name = input(\"Name: \")\nprint(f\"Hello, {name}!\")"' \
    "Alice" \
    "completed" \
    "Hello, Alice!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# Python 3 - Math
TOTAL=$((TOTAL + 1))
if test_submission \
    "Python math operations" \
    71 \
    '"import math\nprint(math.factorial(5))"' \
    "" \
    "completed" \
    "120"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
echo "  JavaScript (Node.js) Tests"
echo "=========================================="

# JavaScript - Hello World
TOTAL=$((TOTAL + 1))
if test_submission \
    "JavaScript Hello World" \
    63 \
    '"console.log(\"Hello from Node.js!\");"' \
    "" \
    "completed" \
    "Hello from Node.js!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# JavaScript - Array operations
TOTAL=$((TOTAL + 1))
if test_submission \
    "JavaScript array operations" \
    63 \
    '"const arr = [1, 2, 3, 4, 5];\nconst sum = arr.reduce((a, b) => a + b, 0);\nconsole.log(\"Sum:\", sum);"' \
    "" \
    "completed" \
    "Sum: 15"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
echo "  Go Tests"
echo "=========================================="

# Go - Hello World
TOTAL=$((TOTAL + 1))
if test_submission \
    "Go Hello World" \
    60 \
    '"package main\nimport \"fmt\"\nfunc main() {\n    fmt.Println(\"Hello from Go!\")\n}"' \
    "" \
    "completed" \
    "Hello from Go!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# Go - With Input
TOTAL=$((TOTAL + 1))
if test_submission \
    "Go with stdin" \
    60 \
    '"package main\nimport \"fmt\"\nfunc main() {\n    var x, y int\n    fmt.Scan(&x, &y)\n    fmt.Printf(\"Sum: %d\\n\", x+y)\n}"' \
    "5 3" \
    "completed" \
    "Sum: 8"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
echo "  C (GCC) Tests"
echo "=========================================="

# C - Hello World
TOTAL=$((TOTAL + 1))
if test_submission \
    "C Hello World" \
    50 \
    '"#include <stdio.h>\nint main() {\n    printf(\"Hello from C!\\\\n\");\n    return 0;\n}"' \
    "" \
    "completed" \
    "Hello from C!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# C - Factorial
TOTAL=$((TOTAL + 1))
if test_submission \
    "C factorial" \
    50 \
    '"#include <stdio.h>\nint factorial(int n) {\n    if (n <= 1) return 1;\n    return n * factorial(n - 1);\n}\nint main() {\n    int n;\n    scanf(\"%d\", &n);\n    printf(\"Factorial: %d\\\\n\", factorial(n));\n    return 0;\n}"' \
    "6" \
    "completed" \
    "Factorial: 720"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
echo "  C++ (G++) Tests"
echo "=========================================="

# C++ - Hello World
TOTAL=$((TOTAL + 1))
if test_submission \
    "C++ Hello World" \
    54 \
    '"#include <iostream>\nint main() {\n    std::cout << \"Hello from C++!\" << std::endl;\n    return 0;\n}"' \
    "" \
    "completed" \
    "Hello from C++!"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# C++ - Fibonacci
TOTAL=$((TOTAL + 1))
if test_submission \
    "C++ fibonacci" \
    54 \
    '"#include <iostream>\nint fib(int n) {\n    if (n <= 1) return n;\n    return fib(n - 1) + fib(n - 2);\n}\nint main() {\n    int n;\n    std::cin >> n;\n    std::cout << \"Fib(\" << n << \") = \" << fib(n) << std::endl;\n    return 0;\n}"' \
    "7" \
    "completed" \
    "Fib(7) = 13"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# C++ - Vector operations
TOTAL=$((TOTAL + 1))
if test_submission \
    "C++ vector operations" \
    54 \
    '"#include <iostream>\n#include <vector>\nint main() {\n    std::vector<int> v = {1, 2, 3, 4, 5};\n    int sum = 0;\n    for (int x : v) sum += x;\n    std::cout << \"Sum: \" << sum << std::endl;\n    return 0;\n}"' \
    "" \
    "completed" \
    "Sum: 15"; then
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo "=========================================="
echo "  Error Handling Tests"
echo "=========================================="

# C++ - Compilation Error
TOTAL=$((TOTAL + 1))
if test_submission \
    "C++ compilation error" \
    54 \
    '"#include <iostream>\nint main() {\n    std::cout << \"Missing semicolon\"\n    return 0;\n}"' \
    "" \
    "completed" \
    ""; then
    # Should complete but with exit code 1
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

# Python - Runtime Error
TOTAL=$((TOTAL + 1))
if test_submission \
    "Python runtime error" \
    71 \
    '"x = 10 / 0"' \
    "" \
    "completed" \
    ""; then
    # Should complete but with exit code 1
    PASSED=$((PASSED + 1))
else
    FAILED=$((FAILED + 1))
fi
echo ""

echo ""
echo "=========================================="
echo "  Test Results Summary"
echo "=========================================="
echo ""
echo -e "Total Tests:  ${BLUE}${TOTAL}${NC}"
echo -e "Passed:       ${GREEN}${PASSED}${NC}"
echo -e "Failed:       ${RED}${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  Some tests failed${NC}"
    exit 1
fi

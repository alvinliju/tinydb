#!/bin/bash

# Storage Load Test Script
# Usage: ./test.sh

MASTER_URL="http://localhost:3000"
TEST_FILE="test_1GB.bin"
RESULTS_DIR="test_results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}=== TinyDB Storage Load Test ===${NC}"

# Create results directory
mkdir -p $RESULTS_DIR

# Generate 500MB test file
echo -e "${YELLOW}Generating 500MB test file...${NC}"
if [ ! -f $TEST_FILE ]; then
    dd if=/dev/urandom of=$TEST_FILE bs=1M count=1000 2>/dev/null
    echo -e "${GREEN}✓ Created $TEST_FILE${NC}"
else
    echo -e "${GREEN}✓ Using existing $TEST_FILE${NC}"
fi

# Test 1: Single large file upload
echo -e "\n${YELLOW}=== Test 1: Single 500MB Upload ===${NC}"
time curl -X PUT \
    -H "Content-Type: application/octet-stream" \
    --data-binary @$TEST_FILE \
    "$MASTER_URL/large_test_file" \
    -w "Status: %{http_code}\nTime: %{time_total}s\nSpeed: %{speed_upload} bytes/sec\n"

# Test 2: Concurrent small uploads (1MB each)
echo -e "\n${YELLOW}=== Test 2: Concurrent Small Uploads ===${NC}"
echo "Creating 1MB test files..."
for i in {1..10}; do
    dd if=/dev/urandom of="small_${i}.bin" bs=1M count=1 2>/dev/null
done

echo "Running concurrent uploads..."
hey -n 10 -c 5 -m PUT \
    -H "Content-Type: application/octet-stream" \
    -D small_1.bin \
    "$MASTER_URL/concurrent_test" > $RESULTS_DIR/concurrent_upload.txt

echo -e "${GREEN}Results saved to $RESULTS_DIR/concurrent_upload.txt${NC}"
cat $RESULTS_DIR/concurrent_upload.txt

# Test 3: Read performance
echo -e "\n${YELLOW}=== Test 3: Read Performance ===${NC}"
echo "Testing read speed..."
hey -n 20 -c 5 \
    "$MASTER_URL/large_test_file" > $RESULTS_DIR/read_test.txt

echo -e "${GREEN}Results saved to $RESULTS_DIR/read_test.txt${NC}"
cat $RESULTS_DIR/read_test.txt

# Test 4: Mixed workload
echo -e "\n${YELLOW}=== Test 4: Mixed Read/Write Workload ===${NC}"
# Background reads
hey -n 50 -c 2 "$MASTER_URL/large_test_file" > $RESULTS_DIR/mixed_reads.txt &
READ_PID=$!

# Concurrent writes
for i in {1..5}; do
    curl -X PUT \
        -H "Content-Type: application/octet-stream" \
        --data-binary @small_${i}.bin \
        "$MASTER_URL/mixed_test_${i}" &
done

wait $READ_PID
wait

echo -e "${GREEN}Mixed workload complete${NC}"

# Test 5: Stress test with small files
echo -e "\n${YELLOW}=== Test 5: Small File Stress Test ===${NC}"
echo "Creating 100KB test file..."
dd if=/dev/urandom of=tiny.bin bs=1K count=100 2>/dev/null

echo "Running stress test..."
hey -n 100 -c 10 -m PUT \
    -H "Content-Type: application/octet-stream" \
    -D tiny.bin \
    "$MASTER_URL/stress_test" > $RESULTS_DIR/stress_test.txt

echo -e "${GREEN}Results saved to $RESULTS_DIR/stress_test.txt${NC}"
cat $RESULTS_DIR/stress_test.txt

# Cleanup
echo -e "\n${YELLOW}Cleaning up test files...${NC}"
rm -f small_*.bin tiny.bin
# Keep the 500MB file for future tests
# rm -f $TEST_FILE

echo -e "\n${GREEN}=== Test Complete ===${NC}"
echo -e "Results saved in: $RESULTS_DIR/"
echo -e "Test file kept: $TEST_FILE"

# Summary
echo -e "\n${YELLOW}=== Quick Analysis ===${NC}"
echo "Check these key metrics:"
echo "1. Upload speed (bytes/sec) for large files"
echo "2. Requests/sec for small files"
echo "3. Error rate under concurrent load"
echo "4. Memory/CPU usage during tests"

echo -e "\n${YELLOW}Monitor your servers with:${NC}"
echo "top -p \$(pgrep -f 'your_binary')"
echo "netstat -an | grep :300[0-9]"

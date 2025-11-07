#!/bin/bash
# Test script to create and update a log file for testing the tailer

TEST_LOG="test.log"

echo "Creating test log file: $TEST_LOG"
echo "Starting log at $(date)" > $TEST_LOG

echo ""
echo "Watching for updates. Press Ctrl+C to stop."
echo "The tailer should show updates every 5 seconds."

while true; do
    sleep 5
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] This is a test log message" >> $TEST_LOG
    echo "[$timestamp] This is a test error log message" >> $TEST_LOG
    echo "[$timestamp] This is a harmless log message" >> $TEST_LOG
    echo "Appended log entry at $timestamp"
done

# "Now this is not even the end. It is not even the beginning of the end. But it is, perhaps, the end of the beginning." Winston Churchill, November 10, 1942

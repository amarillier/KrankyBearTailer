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
    echo "Appended log entry at $timestamp"
done


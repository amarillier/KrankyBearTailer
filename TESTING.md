# Testing Guide

## Testing the Tailing Functionality

### Step 1: Create a test log file
```bash
echo "Starting log at $(date)" > test.log
```

### Step 2: Run the application
```bash
go run main.go
```

### Step 3: Open the test file
- Click the "Open File" button in the toolbar, OR
- Use menu: File → Open File, OR
- Use keyboard shortcut: Cmd+O (Mac) / Ctrl+O (Windows/Linux)

### Step 4: Add test data to the log file
In another terminal, run:
```bash
# Add entries every few seconds
for i in {1..10}; do
  echo "[$i] Test log entry at $(date)" >> test.log
  sleep 2
done
```

### Step 5: Verify it's working
- You should see new log entries appearing in the app every 2 seconds
- Add keywords to highlight specific lines
- Use the "View" button to see all watched keywords

## Testing Preferences/Session Restore

1. Open one or more log files
2. Add some keywords to each file
3. Close the application
4. Reopen the application
5. Your files and keywords should be automatically restored!

## Troubleshooting

### Tail not updating?
- Make sure you're adding NEW lines to the file (append with `>>`)
- Check that the file is growing in size
- Wait a few seconds - the check interval is 500ms
- Try closing and reopening the file tab

### Keywords not highlighting?
- Check the "View Keywords" window to see if your keyword was added
- Make sure the keyword text appears in the log lines
- Keywords are case-insensitive

### Tab close button not working?
- Make sure you're using the new build (with DocTabs)
- Only file tabs have close buttons (Welcome tab doesn't)
- Closing should stop tailing and clean up resources


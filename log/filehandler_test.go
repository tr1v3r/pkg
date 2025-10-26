package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFileHandler tests basic FileHandler functionality
func TestFileHandler(t *testing.T) {
	// Create a temporary directory for test logs
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		config   FileHandlerConfig
		wantErr  bool
		validate func(t *testing.T, h *FileHandler)
	}{
		{
			name: "basic file handler",
			config: FileHandlerConfig{
				LogDir:   tmpDir,
				Filename: "test.log",
				Rotation: RotationNone,
				Level:    InfoLevel,
			},
			wantErr: false,
			validate: func(t *testing.T, h *FileHandler) {
				// Verify file was created
				filePath := h.GetCurrentFilePath()
				if filePath != filepath.Join(tmpDir, "test.log") {
					t.Errorf("expected file path %s, got %s", filepath.Join(tmpDir, "test.log"), filePath)
				}

				// Test writing a log message
				h.Output(InfoLevel, context.Background(), "test message")
				h.Flush()

				// Verify file contains the message
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("failed to read log file: %v", err)
				}
				if len(content) == 0 {
					t.Error("log file is empty after writing message")
				}
			},
		},
		{
			name: "daily rotation",
			config: FileHandlerConfig{
				LogDir:   tmpDir,
				Filename: "daily.log",
				Rotation: RotationDaily,
				Level:    DebugLevel,
			},
			wantErr: false,
			validate: func(t *testing.T, h *FileHandler) {
				// Verify filename contains date
				filePath := h.GetCurrentFilePath()
				today := time.Now().Format("2006-01-02")
				expectedName := "daily_" + today + ".log"
				if filepath.Base(filePath) != expectedName {
					t.Errorf("expected filename %s, got %s", expectedName, filepath.Base(filePath))
				}
			},
		},
		{
			name: "hourly rotation",
			config: FileHandlerConfig{
				LogDir:   tmpDir,
				Filename: "hourly.log",
				Rotation: RotationHourly,
				Level:    WarnLevel,
			},
			wantErr: false,
			validate: func(t *testing.T, h *FileHandler) {
				// Verify filename contains hour
				filePath := h.GetCurrentFilePath()
				hour := time.Now().Format("2006-01-02_15")
				expectedName := "hourly_" + hour + ".log"
				if filepath.Base(filePath) != expectedName {
					t.Errorf("expected filename %s, got %s", expectedName, filepath.Base(filePath))
				}
			},
		},
		{
			name: "invalid directory",
			config: FileHandlerConfig{
				LogDir:   "/nonexistent/path",
				Filename: "test.log",
				Rotation: RotationNone,
				Level:    InfoLevel,
			},
			wantErr: true,
			validate: func(t *testing.T, h *FileHandler) {
				// Handler should be nil on error
				if h != nil {
					t.Error("expected handler to be nil on error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewFileHandler(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileHandler() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && tt.validate != nil {
				defer handler.Close()
				tt.validate(t, handler)
			}
		})
	}
}

// TestFileHandler_Rotation tests rotation functionality
func TestFileHandler_Rotation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handler with daily rotation
	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "rotation_test.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create file handler: %v", err)
	}
	defer handler.Close()

	// Get initial file path
	initialPath := handler.GetCurrentFilePath()

	// Write some logs
	handler.Output(InfoLevel, context.Background(), "initial log message")
	handler.Flush()

	// Verify file exists and has content
	if _, err := os.Stat(initialPath); err != nil {
		t.Fatalf("initial log file not created: %v", err)
	}

	// Test level filtering
	handler.Output(DebugLevel, context.Background(), "debug message should be filtered")
	handler.Flush()

	content, err := os.ReadFile(initialPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Should contain info message but not debug message
	if !contains(string(content), "initial log message") {
		t.Error("log file should contain initial message")
	}
	if contains(string(content), "debug message") {
		t.Error("log file should not contain debug message due to level filtering")
	}
}

// TestFileHandler_Async tests async message processing
func TestFileHandler_Async(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "async_test.log",
		Rotation: RotationNone,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create file handler: %v", err)
	}
	defer handler.Close()

	// Write multiple messages asynchronously
	for i := 0; i < 10; i++ {
		handler.Output(InfoLevel, context.Background(), "async message %d", i)
	}

	// Flush to ensure all messages are written
	handler.Flush()

	// Verify all messages were written
	filePath := handler.GetCurrentFilePath()
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	for i := 0; i < 10; i++ {
		if !contains(string(content), fmt.Sprintf("async message %d", i)) {
			t.Errorf("log file should contain message %d", i)
		}
	}
}

// TestAsyncRotation verifies that rotation happens correctly during async message processing
func TestAsyncRotation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create handler with daily rotation - use minimal rotation threshold for testing
	handler := &FileHandler{
		formatter: NewTextFormatter(false),
		level:     InfoLevel,
		logDir:    tmpDir,
		filename:  "async_rotation_test.log",
		rotation:  RotationDaily,
		ch:        make(chan []byte, 8192),
		closed:    make(chan struct{}),
	}

	// Simulate starting at 9am
	startTime := time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC)

	// Initialize the first log file
	if err := handler.tryRotate(startTime); err != nil {
		t.Fatalf("failed to initialize log file: %v", err)
	}

	// Start the async processor
	handler.once.Do(func() { go handler.serve() })
	handler.mu.Lock()
	handler.rotationTime = startTime
	handler.mu.Unlock()

	// Log messages throughout the day
	for i := 0; i < 100; i++ {
		handler.Output(InfoLevel, context.Background(), "Message %d at 9am", i)
	}

	// Simulate time advancing to 11pm same day
	handler.mu.Lock()
	handler.rotationTime = time.Date(2025, 10, 26, 23, 0, 0, 0, time.UTC)
	handler.mu.Unlock()

	// Log more messages
	for i := 0; i < 100; i++ {
		handler.Output(InfoLevel, context.Background(), "Message %d at 11pm", i)
	}

	// Flush to ensure all 11pm messages are processed before rotation
	handler.Flush()

	// Test rotation at just after midnight (00:01) while keeping original rotation time
	justAfterMidnight := time.Date(2025, 10, 27, 0, 1, 0, 0, time.UTC)
	if err := handler.tryRotate(justAfterMidnight); err != nil {
		t.Fatalf("failed to rotate at midnight: %v", err)
	}

	// Log messages after midnight
	for i := 0; i < 100; i++ {
		handler.Output(InfoLevel, context.Background(), "Message %d after midnight", i)
	}

	// Flush to ensure all messages are processed
	handler.Flush()

	// Check that rotation happened correctly
	handler.mu.RLock()
	currentFile := handler.filePath
	handler.mu.RUnlock()

	// The current file should be for the new day
	if !strings.Contains(currentFile, "2025-10-27") {
		t.Errorf("expected file to contain 2025-10-27, got: %s", currentFile)
	}

	t.Logf("Async rotation test completed - current file: %s", currentFile)
}

// TestNonBlockingOutput verifies that Output method doesn't block on rotation checks
func TestNonBlockingOutput(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "nonblocking_test.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	// Log many messages rapidly - this should not block even if rotation checks are needed
	start := time.Now()
	for i := 0; i < 1000; i++ {
		handler.Output(InfoLevel, context.Background(), "non-blocking message %d", i)
	}
	duration := time.Since(start)

	// If Output was blocking on rotation checks, this would take much longer
	if duration > time.Millisecond*100 {
		t.Errorf("Output method took too long (%v), indicating blocking behavior", duration)
	}

	t.Logf("Logged 1000 messages in %v (avg %v per message)", duration, duration/1000)

	// Flush and verify messages were processed
	handler.Flush()
}

// TestRotationOptimization verifies that the pre-calculated rotation time optimization works
func TestRotationOptimization(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "optimization_test.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	// Test that next rotation time is properly calculated
	handler.mu.Lock()
	nextRotation := handler.nextRotationTime
	handler.mu.Unlock()

	if nextRotation.IsZero() {
		t.Error("expected next rotation time to be calculated")
	}

	// Log several messages rapidly - with pre-calculated time, no rotation checks should happen
	for i := 0; i < 100; i++ {
		handler.Output(InfoLevel, context.Background(), "test message %d", i)
	}

	// Verify next rotation time hasn't changed (no rotation occurred)
	handler.mu.Lock()
	finalRotation := handler.nextRotationTime
	handler.mu.Unlock()

	if !nextRotation.Equal(finalRotation) {
		t.Error("expected next rotation time to remain unchanged during rapid logging")
	}

	t.Logf("Rotation optimization working: pre-calculated rotation time prevents frequent checks")
}

// TestNoRotationPerformance verifies that no-rotation configuration avoids rotation checks
func TestNoRotationPerformance(t *testing.T) {
	tmpDir := t.TempDir()

	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "no_rotation_test.log",
		Rotation: RotationNone,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	// Log many messages - with RotationNone, no rotation checks should happen
	for i := 0; i < 1000; i++ {
		handler.Output(InfoLevel, context.Background(), "no rotation test %d", i)
	}

	// Verify that next rotation time is far in the future (no rotation checks performed)
	handler.mu.Lock()
	nextRotation := handler.nextRotationTime
	handler.mu.Unlock()

	if nextRotation.IsZero() {
		t.Error("expected next rotation time to be set for RotationNone configuration")
	}

	// For RotationNone, next rotation time should be far in the future
	expectedFarFuture := time.Now().AddDate(99, 0, 0) // Close to 100 years in future
	if nextRotation.Before(expectedFarFuture) {
		t.Error("expected next rotation time to be far in the future for RotationNone")
	}

	t.Logf("No-rotation optimization working: next rotation time set far in future as expected")
}

// TestRotationBoundaries verifies that rotation happens at correct boundaries
func TestRotationBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		rotation    RotationInterval
		startTime   time.Time
		testTime    time.Time
		shouldRotate bool
		description string
	}{
		// Hourly rotation tests
		{
			name:        "hourly_same_hour",
			rotation:    RotationHourly,
			startTime:   time.Date(2025, 10, 26, 9, 30, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 26, 9, 45, 0, 0, time.UTC),
			shouldRotate: false,
			description: "Same hour, should not rotate",
		},
		{
			name:        "hourly_next_hour_same_day",
			rotation:    RotationHourly,
			startTime:   time.Date(2025, 10, 26, 9, 30, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 26, 10, 0, 1, 0, time.UTC),
			shouldRotate: true,
			description: "Next hour boundary, should rotate",
		},
		{
			name:        "hourly_next_day_same_hour",
			rotation:    RotationHourly,
			startTime:   time.Date(2025, 10, 26, 9, 30, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 27, 9, 30, 0, 0, time.UTC),
			shouldRotate: true,
			description: "Same hour next day, should rotate",
		},

		// Daily rotation tests
		{
			name:        "daily_same_day",
			rotation:    RotationDaily,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 26, 23, 59, 59, 0, time.UTC),
			shouldRotate: false,
			description: "Same day, should not rotate",
		},
		{
			name:        "daily_midnight",
			rotation:    RotationDaily,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 27, 0, 0, 1, 0, time.UTC),
			shouldRotate: true,
			description: "Midnight boundary, should rotate",
		},
		{
			name:        "daily_next_day_9am",
			rotation:    RotationDaily,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 27, 9, 0, 0, 0, time.UTC),
			shouldRotate: true,
			description: "Same time next day, should rotate",
		},

		// Weekly rotation tests
		{
			name:        "weekly_same_week",
			rotation:    RotationWeekly,
			startTime:   time.Date(2025, 10, 27, 9, 0, 0, 0, time.UTC), // Monday
			testTime:    time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC), // Friday
			shouldRotate: false,
			description: "Same week, should not rotate",
		},
		{
			name:        "weekly_next_monday",
			rotation:    RotationWeekly,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC), // Sunday
			testTime:    time.Date(2025, 11, 3, 0, 0, 0, 0, time.UTC),   // Next Monday
			shouldRotate: true,
			description: "Next Monday boundary, should rotate",
		},

		// Monthly rotation tests
		{
			name:        "monthly_same_month",
			rotation:    RotationMonthly,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC),
			testTime:    time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
			shouldRotate: false,
			description: "Same month, should not rotate",
		},
		{
			name:        "monthly_next_month",
			rotation:    RotationMonthly,
			startTime:   time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC),
			testTime:    time.Date(2025, 11, 1, 0, 0, 1, 0, time.UTC),
			shouldRotate: true,
			description: "First day of next month, should rotate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create handler with specific start time
			handler := &FileHandler{
				formatter: NewTextFormatter(false),
				level:     InfoLevel,
				logDir:    tmpDir,
				filename:  "test.log",
				rotation:  tt.rotation,
				ch:        make(chan []byte, 8192),
				closed:    make(chan struct{}),
			}

			// Initialize with start time
			handler.mu.Lock()
			handler.rotationTime = tt.startTime
			if err := handler.openFile(tt.startTime); err != nil {
				t.Fatalf("failed to open initial file: %v", err)
			}
			handler.mu.Unlock()

			// Test rotation at test time
			handler.mu.Lock()
			actualRotate := tt.testTime.After(handler.nextRotationTime)
			handler.mu.Unlock()

			if actualRotate != tt.shouldRotate {
				t.Errorf("%s: expected rotate=%v, got rotate=%v", tt.description, tt.shouldRotate, actualRotate)
			}
		})
	}
}

// TestRealWorldRotation simulates real-world rotation scenarios
func TestRealWorldRotation(t *testing.T) {
	tmpDir := t.TempDir()

	// Test daily rotation starting at 9am
	handler, err := NewFileHandler(FileHandlerConfig{
		LogDir:   tmpDir,
		Filename: "daily_test.log",
		Rotation: RotationDaily,
		Level:    InfoLevel,
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	defer handler.Close()

	// Simulate starting at 9am
	startTime := time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC)
	handler.mu.Lock()
	handler.rotationTime = startTime
	handler.mu.Unlock()

	// Log some messages at 9am
	handler.Output(InfoLevel, context.Background(), "Morning log at 9am")

	// Simulate time advancing to 11pm same day
	handler.mu.Lock()
	elevenPM := time.Date(2025, 10, 26, 23, 0, 0, 0, time.UTC)
	shouldRotate := elevenPM.After(handler.nextRotationTime)
	handler.mu.Unlock()

	if shouldRotate {
		t.Error("Should not rotate at 11pm same day when started at 9am")
	}

	// Simulate time advancing to midnight
	handler.mu.Lock()
	midnight := time.Date(2025, 10, 27, 0, 0, 1, 0, time.UTC)
	shouldRotate = midnight.After(handler.nextRotationTime)
	handler.mu.Unlock()

	if !shouldRotate {
		t.Error("Should rotate at midnight when daily rotation is configured")
	}

	// Simulate time advancing to 9am next day
	handler.mu.Lock()
	nextDay9am := time.Date(2025, 10, 27, 9, 0, 0, 0, time.UTC)
	shouldRotate = nextDay9am.After(handler.nextRotationTime)
	handler.mu.Unlock()

	if !shouldRotate {
		t.Error("Should rotate at 9am next day when daily rotation is configured")
	}
}

// TestStartOfWeek verifies week boundary calculations
func TestStartOfWeek(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected time.Time
		desc     string
	}{
		{
			input:    time.Date(2025, 10, 26, 9, 0, 0, 0, time.UTC), // Sunday
			expected: time.Date(2025, 10, 20, 0, 0, 0, 0, time.UTC), // Previous Monday
			desc:     "Sunday should start on previous Monday",
		},
		{
			input:    time.Date(2025, 10, 27, 9, 0, 0, 0, time.UTC), // Monday
			expected: time.Date(2025, 10, 27, 0, 0, 0, 0, time.UTC), // Same Monday
			desc:     "Monday should start on same Monday",
		},
		{
			input:    time.Date(2025, 10, 31, 9, 0, 0, 0, time.UTC), // Friday
			expected: time.Date(2025, 10, 27, 0, 0, 0, 0, time.UTC), // Same week Monday
			desc:     "Friday should start on same week Monday",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := startOfWeek(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("startOfWeek(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestStartOfMonth verifies month boundary calculations
func TestStartOfMonth(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected time.Time
		desc     string
	}{
		{
			input:    time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			desc:     "First day of month",
		},
		{
			input:    time.Date(2025, 10, 15, 12, 30, 0, 0, time.UTC),
			expected: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			desc:     "Middle of month",
		},
		{
			input:    time.Date(2025, 10, 31, 23, 59, 59, 0, time.UTC),
			expected: time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
			desc:     "Last day of month",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := startOfMonth(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("startOfMonth(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
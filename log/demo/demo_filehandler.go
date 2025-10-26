package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tr1v3r/pkg/log"
)

func main() {
	// Clean up any existing logs directory
	os.RemoveAll("./demo_logs")

	fmt.Println("=== FileHandler Demo ===")

	// Example 1: Basic file logging
	fmt.Println("\n1. Basic file logging:")
	basicHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		LogDir:   "./demo_logs",
		Filename: "basic.log",
		Rotation: log.RotationNone,
		Level:    log.InfoLevel,
	})
	if err != nil {
		fmt.Printf("Error creating basic handler: %v\n", err)
		return
	}
	defer basicHandler.Close()

	basicLogger := log.NewLogger(basicHandler)
	basicLogger.Info("Application started with basic file logging")
	basicLogger.Warn("This is a warning message")
	fmt.Printf("Log file created: %s\n", basicHandler.GetCurrentFilePath())

	// Example 2: Daily rotation
	fmt.Println("\n2. Daily rotation:")
	dailyHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		LogDir:   "./demo_logs",
		Filename: "daily_app.log",
		Rotation: log.RotationDaily,
		Level:    log.DebugLevel,
	})
	if err != nil {
		fmt.Printf("Error creating daily handler: %v\n", err)
		return
	}
	defer dailyHandler.Close()

	dailyLogger := log.NewLogger(dailyHandler)
	dailyLogger.Debug("Debug message for daily logs")
	dailyLogger.Info("Info message for daily logs")
	fmt.Printf("Daily log file: %s\n", dailyHandler.GetCurrentFilePath())

	// Example 3: Hourly rotation with context
	fmt.Println("\n3. Hourly rotation with context:")
	ctx := log.WithLogID(context.Background(), "request-123")
	hourlyHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		LogDir:   "./demo_logs",
		Filename: "hourly_app.log",
		Rotation: log.RotationHourly,
		Level:    log.InfoLevel,
	})
	if err != nil {
		fmt.Printf("Error creating hourly handler: %v\n", err)
		return
	}
	defer hourlyHandler.Close()

	hourlyLogger := log.NewLogger(hourlyHandler)
	hourlyLogger.CtxInfo(ctx, "Request processed successfully")
	fmt.Printf("Hourly log file: %s\n", hourlyHandler.GetCurrentFilePath())

	// Example 4: Multiple handlers (console + file)
	fmt.Println("\n4. Multiple handlers (console + file):")
	fileHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		LogDir:   "./demo_logs",
		Filename: "multi.log",
		Rotation: log.RotationNone,
		Level:    log.InfoLevel,
	})
	if err != nil {
		fmt.Printf("Error creating file handler: %v\n", err)
		return
	}
	defer fileHandler.Close()

	consoleHandler := log.NewConsoleHandler(log.InfoLevel)
	defer consoleHandler.Close()

	multiLogger := log.NewLogger(consoleHandler, fileHandler)
	multiLogger.Info("This message goes to both console and file")
	fmt.Printf("Multi log file: %s\n", fileHandler.GetCurrentFilePath())

	// Example 5: Level filtering
	fmt.Println("\n5. Level filtering (Warn and above only):")
	filteredHandler, err := log.NewFileHandler(log.FileHandlerConfig{
		LogDir:   "./demo_logs",
		Filename: "filtered.log",
		Rotation: log.RotationNone,
		Level:    log.WarnLevel,
	})
	if err != nil {
		fmt.Printf("Error creating filtered handler: %v\n", err)
		return
	}
	defer filteredHandler.Close()

	filteredLogger := log.NewLogger(filteredHandler)
	filteredLogger.Trace("This trace message won't appear")
	filteredLogger.Debug("This debug message won't appear")
	filteredLogger.Info("This info message won't appear")
	filteredLogger.Warn("This warning message WILL appear")
	filteredLogger.Error("This error message WILL appear")
	fmt.Printf("Filtered log file: %s\n", filteredHandler.GetCurrentFilePath())

	// Example 6: Different rotation intervals
	fmt.Println("\n6. Different rotation intervals:")
	intervals := []struct {
		name     string
		interval log.RotationInterval
	}{
		{"None", log.RotationNone},
		{"Hourly", log.RotationHourly},
		{"Daily", log.RotationDaily},
		{"Weekly", log.RotationWeekly},
		{"Monthly", log.RotationMonthly},
	}

	for _, interval := range intervals {
		handler, err := log.NewFileHandler(log.FileHandlerConfig{
			LogDir:   "./demo_logs",
			Filename: fmt.Sprintf("%s_test.log", interval.name),
			Rotation: interval.interval,
			Level:    log.InfoLevel,
		})
		if err != nil {
			fmt.Printf("Error creating %s handler: %v\n", interval.name, err)
			continue
		}

		logger := log.NewLogger(handler)
		logger.Info("Test message for %s rotation", interval.name)
		fmt.Printf("  %s: %s\n", interval.name, handler.GetCurrentFilePath())
		handler.Close()
	}

	fmt.Println("\n=== Demo completed successfully ===")
	fmt.Println("Check the ./demo_logs directory for generated log files")
}
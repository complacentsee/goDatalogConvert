package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/complacentsee/goDatalogConvert/libdat"
)

const (
	BatchSize = 1000
)

func main() {
	// Define the command-line flag for the directory path
	dirPath := flag.String("path", ".", "Path to the directory containing DAT files")
	debugLevel := flag.Bool("debug", false, "Enable Debug Logging")
	flag.Parse()

	var programLevel = new(slog.LevelVar) // Info by default

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel}))
	slog.SetDefault(logger)

	if *debugLevel {
		programLevel.Set(slog.LevelDebug)
		slog.Debug("Debug level logging enabled")
	}

	// Check if the directory exists
	if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
		slog.Error("Error: Directory not found")
		return
	}

	// Initialize the DatReader
	dr, err := libdat.NewDatReader(*dirPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Process each file in the directory
	for _, floatfileName := range dr.GetFloatFiles() {

		slog.Info(fmt.Sprintf("Converting %s", filepath.Base(floatfileName)))
		pointIDs := make(map[int]int)

		// Read and process the Tag file associated with the Float file
		tags, err := dr.ReadTagFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading tag file for %s: %v", floatfileName, err))
			return
		}

		for _, tag := range tags {
			//libdat.PrintTagRecord(tag)
			pointIDs[tag.ID] = tag.ID // Simplified mapping since PI stuff is removed
		}

		// Process each record in the Float file
		records, err := dr.ReadFloatFile(floatfileName)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading float file for %s: %v", floatfileName, err))
			return
		}

		for _, record := range records {
			if record.IsValid {
				slog.Debug(fmt.Sprintf("Datetime: %s | TagID: %d | Value: %f | Status: %c | Marker: %c",
					record.Datetime.Format("2006-01-02 15:04:05.000"),
					record.TagID,
					record.Val,
					record.Status,
					record.Marker))
			}
		}

	}
}

// Placeholder function to process the batch (e.g., store the data)
// func processBatch(batchCount int, ptids []int32, vs []float64, ts []libdat.PITimestamp) {
// 	// This function would normally handle storing or further processing the batch
// 	fmt.Printf("Processing batch of %d records\n", batchCount)
// }

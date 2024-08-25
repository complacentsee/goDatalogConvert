package main

import (
	"flag"
	"fmt"
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
	flag.Parse()

	// Check if the directory exists
	if _, err := os.Stat(*dirPath); os.IsNotExist(err) {
		fmt.Println("Error: Directory not found")
		return
	}

	// Initialize the DatReader
	dr, err := libdat.NewDatReader(*dirPath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	allowedTagnames := []string{}
	// allowedTagnames = append(allowedTagnames, "AI\\FY59_1")

	// Process each file in the directory
	for _, floatfileName := range dr.GetFloatFiles() {
		fmt.Printf("Converting %s\n", filepath.Base(floatfileName))
		pointIDs := make(map[int]int)

		// Read and process the Tag file associated with the Float file
		// Read and process the Tag file associated with the Float file
		tags, err := dr.ReadTagFile(floatfileName)
		if err != nil {
			fmt.Printf("Error reading tag file for %s: %v\n", floatfileName, err)
			return
		}

		for _, tag := range tags {
			if len(allowedTagnames) > 0 && !contains(allowedTagnames, tag.Name) {
				continue
			}
			//libdat.PrintTagRecord(tag)
			pointIDs[tag.ID] = tag.ID // Simplified mapping since PI stuff is removed
		}

		// // Prepare for batch processing
		// batchCount := 0
		// ptids := make([]int32, BatchSize)
		// vs := make([]float64, BatchSize)
		// ts := make([]libdat.PITimestamp, BatchSize)

		// Process each record in the Float file
		records, err := dr.ReadFloatFile(floatfileName)
		if err != nil {
			fmt.Printf("Error reading float file for %s: %v\n", floatfileName, err)
			return
		}

		for _, record := range records {
			// Now, print the record in a readable format
			if record.IsValid {
				fmt.Printf("Datetime: %s | TagID: %d | Value: %f | Status: %c | Marker: %c\n",
					record.Datetime.Format("2006-01-02 15:04:05.000"),
					record.TagID,
					record.Val,
					record.Status,
					record.Marker)
			}
		}

	}
}

// Utility function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Placeholder function to process the batch (e.g., store the data)
// func processBatch(batchCount int, ptids []int32, vs []float64, ts []libdat.PITimestamp) {
// 	// This function would normally handle storing or further processing the batch
// 	fmt.Printf("Processing batch of %d records\n", batchCount)
// }

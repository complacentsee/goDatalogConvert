package libUtil

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
)

func LoadTagMapCSV(filePath string, tagMap map[string]string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read the file line by line to avoid loading everything into memory at once
	for i := 0; ; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading CSV file at line %d: %w", i+1, err)
		}

		if len(record) < 2 {
			slog.Error(fmt.Sprintf("Tag Name mapping file doesn't have two records on row %d", i+1))
			continue
		}

		tagMap[record[0]] = record[1]
	}

	return nil
}

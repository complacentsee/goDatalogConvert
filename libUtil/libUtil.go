package libUtil

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
)

func LoadTagMapCSV(filePath string, tagMap *map[string]string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Initialize the map if it’s nil
	if *tagMap == nil {
		*tagMap = make(map[string]string)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for i, record := range records {
		if len(record) < 2 {
			slog.Error(fmt.Sprintf("Tag Name mapping file doesn't have two records on row %d", i))
			continue // or handle the error as appropriate
		}
		(*tagMap)[record[0]] = record[1]
	}
	return nil
}

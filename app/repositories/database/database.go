package database

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// '''
// insights-logs-postgresqlflexdatabasexacts:

// Provides information about database activities, such as executed queries, transactions, and other database operations.
// insights-logs-postgresqlflexquerystoreruntime:

// Contains runtime statistics for queries, including execution times, CPU usage, and memory usage for individual queries.
// insights-logs-postgresqlflexquerystorewaitstats:

// Provides wait statistics for queries, helping to identify bottlenecks and performance issues related to waiting for resources.
// insights-logs-postgresqlflextablestats:

// Contains statistics related to table usage, such as read/write operations, index usage, and table scans.
// insights-logs-postgresqllogs:

// General PostgreSQL logs, including error logs, connection logs, and other server-related logs.
// '''

type PostgreSQLFlexQueryStoreRuntime struct {
	Category      string     `json:"-"`
	Location      string     `json:"-"`
	OperationName string     `json:"-"`
	Properties    Properties `json:"properties"`
	ResourceID    string     `json:"-"`
	Time          time.Time  `json:"time"`
}

type Properties struct {
	MinTime             float64   `json:"Min_time"`
	MaxTime             float64   `json:"Max_time"`
	MeanTime            float64   `json:"Mean_time"`
	StddevTime          float64   `json:"Stddev_time"`
	Rows                int       `json:"Rows"`
	SharedBlksHit       int       `json:"Shared_blks_hit"`
	SharedBlksRead      int       `json:"Shared_blks_read"`
	SharedBlksDirtied   int       `json:"Shared_blks_dirtied"`
	SharedBlksWritten   int       `json:"Shared_blks_written"`
	LocalBlksHit        int       `json:"Local_blks_hit"`
	LocalBlksRead       int       `json:"Local_blks_read"`
	TempBlksWritten     int       `json:"Temp_blks_written"`
	BlkReadTime         float64   `json:"Blk_read_time"`
	BlkWriteTime        float64   `json:"Blk_write_time"`
	IsSystemQuery       bool      `json:"Is_system_query"`
	QueryType           string    `json:"Query_type"`
	TempBlksRead        int       `json:"Temp_blks_read"`
	LocalBlksWritten    int       `json:"Local_blks_written"`
	LocalBlksDirtied    int       `json:"Local_blks_dirtied"`
	RuntimeStatsEntryID int       `json:"Runtime_stats_entry_id"`
	UserID              int       `json:"Userid"`
	DbID                int       `json:"Dbid"`
	QueryID             int64     `json:"Queryid"`
	QueryIDStr          string    `json:"Queryid_str"`
	PlanID              string    `json:"Plan_id"`
	StartTime           time.Time `json:"Start_time"`
	EndTime             time.Time `json:"End_time"`
	Calls               int       `json:"Calls"`
	TotalTime           float64   `json:"Total_time"`
}

func connectToAzureStorage() (*azblob.Client, error) {
	credential, err := azblob.NewSharedKeyCredential(os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"), os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"))
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %v", err)
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", os.Getenv("AZURE_STORAGE_ACCOUNT_NAME"))
	serviceClient, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create service client: %v", err)
	}

	return serviceClient, nil
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func generatePaths(startTime, endTime time.Time) []string {
	var paths []string
	for t := startTime; !t.After(endTime); t = t.Add(time.Hour) {
		yearStr := fmt.Sprintf("%d", t.Year())
		monthStr := fmt.Sprintf("%02d", t.Month())
		dayStr := fmt.Sprintf("%02d", t.Day())
		hourStr := fmt.Sprintf("%02d", t.Hour())

		path := fmt.Sprintf("/resourceId=/SUBSCRIPTIONS/0F5C9D51-B501-4B06-B637-9067BA7B3662/RESOURCEGROUPS/NANA/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/NANA/y=%s/m=%s/d=%s/h=%s/m=00/PT1H.json", yearStr, monthStr, dayStr, hourStr)
		paths = append(paths, path)
	}
	return paths
}

func downloadBlob(ctx context.Context, client *azblob.Client, path string) []PostgreSQLFlexQueryStoreRuntime {

	fmt.Println(path)
	downloadResponse, err := client.DownloadStream(ctx, "insights-logs-postgresqlflexquerystoreruntime", path, nil)
	handleError(err)

	// Assert that the content is correct
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	handleError(err)

	var models []PostgreSQLFlexQueryStoreRuntime
	scanner := bufio.NewScanner(bytes.NewReader(actualBlobData))
	for scanner.Scan() {
		var model PostgreSQLFlexQueryStoreRuntime
		err := json.Unmarshal(scanner.Bytes(), &model)
		if err != nil {
			// Try to fix time fields by replacing missing timezone with Z
			fixed := scanner.Bytes()
			fixedStr := string(fixed)
			// Add Z if missing at the end of time fields
			fixedStr = fixTimeFields(fixedStr)
			err = json.Unmarshal([]byte(fixedStr), &model)
		}
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err, "for path:", path, "data:", string(scanner.Bytes()))
			handleError(err)
		}
		models = append(models, model)
	}
	if err := scanner.Err(); err != nil {
		handleError(err)
	}

	fmt.Println(len(actualBlobData))
	return models
}

// fixTimeFields adds 'Z' to time fields if missing
func fixTimeFields(jsonStr string) string {
	re := regexp.MustCompile(`"(time|Start_time|End_time)":"([0-9T:\-\.]+)"`)
	return re.ReplaceAllStringFunc(jsonStr, func(s string) string {
		// Check if ends with Z or timezone
		if s[len(s)-2:] == "Z" || s[len(s)-6:] == "+00:00" || s[len(s)-3:] == "+00" {
			return s
		}
		// Insert Z before the last quote
		return s[:len(s)-1] + "Z" + `"`
	})
}

package database

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type PostgresMetric struct {
	Count      int       `json:"count"`
	Total      float64   `json:"total"`
	Minimum    float64   `json:"minimum"`
	Maximum    float64   `json:"maximum"`
	Average    float64   `json:"average"`
	ResourceID string    `json:"resourceId"`
	Time       time.Time `json:"time"`
	MetricName string    `json:"metricName"`
	TimeGrain  string    `json:"timeGrain"`
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

func downloadBlob(ctx context.Context, client *azblob.Client, path string) ([]PostgresMetric, bool) {

	fmt.Println(path)
	downloadResponse, err := client.DownloadStream(ctx, "insights-metrics-pt1m", path, nil)
	if err != nil {
		var responseErr *azcore.ResponseError
		if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
			// Ignore 404 error
			fmt.Println(fmt.Sprintf("Path does not exist, ignoring 404 error: %s", path))
			return nil, false
		} else {
			handleError(err)
			return nil, false
		}
	}

	fmt.Sprintf("Reading path: %s", path)
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	handleError(err)

	var models []PostgresMetric
	scanner := bufio.NewScanner(bytes.NewReader(actualBlobData))
	for scanner.Scan() {
		var model PostgresMetric
		err := json.Unmarshal(scanner.Bytes(), &model)
		if err != nil {
			handleError(err)
		}
		models = append(models, model)
	}
	if err := scanner.Err(); err != nil {
		handleError(err)
	}

	fmt.Println(len(actualBlobData))
	return models, true
}

func main() {
	client, err := connectToAzureStorage()
	if err != nil {
		fmt.Println(err)
	}

	ctx := context.Background()

	inputStartTime := "2025-01-01T01:11:23.000Z"
	parsedInputStartTime, err := time.Parse(time.RFC3339, inputStartTime)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return
	}
	parsedInputStartTime = parsedInputStartTime.Truncate(time.Hour)

	inputEndTime := "2025-01-02T01:11:23.000Z"
	parsedInputEndTime, err := time.Parse(time.RFC3339, inputEndTime)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return
	}
	parsedInputEndTime = parsedInputEndTime.Truncate(time.Hour)

	paths := generatePaths(parsedInputStartTime, parsedInputEndTime)

	var wg sync.WaitGroup
	results := make(chan []PostgresMetric, len(paths))
	var storeRuntimes []PostgresMetric

	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			result, success := downloadBlob(ctx, client, path)
			if success {
				results <- result
			}
		}(path)
	}

	wg.Wait()
	close(results)

	for result := range results {
		storeRuntimes = append(storeRuntimes, result...)
	}

	fmt.Println(storeRuntimes)

}

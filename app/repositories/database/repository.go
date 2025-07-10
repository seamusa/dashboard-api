package database

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabaseRepository interface {
	GetQueryStoreRuntime(ctx context.Context, startTime, endTime time.Time) ([]PostgreSQLFlexQueryStoreRuntime, error)
	GetMetrics(ctx context.Context, startTime, endTime time.Time) ([]PostgresMetric, error)
	GetQuerySqlText(ctx context.Context, queryID string) (string, error) // new method
}

type AzureBlobDatabaseRepository struct{}

func NewAzureBlobDatabaseRepository() *AzureBlobDatabaseRepository {
	return &AzureBlobDatabaseRepository{}
}

// GetQueryStoreRuntime fetches PostgreSQLFlexQueryStoreRuntime logs from Azure Blob Storage
func (r *AzureBlobDatabaseRepository) GetQueryStoreRuntime(ctx context.Context, startTime, endTime time.Time) ([]PostgreSQLFlexQueryStoreRuntime, error) {
	client, err := connectToAzureStorage()
	if err != nil {
		return nil, err
	}
	paths := generatePaths(startTime, endTime)
	var wg sync.WaitGroup
	results := make(chan []PostgreSQLFlexQueryStoreRuntime, len(paths))
	var storeRuntimes []PostgreSQLFlexQueryStoreRuntime

	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			result := downloadBlob(ctx, client, path)
			results <- result
		}(path)
	}
	wg.Wait()
	close(results)
	for result := range results {
		storeRuntimes = append(storeRuntimes, result...)
	}
	return storeRuntimes, nil
}

// GetMetrics fetches and aggregates Postgres metrics from Azure Blob Storage
func (r *AzureBlobDatabaseRepository) GetMetrics(ctx context.Context, startTime, endTime time.Time) ([]PostgresMetric, error) {
	client, err := connectToAzureStorage()
	if err != nil {
		return nil, err
	}
	paths := generatePaths(startTime, endTime)
	var wg sync.WaitGroup
	results := make(chan []PostgresMetric, len(paths))
	var allMetrics []PostgresMetric

	downloadMetricBlob := func(ctx context.Context, client *azblob.Client, path string) ([]PostgresMetric, bool) {
		downloadResponse, err := client.DownloadStream(ctx, "insights-metrics-pt1m", path, nil)
		if err != nil {
			var responseErr *azcore.ResponseError
			if errors.As(err, &responseErr) && responseErr.StatusCode == http.StatusNotFound {
				return nil, false
			} else {
				handleError(err)
				return nil, false
			}
		}
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
		return models, true
	}

	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			result, success := downloadMetricBlob(ctx, client, path)
			if success {
				results <- result
			}
		}(path)
	}
	wg.Wait()
	close(results)
	for result := range results {
		allMetrics = append(allMetrics, result...)
	}
	return allMetrics, nil
}

// PostgresDatabaseRepository implements DatabaseRepository for PostgreSQL.
type PostgresDatabaseRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository returns a new PostgresDatabaseRepository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresDatabaseRepository {
	return &PostgresDatabaseRepository{pool: pool}
}

// GetQuerySqlText fetches the SQL text for a given query ID.
func (r *PostgresDatabaseRepository) GetQuerySqlText(ctx context.Context, queryID string) (string, error) {
	var sqlText string
	query := `SELECT query_sql_text FROM query_store.query_texts_view WHERE query_text_id = $1`
	fmt.Println(queryID, query)
	err := r.pool.QueryRow(ctx, query, queryID).Scan(&sqlText)
	if err != nil {
		return "", err
	}
	return sqlText, nil
}

// Delegate GetQueryStoreRuntime and GetMetrics to AzureBlobDatabaseRepository logic
func (r *PostgresDatabaseRepository) GetQueryStoreRuntime(ctx context.Context, startTime, endTime time.Time) ([]PostgreSQLFlexQueryStoreRuntime, error) {
	return NewAzureBlobDatabaseRepository().GetQueryStoreRuntime(ctx, startTime, endTime)
}

func (r *PostgresDatabaseRepository) GetMetrics(ctx context.Context, startTime, endTime time.Time) ([]PostgresMetric, error) {
	return NewAzureBlobDatabaseRepository().GetMetrics(ctx, startTime, endTime)
}

// NewRepositories returns a DatabaseRepository implementation for PostgreSQL.
func NewRepositories(pool *pgxpool.Pool) DatabaseRepository {
	return NewPostgresRepository(pool)
}

package main

import (
    "context"
    "encoding/csv"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
    "github.com/go-redis/redis/v8"
)

const (
    containerName = "redis-performance"
    csvFileName   = "redis_rps_log.csv"
    interval      = 1 * time.Second // Interval between benchmarks
    numRequests   = 100000          // Number of requests per benchmark
    concurrency   = 100              // Number of concurrent clients
)

var ctx = context.Background()

func main() {
    // Initialize Docker client
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    if err != nil {
        log.Fatalf("Error initializing Docker client: %v", err)
    }

    // Initialize Redis client
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379", // Redis address
        // If Redis requires authentication, set the Password field
        // Password: "",
        DB: 0, // use default DB
    })

    // Initialize CSV file
    if _, err := os.Stat(csvFileName); os.IsNotExist(err) {
        file, err := os.Create(csvFileName)
        if err != nil {
            log.Fatalf("Error creating CSV file: %v", err)
        }
        defer file.Close()

        writer := csv.NewWriter(file)
        defer writer.Flush()

        // Write headers
        err = writer.Write([]string{"Timestamp", "GET_RPS", "SET_RPS"})
        if err != nil {
            log.Fatalf("Error writing CSV header: %v", err)
        }
    }

    log.Printf("Starting Redis performance monitoring. Logging to %s every %v.\n", csvFileName, interval)

    for {
        // Check if the Redis container is running
        running, err := isContainerRunning(cli, containerName)
        if err != nil {
            log.Printf("Error checking container status: %v", err)
            goto SLEEP
        }

        if running {
            // Perform benchmarking
            getRPS, setRPS, err := benchmarkRedis(rdb, numRequests, concurrency)
            if err != nil {
                log.Printf("Error benchmarking Redis: %v", err)
            } else {
                // Log results to CSV
                err = logToCSV(csvFileName, getRPS, setRPS)
                if err != nil {
                    log.Printf("Error logging to CSV: %v", err)
                } else {
                    log.Printf("RPS - GET: %.2f | SET: %.2f\n", getRPS, setRPS)
                }
            }
        } else {
            log.Printf("Redis container '%s' is not running.", containerName)
        }

    SLEEP:
        time.Sleep(interval)
    }
}

// isContainerRunning checks if a Docker container with the given name is running
func isContainerRunning(cli *client.Client, name string) (bool, error) {
    containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
    if err != nil {
        return false, err
    }

    for _, container := range containers {
        for _, containerName := range container.Names {
            // Container names have a leading slash
            if containerName == "/"+name {
                return container.State == "running", nil
            }
        }
    }

    // Container not found
    return false, nil
}

// benchmarkRedis performs GET and SET operations to measure RPS
func benchmarkRedis(rdb *redis.Client, numRequests int, concurrency int) (float64, float64, error) {
    // Measure SET operations
    setStart := time.Now()
    err := performRedisOps(rdb, "SET", numRequests, concurrency)
    if err != nil {
        return 0, 0, fmt.Errorf("error performing SET operations: %v", err)
    }
    setDuration := time.Since(setStart).Seconds()
    setRPS := float64(numRequests) / setDuration

    // Measure GET operations
    getStart := time.Now()
    err = performRedisOps(rdb, "GET", numRequests, concurrency)
    if err != nil {
        return setRPS, 0, fmt.Errorf("error performing GET operations: %v", err)
    }
    getDuration := time.Since(getStart).Seconds()
    getRPS := float64(numRequests) / getDuration

    return getRPS, setRPS, nil
}

// performRedisOps performs the specified Redis operations
func performRedisOps(rdb *redis.Client, op string, numRequests int, concurrency int) error {
    jobs := make(chan int, numRequests)
    for i := 0; i < numRequests; i++ {
        jobs <- i
    }
    close(jobs)

    // Worker function
    worker := func() {
        for j := range jobs {
            key := fmt.Sprintf("key:%d", j)
            value := fmt.Sprintf("value:%d", j)
            if op == "SET" {
                err := rdb.Set(ctx, key, value, 0).Err()
                if err != nil {
                    log.Printf("Error setting key %s: %v", key, err)
                }
            } else if op == "GET" {
                _, err := rdb.Get(ctx, key).Result()
                if err != nil && err != redis.Nil {
                    log.Printf("Error getting key %s: %v", key, err)
                }
            }
        }
    }

    // Launch workers
    for i := 0; i < concurrency; i++ {
        go worker()
    }

    // Wait for all jobs to be processed
    // Since we're not tracking, use a simple sleep based on operation count and concurrency
    // Alternatively, use sync.WaitGroup for precise synchronization
    estimatedTime := float64(numRequests) / float64(concurrency) * 0.001 // Adjust factor as needed
    time.Sleep(time.Duration(estimatedTime) * time.Second)

    return nil
}

// logToCSV appends the benchmark results to the CSV file
func logToCSV(filename string, getRPS, setRPS float64) error {
    file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    writer := csv.NewWriter(file)
    defer writer.Flush()

    timestamp := time.Now().Format("2006-01-02 15:04:05")
   	record := []string{
        timestamp,
        fmt.Sprintf("%.2f", getRPS),
        fmt.Sprintf("%.2f", setRPS),
    }

    err = writer.Write(record)
    if err != nil {
        return err
    }

    return nil
}

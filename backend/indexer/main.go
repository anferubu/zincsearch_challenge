package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"enron-email-search/shared"
)

const (
	BATCH_SIZE = 10000
	WORKERS    = 10
	SENDERS    = 5
)

type EmailBatch struct {
	Emails []shared.Email
	mutex  sync.RWMutex
}

func createIndex(config *shared.Config) error {
	client := &http.Client{Timeout: time.Second * 30}

	// Check if the index already exists
	url := fmt.Sprintf("%s/api/index/%s", config.ZINC_HOST, config.ZINC_INDEX)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request to check index existence: %v", err)
	}
	req.SetBasicAuth(config.ZINC_USER, config.ZINC_PASSWORD)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error checking if index exists: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Delete index if it exists
		log.Printf("Index %s exists. Deleting it...\n", config.ZINC_INDEX)
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return fmt.Errorf("error creating delete request: %v", err)
		}
		req.SetBasicAuth(config.ZINC_USER, config.ZINC_PASSWORD)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("error deleting index: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("error deleting index: %s", string(body))
		}

		log.Println("Index deleted successfully.")
	} else if resp.StatusCode != http.StatusNotFound {
		// If the status is not 404 (index not found), something went wrong
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response checking index existence: %s", string(body))
	}

	// Create new index
	url = fmt.Sprintf("%s/api/index", config.ZINC_HOST)

	mapping := map[string]interface{}{
		"name":         config.ZINC_INDEX,
		"storage_type": "disk",
		"shard_num":    6, // equal or less than your CPU cores
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"from": map[string]interface{}{
					"type":  "keyword",
					"store": true,
				},
				"to": map[string]interface{}{
					"type":  "keyword",
					"store": true,
				},
				"subject": map[string]interface{}{
					"type":  "text",
					"store": true,
				},
				"body": map[string]interface{}{
					"type":  "text",
					"store": true,
				},
				"datetime": map[string]interface{}{
					"type":   "date",
					"store":  true,
					"format": "2006-01-02T15:04:05.000Z",
				},
			},
		},
	}

	payload, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("error serializing index mapping: %v", err)
	}

	req, err = http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.SetBasicAuth(config.ZINC_USER, config.ZINC_PASSWORD)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error from ZincSearch: %s", string(body))
	}

	log.Println("Index created or updated successfully.")
	return nil
}

func parseEmail(filePath string) (shared.Email, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return shared.Email{}, fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close()

	var email shared.Email
	var bodyBuffer strings.Builder
	bodyBuffer.Grow(4096) // Average file size: ~4 KB

	reader := bufio.NewReader(file)
	bodyStarted := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return shared.Email{}, fmt.Errorf("error reading file %s: %w", filePath, err)
		}

		line = strings.TrimRight(line, "\r\n")

		if err == io.EOF && line == "" {
			break
		}

		if strings.TrimSpace(line) == "" && !bodyStarted {
			bodyStarted = true
			continue
		}

		if !bodyStarted {
			switch {
			case strings.HasPrefix(line, "Message-ID:"):
				email.ID = strings.TrimSpace(strings.TrimPrefix(line, "Message-ID:"))
			case strings.HasPrefix(line, "Date:"):
				dateStr := strings.TrimSpace(strings.TrimPrefix(line, "Date:"))
				parsedTime, err := time.Parse("Mon, _2 Jan 2006 15:04:05 -0700 (MST)", dateStr)
				if err == nil {
					email.Datetime = parsedTime.UTC().Format("2006-01-02T15:04:05.000Z")
				}
			case strings.HasPrefix(line, "From:"):
				email.From = strings.TrimSpace(strings.TrimPrefix(line, "From:"))
			case strings.HasPrefix(line, "To:"):
				email.To = strings.TrimSpace(strings.TrimPrefix(line, "To:"))
			case strings.HasPrefix(line, "Subject:"):
				email.Subject = strings.TrimSpace(strings.TrimPrefix(line, "Subject:"))
			}
		} else {
			bodyBuffer.WriteString(line + "\n")
		}

		if err == io.EOF {
			break
		}
	}

	email.Body = bodyBuffer.String()
	return email, nil
}

func processFiles(filesChan <-chan string, batchChan chan<- *EmailBatch, errorsChan chan<- error) {
	batch := &EmailBatch{
		Emails: make([]shared.Email, 0, BATCH_SIZE),
	}

	for filePath := range filesChan {
		email, err := parseEmail(filePath)
		if err != nil {
			errorsChan <- fmt.Errorf("error en archivo %s: %w", filePath, err)
			continue
		}

		// Verify that the email is valid
		if email.ID == "" || email.Body == "" {
			continue
		}

		// Add email to the current batch
		batch.mutex.Lock()
		batch.Emails = append(batch.Emails, email)
		batch.mutex.Unlock()

		currentLen := len(batch.Emails)

		if currentLen >= BATCH_SIZE {
			log.Printf("Sending batch of %d emails", currentLen)
			batchChan <- batch

			batch = &EmailBatch{
				Emails: make([]shared.Email, 0, BATCH_SIZE),
			}
		}
	}

	// Send the last batch if it contains emails
	batch.mutex.RLock()
	remainingEmails := len(batch.Emails)
	batch.mutex.RUnlock()

	if remainingEmails > 0 {
		log.Printf("Sending final batch of %d emails", remainingEmails)
		batchChan <- batch
	}
}

func sendBatches(batchChan <-chan *EmailBatch, config *shared.Config, errorsChan chan<- error, numSenders int) error {
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			MaxConnsPerHost:     50,
		},
	}

	url := fmt.Sprintf("%s/api/_bulkv2", config.ZINC_HOST)

	// Launch multiple sender goroutines
	for i := 0; i < numSenders; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for batch := range batchChan {
				bulkData := struct {
					Index   string         `json:"index"`
					Records []shared.Email `json:"records"`
				}{
					Index:   config.ZINC_INDEX,
					Records: batch.Emails,
				}

				payload, err := json.Marshal(bulkData)
				if err != nil {
					errorsChan <- fmt.Errorf("worker %d error serializing batch: %w", workerID, err)
					continue
				}

				req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
				if err != nil {
					errorsChan <- fmt.Errorf("worker %d error creating request: %w", workerID, err)
					continue
				}

				req.SetBasicAuth(config.ZINC_USER, config.ZINC_PASSWORD)
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					errorsChan <- fmt.Errorf("worker %d error sending request: %w", workerID, err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					errorsChan <- fmt.Errorf("worker %d error from ZincSearch: %s", workerID, body)
					continue
				}
				resp.Body.Close()

				log.Printf("Worker %d successfully indexed batch of %d emails\n", workerID, len(batch.Emails))
			}
		}(i)
	}

	wg.Wait() // wait until all goroutines have called Done
	return nil
}

func indexEmails(rootDir string, config *shared.Config) error {
	filesChan := make(chan string, BATCH_SIZE)
	batchChan := make(chan *EmailBatch, WORKERS*3) // A large size prevents immediate blockages
	errorsChan := make(chan error, 100)

	// Start file processing workers
	var wg sync.WaitGroup
	for i := 0; i < WORKERS; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processFiles(filesChan, batchChan, errorsChan)
		}()
	}

	// Start batch sending worker
	var sendWg sync.WaitGroup
	sendWg.Add(1)
	go func() {
		defer sendWg.Done()
		sendBatches(batchChan, config, errorsChan, SENDERS)
	}()

	// Walk through directory and send files to workers
	fileCount := 0
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			filesChan <- path
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	log.Printf("Found %d files to process", fileCount)
	close(filesChan)

	// Read errors during execution
	var errWg sync.WaitGroup
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for err := range errorsChan {
			log.Printf("Error during processing: %v\n", err)
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(batchChan)
	sendWg.Wait()
	close(errorsChan)
	errWg.Wait()

	return nil
}

func main() {
	// Setup CPU profiling
	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// -----------------------------------------------------------------------
	// Main process
	// Load env variables
	config, err := shared.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Create index
	if err := createIndex(config); err != nil {
		log.Fatalf("Error creating index: %v", err)
	}

	// Validate command-line arguments
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./indexer [emails-path]")
	}

	// Process and index emails
	dir := os.Args[1]
	startTime := time.Now()

	if err := indexEmails(dir, config); err != nil {
		log.Fatalf("Error indexing emails: %v", err)
	}

	// -----------------------------------------------------------------------
	// Other Profiling
	// Setup goroutine profiling
	grFile, err := os.Create("goroutine.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer grFile.Close()

	if err := pprof.Lookup("goroutine").WriteTo(grFile, 0); err != nil {
		log.Fatal(err)
	}

	// Setup memory profiling
	memFile, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer memFile.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		log.Fatal(err)
	}

	log.Printf("Indexing completed in %v\n", time.Since(startTime))
}

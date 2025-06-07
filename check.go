package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec" 
	"runtime"  
	"sync"
)

const testURL = "https://httpbin.org/user-agent"
type FailedResult struct {
	UserAgent string
	Reason    string
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func checkUserAgent(ua string, failedChan chan<- FailedResult, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Error creating request: %v", err)}
		return
	}

	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Request failed: %v", err)}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Received status code: %d", resp.StatusCode)}
	}
}

func main() {

	clearScreen()

	if len(os.Args) < 2 {
		log.Fatal("Please enter:\n go run main.go user_agents.txt")
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening file '%s': %v", fileName, err)
	}
	defer file.Close()

	var wg sync.WaitGroup
	failedChan := make(chan FailedResult)

	var failedUserAgents []FailedResult

	var readerWg sync.WaitGroup
	readerWg.Add(1)
	go func() {
		defer readerWg.Done()
		for result := range failedChan {
			failedUserAgents = append(failedUserAgents, result)
		}
	}()

	scanner := bufio.NewScanner(file)
	fmt.Println("🔎 Starting to check User-Agents... Please wait.")

	for scanner.Scan() {
		userAgent := scanner.Text()
		if userAgent != "" {
			wg.Add(1)
			go checkUserAgent(userAgent, failedChan, &wg)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	wg.Wait()
	close(failedChan)
	readerWg.Wait()
	clearScreen()
	fmt.Println("✅ Review completed.")
	fmt.Println("------------------------------------")

	if len(failedUserAgents) == 0 {
		fmt.Println("🎉 All User-Agents are working correctly!")
	} else {
		fmt.Printf("❌ Found %d unsuccessful User-Agent(s):\n\n", len(failedUserAgents))
		for _, result := range failedUserAgents {
			fmt.Printf("User-Agent: %s\n", result.UserAgent)
			fmt.Printf("Reason: %s\n\n", result.Reason)
		}
	}
}

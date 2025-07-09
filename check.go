package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	testURL    = "https://httpbin.org/user-agent"
	maxRetries = 3
	timeoutSec = 5
)

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

func printProgress(current, total int) {
	percent := (current * 100) / total
	bar := strings.Repeat("█", percent/2) + strings.Repeat("-", 50-percent/2)
	fmt.Printf("\rProgress: [%s] %d%% (%d/%d)", bar, percent, current, total)
	if current == total {
		fmt.Print("\n")
	}
}


func checkUserAgent(ua string, activeChan chan<- string, failedChan chan<- FailedResult, semaphore chan struct{}, wg *sync.WaitGroup, progressChan chan<- struct{}) {
	defer wg.Done()
	defer func() { <-semaphore }() 

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		client := &http.Client{Timeout: timeoutSec * time.Second}
		req, err := http.NewRequest("GET", testURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("Error creating request: %v", err)
			continue
		}
		req.Header.Set("User-Agent", ua)
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("Request failed: %v", err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("Received status code: %d", resp.StatusCode)
			continue
		}
		activeChan <- ua
		progressChan <- struct{}{}
		return
	}

	failedChan <- FailedResult{UserAgent: ua, Reason: lastErr.Error()}
	progressChan <- struct{}{}
}

func getUserAgentsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var agents []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ua := scanner.Text()
		if ua != "" {
			agents = append(agents, ua)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return agents, nil
}

func getUserAgentsFromInput() []string {
	fmt.Println("Enter your User-Agents separated by a comma ',' (e.g. UA1,UA2,UA3):")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	parts := strings.Split(line, ",")
	var agents []string
	for _, ua := range parts {
		trimmed := strings.TrimSpace(ua)
		if trimmed != "" {
			agents = append(agents, trimmed)
		}
	}
	return agents
}


func chooseSpeedMenu() int {
	fmt.Println("\nSelect speed:")
	fmt.Println("1 - Fast (50 concurrent checks)")
	fmt.Println("2 - Medium (10 concurrent checks)")
	fmt.Print("Enter your choice (1 or 2): ")
	var speedChoice string
	fmt.Scanln(&speedChoice)
	switch speedChoice {
	case "1":
		return 50
	case "2":
		return 10
	default:
		fmt.Println("Invalid speed. Defaulting to Medium (10).")
		return 10
	}
}

func runCheckProcess(userAgents []string, concurrency int) {
	total := len(userAgents)
	if total == 0 {
		fmt.Println("No User-Agents found.")
		return
	}

	var wg sync.WaitGroup
	activeChan := make(chan string)
	failedChan := make(chan FailedResult)
	progressChan := make(chan struct{})
	semaphore := make(chan struct{}, concurrency)

	var activeUserAgents []string
	var failedUserAgents []FailedResult

	var readerWg sync.WaitGroup
	readerWg.Add(2)


	go func() {
		defer readerWg.Done()
		for ua := range activeChan {
			activeUserAgents = append(activeUserAgents, ua)
		}
	}()


	go func() {
		defer readerWg.Done()
		for result := range failedChan {
			failedUserAgents = append(failedUserAgents, result)
		}
	}()


	var progressWg sync.WaitGroup
	progressWg.Add(1)
	go func() {
		defer progressWg.Done()
		current := 0
		for range progressChan {
			current++
			printProgress(current, total)
		}
	}()

	fmt.Println("🔎 Checking User-Agents... Please wait.")
	startTime := time.Now()
	for _, userAgent := range userAgents {
		semaphore <- struct{}{} 
		wg.Add(1)
		go checkUserAgent(userAgent, activeChan, failedChan, semaphore, &wg, progressChan)
	}

	wg.Wait()
	close(activeChan)
	close(failedChan)
	close(progressChan)
	readerWg.Wait()
	progressWg.Wait()
	elapsed := time.Since(startTime)

	clearScreen()
	fmt.Println("✅ Review completed.")
	fmt.Println("------------------------------------")

	fmt.Println("🎯 Active User-Agents:")
	if len(activeUserAgents) == 0 {
		fmt.Println("No active User-Agents found!")
	} else {
		for _, ua := range activeUserAgents {
			fmt.Println(ua)
			fmt.Println("------------------------------------")
		}
	}
	fmt.Println("------------------------------------")

	if len(failedUserAgents) == 0 {
		fmt.Println("🎉 All User-Agents are working correctly!")
	} else {
		fmt.Printf("❌ %d inactive User-Agent(s) found:\n\n", len(failedUserAgents))
		for _, result := range failedUserAgents {
			fmt.Printf("User-Agent: %s\n", result.UserAgent)
			fmt.Printf("Reason: %s\n", result.Reason)
			fmt.Println("------------------------------------")
		}
	}

	fmt.Printf("\nSummary:\nTotal: %d  |  Active: %d  |  Inactive: %d  |  Time: %s\n",
		total, len(activeUserAgents), len(failedUserAgents), elapsed.Round(time.Second).String())
}

func main() {
	clearScreen()
	fmt.Println("Welcome to User-Agent Checker!")
	fmt.Println("==============================")
	fmt.Println("Please choose an option:")
	fmt.Println("1 - Use default User-Agents from user_agents.txt")
	fmt.Println("2 - Enter your own User-Agents (comma separated)")
	fmt.Print("Enter your choice (1 or 2): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		agents, err := getUserAgentsFromFile("user_agents.txt")
		if err != nil {
			fmt.Printf("Error reading user_agents.txt: %v\n", err)
			return
		}
		concurrency := chooseSpeedMenu()
		runCheckProcess(agents, concurrency)
	case "2":
		agents := getUserAgentsFromInput()
		concurrency := chooseSpeedMenu()
		runCheckProcess(agents, concurrency)
	default:
		fmt.Println("Invalid option. Exiting.")
	}
}

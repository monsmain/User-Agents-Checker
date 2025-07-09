package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
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


func printProgress(current, total int) {
	percent := (current * 100) / total
	bar := strings.Repeat("█", percent/2) + strings.Repeat("-", 50-percent/2)
	fmt.Printf("\rProgress: [%s] %d%% (%d/%d)", bar, percent, current, total)
	if current == total {
		fmt.Println()
	}
}

func checkUserAgent(ua string, activeChan chan<- string, failedChan chan<- FailedResult, wg *sync.WaitGroup, progressChan chan<- struct{}) {
	defer wg.Done()

	client := &http.Client{}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Error creating request: %v", err)}
		progressChan <- struct{}{}
		return
	}

	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Request failed: %v", err)}
		progressChan <- struct{}{}
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		failedChan <- FailedResult{UserAgent: ua, Reason: fmt.Sprintf("Received status code: %d", resp.StatusCode)}
		progressChan <- struct{}{}
		return
	}
	activeChan <- ua
	progressChan <- struct{}{}
}

func main() {
	clearScreen()

	if len(os.Args) < 2 {
		log.Fatal("Please run as:\n go run check.go user_agents.txt")
	}

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening file '%s': %v", fileName, err)
	}
	defer file.Close()


	var userAgents []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ua := scanner.Text()
		if ua != "" {
			userAgents = append(userAgents, ua)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	total := len(userAgents)
	if total == 0 {
		log.Fatal("No User-Agents found in file.")
	}

	file.Seek(0, 0)

	var wg sync.WaitGroup
	activeChan := make(chan string)
	failedChan := make(chan FailedResult)
	progressChan := make(chan struct{})

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

	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		userAgent := scanner.Text()
		if userAgent != "" {
			wg.Add(1)
			go checkUserAgent(userAgent, activeChan, failedChan, &wg, progressChan)
		}
	}

	wg.Wait()
	close(activeChan)
	close(failedChan)
	close(progressChan)
	readerWg.Wait()
	progressWg.Wait()

	clearScreen()
	fmt.Println("✅ Review completed.")
	fmt.Println("------------------------------------")

	// Show active User-Agents
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

	// Show failed User-Agents
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
}

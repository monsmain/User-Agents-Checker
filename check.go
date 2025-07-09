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

func checkUserAgent(ua string, activeChan chan<- string, failedChan chan<- FailedResult, wg *sync.WaitGroup) {
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
		return
	}
	activeChan <- ua
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

	var wg sync.WaitGroup
	activeChan := make(chan string)
	failedChan := make(chan FailedResult)

	var activeUserAgents []string
	var failedUserAgents []FailedResult

	var readerWg sync.WaitGroup
	readerWg.Add(2)

	// Goroutine to collect active User-Agents
	go func() {
		defer readerWg.Done()
		for ua := range activeChan {
			activeUserAgents = append(activeUserAgents, ua)
		}
	}()

	// Goroutine to collect failed User-Agents
	go func() {
		defer readerWg.Done()
		for result := range failedChan {
			failedUserAgents = append(failedUserAgents, result)
		}
	}()

	scanner := bufio.NewScanner(file)
	fmt.Println("🔎 Checking User-Agents... Please wait.")

	for scanner.Scan() {
		userAgent := scanner.Text()
		if userAgent != "" {
			wg.Add(1)
			go checkUserAgent(userAgent, activeChan, failedChan, &wg)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	wg.Wait()
	close(activeChan)
	close(failedChan)
	readerWg.Wait()

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
			fmt.Printf("Reason: %s\n\n", result.Reason)
		}
	}
}

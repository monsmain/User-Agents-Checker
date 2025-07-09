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

// printProgress prints a progress bar in the terminal (always on the same line)
func printProgress(current, total int) {
	percent := (current * 100) / total
	bar := strings.Repeat("█", percent/2) + strings.Repeat("-", 50-percent/2)
	fmt.Printf("\rProgress: [%s] %d%% (%d/%d)", bar, percent, current, total)
	if current == total {
		fmt.Print("\n") // Print newline only at the end
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

func runCheckProcess(userAgents []string) {
	total := len(userAgents)
	if total == 0 {
		fmt.Println("No User-Agents found.")
		return
	}

	var wg sync.WaitGroup
	activeChan := make(chan string)
	failedChan := make(chan FailedResult)
	progressChan := make(chan struct{})

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

	// Goroutine to print progress
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
	for _, userAgent := range userAgents {
		wg.Add(1)
		go checkUserAgent(userAgent, activeChan, failedChan, &wg, progressChan)
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
		runCheckProcess(agents)
	case "2":
		agents := getUserAgentsFromInput()
		runCheckProcess(agents)
	default:
		fmt.Println("Invalid option. Exiting.")
	}
}

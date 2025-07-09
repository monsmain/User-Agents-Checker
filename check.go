package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
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

const (
	ColorGreen  = "\033[01;32m"
	ColorRed    = "\033[01;31m"
	ColorYellow = "\033[01;33m"
	ColorWhite  = "\033[01;37m"
	ColorReset  = "\033[0m"
)

var userAgentPattern = regexp.MustCompile(`^(Mozilla\/5\.0|Opera\/).*(Chrome|Firefox|Safari|Edge|OPR|Opera|MSIE|Trident)\/[0-9\.]+.*$`)

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
	fmt.Printf(ColorYellow+"\rProgress: [%s] %d%% (%d/%d)"+ColorReset, bar, percent, current, total)
	if current == total {
		fmt.Print("\n")
	}
}

func checkUserAgent(ua string, activeChan chan<- string, failedChan chan<- FailedResult, semaphore chan struct{}, wg *sync.WaitGroup, progressChan chan<- struct{}) {
	defer wg.Done()
	defer func() { <-semaphore }()

	if !userAgentPattern.MatchString(ua) {
		failedChan <- FailedResult{UserAgent: ua, Reason: "Invalid User-Agent structure (not real UA)"}
		progressChan <- struct{}{}
		return
	}

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
	fmt.Println(ColorWhite + "Enter your User-Agents separated by a comma ',' (e.g. UA1,UA2,UA3):" + ColorReset)
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

func chooseSpeedMenu() (int, string) {
	fmt.Print(ColorGreen + ":Choose speed [medium/fast] " + ColorReset)
	var speedChoice string
	fmt.Scanln(&speedChoice)
	speedChoice = strings.ToLower(strings.TrimSpace(speedChoice))
	switch speedChoice {
	case "fast":
		fmt.Println(ColorYellow + "Fast mode selected." + ColorReset)
		return 50, "Fast mode selected."
	case "medium":
		fmt.Println(ColorYellow + "Medium mode selected." + ColorReset)
		return 10, "Medium mode selected."
	default:
		fmt.Println(ColorRed + "Invalid speed choice. Defaulting to medium mode" + ColorReset)
		return 10, "Invalid speed choice. Defaulting to medium mode"
	}
}

func runCheckProcess(userAgents []string, concurrency int) {
	total := len(userAgents)
	if total == 0 {
		fmt.Println(ColorRed + "No User-Agents found." + ColorReset)
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

	fmt.Println(ColorYellow + "🔎 Checking User-Agents... Please wait." + ColorReset)
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
	fmt.Println(ColorGreen + "✅ Review completed." + ColorReset)
	fmt.Println("------------------------------------")
	fmt.Println(ColorGreen + "🎯 Active User-Agents:" + ColorReset)
	fmt.Println("------------------------------------")
	if len(activeUserAgents) == 0 {
		fmt.Println(ColorYellow + "No active User-Agents found!" + ColorReset)
	} else {
		for _, ua := range activeUserAgents {
			fmt.Println(ColorGreen + ua + ColorReset)
			fmt.Println("------------------------------------")
		}
	}
	fmt.Println("------------------------------------")

	if len(failedUserAgents) == 0 {
		fmt.Println(ColorGreen + "🎉 All User-Agents are working correctly!" + ColorReset)
	} else {
		fmt.Printf(ColorRed+"❌ %d inactive User-Agent(s) found:\n\n"+ColorReset, len(failedUserAgents))
		for _, result := range failedUserAgents {
			fmt.Printf(ColorRed+"User-Agent: %s\n"+ColorReset, result.UserAgent)
			fmt.Printf(ColorRed+"Reason: %s\n"+ColorReset, result.Reason)
			fmt.Println("------------------------------------")
		}
	}

	fmt.Printf(ColorWhite+"\nSummary:\nTotal: %d  |  Active: %d  |  Inactive: %d  |  Time: %s\n"+ColorReset,
		total, len(activeUserAgents), len(failedUserAgents), elapsed.Round(time.Second).String())
}

func main() {
	clearScreen()
	fmt.Println(ColorGreen + "Welcome to User-Agent Checker!" + ColorReset)
	fmt.Println(ColorWhite + "==============================" + ColorReset)
	fmt.Println(ColorYellow + "Please choose an option:" + ColorReset)
	fmt.Println(ColorWhite + "1 - Use default User-Agents from user_agents.txt" + ColorReset)
	fmt.Println(ColorWhite + "2 - Enter your own User-Agents (comma separated)" + ColorReset)
	fmt.Print(ColorWhite + "Enter your choice (1 or 2): " + ColorReset)

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		agents, err := getUserAgentsFromFile("user_agents.txt")
		if err != nil {
			fmt.Printf(ColorRed+"Error reading user_agents.txt: %v\n"+ColorReset, err)
			return
		}
		concurrency, _ := chooseSpeedMenu()
		runCheckProcess(agents, concurrency)
	case "2":
		agents := getUserAgentsFromInput()
		concurrency, _ := chooseSpeedMenu()
		runCheckProcess(agents, concurrency)
	default:
		fmt.Println(ColorRed + "Invalid option. Exiting." + ColorReset)
	}
}

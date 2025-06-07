package main

import (
	"bufio"  
	"fmt"     
	"log"    
	"net/http" 
	"os"      
	"sync"   
)

const testURL = "https://httpbin.org/user-agent"


func checkUserAgent(ua string, resultsChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	client := &http.Client{}
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		resultsChan <- fmt.Sprintf("Error creating request for '%s': %v", ua, err)
		return
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		resultsChan <- fmt.Sprintf("❌ Unsuccessful: %s (Error: %v)", ua, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		resultsChan <- fmt.Sprintf("✅ Successful: %s", ua)
	} else {
		resultsChan <- fmt.Sprintf("❌ Unsuccessful: %s (Status code: %d)", ua, resp.StatusCode)
	}
}
func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please enter the name of the file containing the User-Agents as an argument..\n go run check.go user_agents.txt")
	}
	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Error opening file'%s': %v", fileName, err)
	}
	defer file.Close()
	var wg sync.WaitGroup
	resultsChan := make(chan string)
	scanner := bufio.NewScanner(file)
	fmt.Println("Starting to check User-Agents...")
	go func() {
		for result := range resultsChan {
			fmt.Println(result)
		}
	}()
	for scanner.Scan() {
		userAgent := scanner.Text()
		if userAgent != "" {
			wg.Add(1)
			go checkUserAgent(userAgent, resultsChan, &wg)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}
	wg.Wait()
	close(resultsChan)
	fmt.Println("\nReview completed.")
}

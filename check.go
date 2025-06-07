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
		resultsChan <- fmt.Sprintf("خطا در ساخت درخواست برای '%s': %v", ua, err)
		return
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		resultsChan <- fmt.Sprintf("❌ ناموفق: %s (خطا: %v)", ua, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		resultsChan <- fmt.Sprintf("✅ موفق: %s", ua)
	} else {
		resultsChan <- fmt.Sprintf("❌ ناموفق: %s (کد وضعیت: %d)", ua, resp.StatusCode)
	}
}
func main() {
	if len(os.Args) < 2 {
		log.Fatal("لطفاً نام فایل حاوی User-Agent ها را به عنوان آرگومان وارد کنید.\nمثال: go run main.go user_agents.txt")
	}
	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("خطا در باز کردن فایل '%s': %v", fileName, err)
	}
	defer file.Close()
	var wg sync.WaitGroup
	resultsChan := make(chan string)
	scanner := bufio.NewScanner(file)
	fmt.Println("شروع بررسی User-Agent ها...")
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
		log.Fatalf("خطا در خواندن فایل: %v", err)
	}
	wg.Wait()
	close(resultsChan)
	fmt.Println("\nبررسی تمام شد.")
}

package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

func getURLContent(url string, client *http.Client) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return content, err
}

func saveURLContentToFile(url string, filePath string, client *http.Client) error {
	content, err := getURLContent(url, client)
	if err != nil {
		Warning.Printf("failed to retrive %s content: %v\n", url, err)
		return err
	}
	err = ioutil.WriteFile(filePath, content, 0777)
	if err != nil {
		Warning.Printf("failed to write %s content to a file: %v\n", url, err)
		return err
	}
	Info.Printf("saved %7d bytes from %s to %s", len(content), url, filePath)
	return nil
}

func startWorker(workerID int, timeout time.Duration, taskCh <-chan *task, resCh chan<- error) {
	client := &http.Client{
		Timeout: timeout,
	}
	for {
		nextTask := <-taskCh
		resCh <- saveURLContentToFile(nextTask.url, nextTask.outFile, client)
		Info.Printf("worker %d finished processing %s", workerID, nextTask.url)
	}
}

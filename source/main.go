package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)
import "fmt"

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func initLogger(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func linesFromReader(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func readURLsFromFile(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return linesFromReader(f)
}

func getURLContent(url string, client http.Client) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return content, err
}

func saveURLContentToFile(url string, filePath string, ch chan <- string, client http.Client) {
	content, err := getURLContent(url, client)
	if err != nil {
		Warning.Println(err)
		ch <- fmt.Sprintf("WARNING: fetching %s failed", url)
		return
	}
	err = ioutil.WriteFile(filePath, content, 0777)
	if err != nil {
		Warning.Println(err)
		ch <- fmt.Sprintf("WARNING: failed to write %s content to a file", url)
		return
	}
	ch <- fmt.Sprintf("%7d bytes from %s", len(content), url)
}

func composeOutputFileName(index int, url string, folder string) (string) {
	return filepath.Join(folder, strconv.Itoa(index) + ".html")
}

func main() {
	logFile, err := os.OpenFile("log.txt", os.O_RDWR | os.O_CREATE | os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	defer logFile.Close()

	initLogger(ioutil.Discard, os.Stdout, logFile, os.Stderr)
	inputFilePtr := flag.String("input", "", "path to input file")
	outputDirPtr := flag.String("output", "./", "path to output directory")
	threadNumbPtr := flag.Int("threads", 0, "number of threads")
	timeoutNumbPtr := flag.Int("timeout", 60, "request timeout")

	flag.Parse()

	fmt.Println("input:", *inputFilePtr)
	fmt.Println("output:", *outputDirPtr)
	fmt.Println("threads:", *threadNumbPtr)
	fmt.Println("timeout:", *timeoutNumbPtr)
	fmt.Println("numCPU:", runtime.NumCPU())
	fmt.Println("default GOMAXPROCS:", runtime.GOMAXPROCS(*threadNumbPtr))
	fmt.Println("set GOMAXPROCS to:", runtime.GOMAXPROCS(0))

	urls, err := readURLsFromFile(*inputFilePtr)
	if err != nil {
		Error.Fatalln(err)
	}

	timeout := time.Duration(*timeoutNumbPtr) * time.Second
	client := http.Client{
		Timeout:timeout,
	}

	ch := make(chan string)
	for index, url := range urls {
		go saveURLContentToFile(url, composeOutputFileName(index, url, *outputDirPtr), ch, client)
	}

	for range urls {
		fmt.Println(<-ch)
	}
}

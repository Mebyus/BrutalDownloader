package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

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
	errorHandle io.Writer,
) {

	Trace = log.New(
		traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	Info = log.New(
		infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	Warning = log.New(
		warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	Error = log.New(
		errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)
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

func composeOutputFileName(index int, url string, folder string) string {
	return filepath.Join(folder, strconv.Itoa(index)+".html")
}

func main() {
	logFile, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Printf("error opening log file: %v", err)
	}
	defer logFile.Close()

	initLogger(
		ioutil.Discard,
		logFile,
		logFile,
		os.Stderr,
	)
	inputFilePtr := flag.String("input", "urls.txt", "path to input file")
	outputDirPtr := flag.String("output", "out", "path to output directory")
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
		Error.Fatalf("unable to retrieve list of urls: %v\n", err)
	}

	tasks := make([]*task, len(urls))
	for i, url := range urls {
		tasks[i] = &task{
			url:     url,
			outFile: composeOutputFileName(i, url, *outputDirPtr),
		}
	}
	timeout := time.Duration(*timeoutNumbPtr) * time.Second

	taskCh := make(chan *task, len(urls)-1)
	resCh := make(chan error, len(urls)-1)
	numberOfWorkers := 4
	for i := 1; i <= numberOfWorkers; i++ {
		go startWorker(i, timeout, taskCh, resCh)
	}
	Info.Printf("started %d workers", numberOfWorkers)
	for _, t := range tasks {
		taskCh <- t
	}

	errCounter := 0
	for range tasks {
		err = <-resCh
		if err != nil {
			errCounter++
		}
	}
	Info.Printf("%d out of %d downloaded successfully", len(tasks)-errCounter, len(tasks))
}

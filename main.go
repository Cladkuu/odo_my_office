package main

import (
	"context"
	"flag"
	"fmt"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if len(os.Args[1:]) == 0 {
		fmt.Println("no args passed")
		os.Exit(2)
	}

	var path string
	flag.StringVar(&path, "path", "", "file path")
	flag.Parse()
	if path == "" {
		fmt.Println("path argument is not passed")
		os.Exit(2)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("not valid path passed: %s. err: %s", path, err.Error())
		os.Exit(2)
	}

	file, err := os.Open(absPath)
	if err != nil {
		fmt.Printf("file with path: %s,- is not opened. err: %s", path, err.Error())
		os.Exit(127)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	generator := NewPathGenerator(file)
	urlParser := NewURLParser(workerPool.NewWorkerPool(3), generator.gen)

	requester := NewHTTPRequester(
		workerPool.NewWorkerPool(5),
		&http.Client{Timeout: time.Duration(5 * time.Second)},
		urlParser.Resp,
	)
	errManager := NewErrorManager(urlParser.error, requester.error)

	printer := NewStdOutPrinter(workerPool.NewWorkerPool(1), os.Stdout, requester.Resp, errManager.Write)

	pipe := workerPool.NewPipeline(generator, printer, urlParser, requester, printer, errManager)
	_ = pipe.Run(ctx)

}

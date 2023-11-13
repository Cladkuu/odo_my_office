package main

import (
	"context"
	"fmt"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	d := time.Duration(3 * time.Second)
	fmt.Println(d.String())

	if len(os.Args[1:]) == 0 {
		fmt.Println("no args passed")
		os.Exit(1) // TODO проверить
	}

	path := os.Args[1]

	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("not valid path passed: %s. err: %s", path, err.Error())
		os.Exit(1) // TODO проверить
	}

	file, err := os.Open(absPath)
	if err != nil {
		fmt.Printf("file with path: %s,- is nit opened. err: %s", path, err.Error())
		os.Exit(1) // TODO проверить
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error)

	generator := NewPathGenerator(file)
	urlParser := NewURLParser(workerPool.NewWorkerPool(3), generator.gen, errCh)

	requester := NewHTTPRequester(
		workerPool.NewWorkerPool(5),
		&http.Client{Timeout: time.Duration(30 * time.Second)},
		urlParser.Resp,
		errCh,
	)
	printer := NewStdOutPrinter(workerPool.NewWorkerPool(1), file, requester.Resp, errCh)

	pipe := workerPool.NewPipeline(generator, printer, urlParser, requester, printer)
	defer pipe.Close()

	err = pipe.Run(ctx)
	if err != nil {
		return
	}

}

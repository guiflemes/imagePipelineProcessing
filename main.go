package main

import "context"

type Result struct{}

func walkPath(ctx context.Context, root string) (<-chan string, <-chan error) {
	pathCh := make(chan string)
	errorCh := make(chan error)

	return pathCh, errorCh
}

func processImage(ctx context.Context, pathCh <-chan string) <-chan Result {
	results := make(chan Result)
	return results
}

func saveImage(results <-chan Result) {}

func SetUpPipeline(root string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pathCh, errorCh := walkPath(ctx, root)

	results := processImage(ctx, pathCh)
	saveImage(results)

	if err := <-errorCh; err != nil {
		return err
	}

	return nil

}

func main() {}

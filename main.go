package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

type Result struct{}

func getContentType(path string) (string, error) {

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	buffer := make([]byte, 512)

	_, err = file.Read(buffer)

	if err != nil {
		return "", err
	}

	return http.DetectContentType(buffer), nil
}

func walkPath(ctx context.Context, root string) (<-chan string, <-chan error) {
	pathCh := make(chan string)
	errorCh := make(chan error, 1)

	go func() {
		errorCh <- filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.Type().IsRegular() {
				return fmt.Errorf("%v is not a file", d)
			}

			contextType, err := getContentType(path)

			if err != nil {
				return err
			}

			if contextType != "image/jpeg" {
				return fmt.Errorf("invalid context type, expected image/jpeg, given %s", contextType)
			}

			select {
			case pathCh <- path:
			case <-ctx.Done():
			}

			return nil
		})
	}()

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

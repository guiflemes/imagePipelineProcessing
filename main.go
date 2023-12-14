package main

import (
	"fmt"
	"image"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/disintegration/imaging"
)

type Result struct {
	srcImagePath   string
	thumbnailImage *image.NRGBA
	err            error
}

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

func walkPath(done <-chan struct{}, root string) (<-chan string, <-chan error) {
	pathCh := make(chan string)
	errorCh := make(chan error, 1)

	go func() {
		defer close(pathCh)
		errorCh <- filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.Type().IsRegular() {
				return nil
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
				fmt.Println("sending path", path)
			case <-done:
				return fmt.Errorf("walk canceled")
			}

			return nil
		})
	}()

	return pathCh, errorCh
}

func processImage(done <-chan struct{}, pathCh <-chan string) <-chan Result {
	results := make(chan Result)
	var wg sync.WaitGroup

	thumbnail := func() {
		for path := range pathCh {
			fmt.Println("process", path)
			srcImage, err := imaging.Open(path)
			if err != nil {
				select {
				case results <- Result{srcImagePath: path, thumbnailImage: nil, err: err}:
				case <-done:
					return
				}
			}

			image := imaging.Thumbnail(srcImage, 100, 100, imaging.Lanczos)

			select {
			case results <- Result{srcImagePath: path, thumbnailImage: image, err: nil}:
			case <-done:
				return
			}
		}
	}

	//fan out
	const maxPool = 5
	wg.Add(maxPool)

	for i := 0; i < maxPool; i++ {
		go func() {
			thumbnail()
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func saveImage(results <-chan Result) error {
	for result := range results {
		if result.err != nil {
			return result.err
		}

		filename := filepath.Base(result.srcImagePath)
		dest := "thumbnail/" + filename

		if err := imaging.Save(result.thumbnailImage, dest); err != nil {
			return err
		}

		fmt.Println(result.srcImagePath, "->", dest)
	}

	return nil
}

func SetUpPipeline(root string) error {
	done := make(chan struct{})

	pathCh, errorCh := walkPath(done, root)

	results := processImage(done, pathCh)
	saveErr := saveImage(results)

	if saveErr != nil {
		return saveErr
	}

	if err := <-errorCh; err != nil {
		return err
	}

	return nil

}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("image directory path not informed")
	}
	start := time.Now()
	root := os.Args[1]
	err := SetUpPipeline(root)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Time taken %s\n", time.Since(start))
}

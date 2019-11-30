package main

import (
	"fmt"
	"github.com/avast/retry-go"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

// FIXME
const skipTest = true

func main() {
	if err := run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if skipTest {
		return nil
	}

	address := os.Getenv("ADDRESS")
	if address == "" {
		return fmt.Errorf("expected ADDRESS env var")
	}

	indexHtml, err := ioutil.ReadFile("index.html")
	if err != nil {
		return fmt.Errorf("cannot read index.html: %v", err)
	}

	if err := ioutil.WriteFile("www/index.html", indexHtml, 0777); err != nil {
		return fmt.Errorf("cannot write index.html file: %v", err)
	}

	err = retry.Do(func() error {
		resp, err := http.Get("http://" + address + "/index.html")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), "World") {
			return fmt.Errorf("expected index.html to contain 'World'; full: <<< %s >>>", string(b))
		}
		return nil
	}, retry.Attempts(5), retry.Delay(3*time.Second))
	if err != nil {
		return err
	}

	// Now, change the index.html file and see the changes mirrored
	// on the server
	b, err := ioutil.ReadFile("www/index.html")
	if err != nil {
		return err
	}
	b = []byte(strings.Replace(string(b), "World", "War", 1))
	if err := ioutil.WriteFile("www/index.html", b, 0777); err != nil {
		return err
	}

	err = retry.Do(func() error {
		resp, err := http.Get("http://" + address + "/index.html")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), "World") {
			return fmt.Errorf("expected index.html to contain 'War' after file sync; full: <<< %s >>>", string(b))
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

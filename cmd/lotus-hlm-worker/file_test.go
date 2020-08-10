package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestTravelFile(t *testing.T) {
	if err := os.MkdirAll("./tmp/tmp/tmp", 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll("./tmp"); err != nil {
			t.Fatal(err)
		}
	}()

	if err := ioutil.WriteFile("./tmp/1", []byte{}, 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile("./tmp/tmp/1", []byte{}, 0666); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile("./tmp/tmp/tmp/1", []byte{}, 0666); err != nil {
		t.Fatal(err)
	}

	_, result, err := travelFile("./tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatal(result)
	}
}

func TestCopyFile(t *testing.T) {
	if err := os.MkdirAll("./tmp", 0666); err != nil {
		t.Fatal(err)
	}
	// defer os.RemoveAll("./tmp")
	ctx := context.Background()
	if err := CopyFile(ctx, "/usr/local/go", "./tmp"); err != nil {
		t.Fatal(err)
	}

	fmt.Println("do copy again for testing")

	// do it again
	if err := CopyFile(ctx, "/usr/local/go", "./tmp"); err != nil {
		t.Fatal(err)
	}
	_, fromResult, err := travelFile("/usr/local/go")
	if err != nil {
		t.Fatal(err)
	}
	_, toResult, err := travelFile("./tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(fromResult) != len(toResult) {
		t.Fatal(len(fromResult), len(toResult))
	}
	// todo: make the file checksum
}

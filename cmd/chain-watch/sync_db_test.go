package main

import "testing"

func TestCurHeight(t *testing.T) {
	InitDB("./")
	height, err := GetCurHeight()
	if err != nil {
		t.Fatal(err)
	}
	if err := AddCurHeight(height + 1); err != nil {
		t.Fatal(err)
	}
	nextHeight, err := GetCurHeight()
	if err != nil {
		t.Fatal(err)
	}
	if nextHeight != height+1 {
		t.Fatal(nextHeight, height+1)
	}
}

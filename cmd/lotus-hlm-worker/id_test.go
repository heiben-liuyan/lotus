package main

import "testing"

func TestGetWorkerID(t *testing.T) {
	repo := "/tmp"
	id := GetWorkerID(repo)
	id1 := GetWorkerID(repo)
	if len(id) == 0 {
		t.Fatal("No id found")
	}
	if id != id1 {
		t.Fatalf("expect:%s,but:%s", id, id1)
	}
}

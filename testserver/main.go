package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from testserver at %s\n", time.Now().Format(time.RFC3339))
	})
	fmt.Println("Test server listening on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatalf("server exited: %v", err)
	}
	fmt.Println("Hi")
}

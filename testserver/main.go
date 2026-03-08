package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from testserver at %s\n", time.Now().Format(time.RFC3339))
	})
	fmt.Println("Test server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

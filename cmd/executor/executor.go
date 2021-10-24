package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/grussorusso/serverledge/pkg/executor"
)

func main() {
	http.HandleFunc("/invoke", executor.InvokeHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", executor.DEFAULT_EXECUTOR_PORT), nil))
}

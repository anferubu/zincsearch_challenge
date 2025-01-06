package main

import (
	"net/http"

	api "enron-email-search/api/src"
)

func main() {
	r := api.SetupRouter()
	http.ListenAndServe(":3000", r)
}

// Command web serves the server-rendered CRM page.
package main

import (
	"log"
	"net/http"

	"github.com/mangrove-ide/example-golang/frontend/web"
)

func main() {
	apiURL := web.APIURL()
	port := web.Port()
	handler, err := web.NewHandler(web.NewAPIClient(apiURL))
	if err != nil {
		log.Fatalf("API_URL %q is not valid: %v", apiURL, err)
	}
	log.Printf("CRM web app listening on http://localhost:%s (API: %s)", port, apiURL)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

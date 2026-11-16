// Command api serves the CRM REST API.
package main

import (
	"log"
	"net/http"

	"github.com/mangrove-ide/example-golang/backend/crm"
)

func main() {
	databasePath := crm.DatabasePath()
	port := crm.Port()
	store, err := crm.OpenStore(databasePath)
	if err != nil {
		log.Fatalf("Could not open the database at %s: %v", databasePath, err)
	}
	log.Printf("CRM API listening on http://localhost:%s (database: %s)", port, databasePath)
	log.Fatal(http.ListenAndServe(":"+port, crm.NewHandler(store)))
}

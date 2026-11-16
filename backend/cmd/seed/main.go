// Command seed adds demo customers that are not in the database yet.
package main

import (
	"log"

	"github.com/mangrove-ide/example-golang/backend/crm"
)

var demoCustomers = []crm.Customer{
	{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"},
	{FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com"},
	{FirstName: "Alan", LastName: "Turing", Email: "alan@example.com"},
	{FirstName: "Katherine", LastName: "Johnson", Email: "katherine@example.com"},
}

func main() {
	databasePath := crm.DatabasePath()
	store, err := crm.OpenStore(databasePath)
	if err != nil {
		log.Fatalf("Could not open the database at %s: %v", databasePath, err)
	}
	defer store.Close()
	seededCount := 0
	for _, customer := range demoCustomers {
		hasEmail, err := store.HasEmail(customer.Email)
		if err != nil {
			log.Fatal(err)
		}
		if !hasEmail {
			if _, err := store.Create(customer.FirstName, customer.LastName, customer.Email); err != nil {
				log.Fatal(err)
			}
			seededCount++
		}
	}
	log.Printf("Seeded %d customers into %s", seededCount, databasePath)
}

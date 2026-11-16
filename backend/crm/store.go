package crm

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InMemoryPath opens a database that lives only as long as the store.
const InMemoryPath = ":memory:"

const customerColumns = "id, first_name, last_name, email, created_at"

const createCustomersTable = `
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE COLLATE NOCASE,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`

type Customer struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
}

// CustomerStore persists customers in SQLite.
type CustomerStore struct {
	database *sql.DB
}

func OpenStore(path string) (*CustomerStore, error) {
	if path != InMemoryPath {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// A single connection serializes access like the other ports do, and keeps a ":memory:" database
	// from being a separate, empty database per pooled connection.
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(createCustomersTable); err != nil {
		database.Close()
		return nil, err
	}
	return &CustomerStore{database: database}, nil
}

func (store *CustomerStore) Close() error {
	return store.database.Close()
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanCustomer(row rowScanner) (Customer, error) {
	var customer Customer
	err := row.Scan(&customer.ID, &customer.FirstName, &customer.LastName, &customer.Email, &customer.CreatedAt)
	return customer, err
}

// List returns every customer ordered by last name, then first name.
func (store *CustomerStore) List() ([]Customer, error) {
	rows, err := store.database.Query("SELECT " + customerColumns + " FROM customers ORDER BY last_name, first_name, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	customers := []Customer{}
	for rows.Next() {
		customer, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, rows.Err()
}

func (store *CustomerStore) Get(customerID int64) (Customer, error) {
	return scanCustomer(store.database.QueryRow("SELECT "+customerColumns+" FROM customers WHERE id = ?", customerID))
}

// HasEmail matches case-insensitively, through the column's NOCASE collation.
func (store *CustomerStore) HasEmail(email string) (bool, error) {
	var customerID int64
	err := store.database.QueryRow("SELECT id FROM customers WHERE email = ?", email).Scan(&customerID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (store *CustomerStore) Create(firstName string, lastName string, email string) (Customer, error) {
	result, err := store.database.Exec("INSERT INTO customers (first_name, last_name, email) VALUES (?, ?, ?)", firstName, lastName, email)
	if err != nil {
		return Customer{}, err
	}
	customerID, err := result.LastInsertId()
	if err != nil {
		return Customer{}, err
	}
	return store.Get(customerID)
}

// Remove reports whether a customer was deleted.
func (store *CustomerStore) Remove(customerID int64) (bool, error) {
	result, err := store.database.Exec("DELETE FROM customers WHERE id = ?", customerID)
	if err != nil {
		return false, err
	}
	removedCount, err := result.RowsAffected()
	return removedCount > 0, err
}

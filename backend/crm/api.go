package crm

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	emailPattern   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	digitsPattern  = regexp.MustCompile(`^\d+$`)
	requiredFields = []string{"firstName", "lastName", "email"}
)

type errorBody struct {
	Error string `json:"error"`
}

type customerAPI struct {
	store *CustomerStore
}

// NewHandler serves the customers REST API under /api/customers.
func NewHandler(store *CustomerStore) http.Handler {
	api := &customerAPI{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/customers", api.list)
	mux.HandleFunc("POST /api/customers", api.create)
	mux.HandleFunc("DELETE /api/customers/{customerId}", api.remove)
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writeError(writer, http.StatusNotFound, "Not found")
	})
	return mux
}

// parseCustomer returns the trimmed string fields, or a message explaining why the body is invalid.
func parseCustomer(body any) (map[string]string, string) {
	source, _ := body.(map[string]any)
	values := map[string]string{}
	missingFields := []string{}
	for _, field := range requiredFields {
		text, _ := source[field].(string)
		values[field] = strings.TrimSpace(text)
		if values[field] == "" {
			missingFields = append(missingFields, field)
		}
	}
	if len(missingFields) > 0 {
		return nil, fmt.Sprintf("Missing required fields: %s", strings.Join(missingFields, ", "))
	}
	if !emailPattern.MatchString(values["email"]) {
		return nil, "Email is not valid"
	}
	return values, ""
}

func (api *customerAPI) list(writer http.ResponseWriter, request *http.Request) {
	customers, err := api.store.List()
	if err != nil {
		writeInternalError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, customers)
}

func (api *customerAPI) create(writer http.ResponseWriter, request *http.Request) {
	rawBody, err := io.ReadAll(request.Body)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "Body could not be read")
		return
	}
	var body any = map[string]any{}
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &body); err != nil {
			writeError(writer, http.StatusBadRequest, "Body is not valid JSON")
			return
		}
	}
	customer, errorMessage := parseCustomer(body)
	if errorMessage != "" {
		writeError(writer, http.StatusBadRequest, errorMessage)
		return
	}
	hasEmail, err := api.store.HasEmail(customer["email"])
	if err != nil {
		writeInternalError(writer, err)
		return
	}
	if hasEmail {
		writeError(writer, http.StatusConflict, "A customer with this email already exists")
		return
	}
	created, err := api.store.Create(customer["firstName"], customer["lastName"], customer["email"])
	if err != nil {
		writeInternalError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, created)
}

func (api *customerAPI) remove(writer http.ResponseWriter, request *http.Request) {
	rawCustomerID := request.PathValue("customerId")
	if !digitsPattern.MatchString(rawCustomerID) {
		writeError(writer, http.StatusNotFound, "Customer not found")
		return
	}
	customerID, err := strconv.ParseInt(rawCustomerID, 10, 64)
	if err != nil {
		writeError(writer, http.StatusNotFound, "Customer not found")
		return
	}
	isRemoved, err := api.store.Remove(customerID)
	if err != nil {
		writeInternalError(writer, err)
		return
	}
	if !isRemoved {
		writeError(writer, http.StatusNotFound, "Customer not found")
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(body); err != nil {
		log.Printf("Could not write the response: %v", err)
	}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, errorBody{Error: message})
}

func writeInternalError(writer http.ResponseWriter, err error) {
	log.Printf("Request failed: %v", err)
	writeError(writer, http.StatusInternalServerError, "Internal server error")
}

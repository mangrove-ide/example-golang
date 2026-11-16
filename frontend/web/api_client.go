package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const customersPath = "/api/customers"

type Customer struct {
	ID        int64  `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type CustomerForm struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// APIError carries the API's own error message, shown to the user as is.
type APIError struct {
	Message string
}

func (apiError *APIError) Error() string {
	return apiError.Message
}

// CustomerAPI is what the web app needs from the REST API; tests stand in a fake for it.
type CustomerAPI interface {
	BaseURL() string
	ListCustomers() ([]Customer, error)
	CreateCustomer(form CustomerForm) (Customer, error)
	DeleteCustomer(customerID int64) error
}

// APIClient calls the CRM REST API over HTTP.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{baseURL: strings.TrimRight(baseURL, "/"), httpClient: http.DefaultClient}
}

func (client *APIClient) BaseURL() string {
	return client.baseURL
}

func (client *APIClient) ListCustomers() ([]Customer, error) {
	customers := []Customer{}
	err := client.request(http.MethodGet, customersPath, nil, &customers)
	return customers, err
}

func (client *APIClient) CreateCustomer(form CustomerForm) (Customer, error) {
	var created Customer
	err := client.request(http.MethodPost, customersPath, form, &created)
	return created, err
}

func (client *APIClient) DeleteCustomer(customerID int64) error {
	return client.request(http.MethodDelete, customersPath+"/"+strconv.FormatInt(customerID, 10), nil, nil)
}

func (client *APIClient) request(method string, path string, body any, result any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequest(method, client.baseURL+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return &APIError{Message: fmt.Sprintf("Could not reach the API at %s", client.baseURL)}
	}
	defer response.Body.Close()
	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		return &APIError{Message: fmt.Sprintf("Could not reach the API at %s", client.baseURL)}
	}
	if response.StatusCode >= http.StatusBadRequest {
		var errorBody struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(rawBody, &errorBody) != nil || errorBody.Error == "" {
			return &APIError{Message: fmt.Sprintf("Request failed with status %d", response.StatusCode)}
		}
		return &APIError{Message: errorBody.Error}
	}
	if result == nil || len(rawBody) == 0 {
		return nil
	}
	return json.Unmarshal(rawBody, result)
}

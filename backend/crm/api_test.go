package crm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) string {
	t.Helper()
	store, err := OpenStore(InMemoryPath)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewHandler(store))
	t.Cleanup(func() {
		server.Close()
		store.Close()
	})
	return server.URL + "/api/customers"
}

func send(t *testing.T, method string, url string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, rawBody
}

func decode[T any](t *testing.T, rawBody []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rawBody, &value); err != nil {
		t.Fatalf("could not decode %q: %v", rawBody, err)
	}
	return value
}

func listCustomers(t *testing.T, baseURL string) []Customer {
	t.Helper()
	status, rawBody := send(t, http.MethodGet, baseURL, nil)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want 200", status)
	}
	return decode[[]Customer](t, rawBody)
}

func createCustomer(t *testing.T, baseURL string, firstName string, lastName string, email string) (int, []byte) {
	t.Helper()
	return send(t, http.MethodPost, baseURL, map[string]string{"firstName": firstName, "lastName": lastName, "email": email})
}

func TestCreatesListsAndDeletesACustomer(t *testing.T) {
	baseURL := newTestServer(t)

	status, rawBody := createCustomer(t, baseURL, " Ada ", "Lovelace", "ada@example.com")
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", status)
	}
	created := decode[Customer](t, rawBody)
	if created.FirstName != "Ada" || created.LastName != "Lovelace" || created.Email != "ada@example.com" {
		t.Errorf("created = %+v", created)
	}
	if created.ID == 0 || created.CreatedAt == "" {
		t.Errorf("created is missing id or createdAt: %+v", created)
	}

	if customers := listCustomers(t, baseURL); len(customers) != 1 {
		t.Errorf("listed %d customers, want 1", len(customers))
	}

	status, rawBody = send(t, http.MethodDelete, baseURL+"/"+strconv.FormatInt(created.ID, 10), nil)
	if status != http.StatusNoContent || len(rawBody) != 0 {
		t.Errorf("delete = %d %q, want 204 with no body", status, rawBody)
	}
	if customers := listCustomers(t, baseURL); len(customers) != 0 {
		t.Errorf("listed %d customers after delete, want 0", len(customers))
	}
}

func TestListsCustomersByLastThenFirstName(t *testing.T) {
	baseURL := newTestServer(t)
	createCustomer(t, baseURL, "Grace", "Hopper", "grace@example.com")
	createCustomer(t, baseURL, "Ada", "Lovelace", "ada@example.com")
	createCustomer(t, baseURL, "Adam", "Hopper", "adam@example.com")

	firstNames := []string{}
	for _, customer := range listCustomers(t, baseURL) {
		firstNames = append(firstNames, customer.FirstName)
	}
	if got := strings.Join(firstNames, ","); got != "Adam,Grace,Ada" {
		t.Errorf("order = %s, want Adam,Grace,Ada", got)
	}
}

func TestRejectsMissingFieldsAndInvalidEmails(t *testing.T) {
	baseURL := newTestServer(t)

	status, rawBody := send(t, http.MethodPost, baseURL, map[string]string{"firstName": "Ada"})
	if status != http.StatusBadRequest {
		t.Errorf("missing fields status = %d, want 400", status)
	}
	if message := decode[errorBody](t, rawBody).Error; !strings.Contains(message, "lastName, email") {
		t.Errorf("missing fields error = %q", message)
	}

	if status, _ := createCustomer(t, baseURL, "Ada", "Lovelace", "not-an-email"); status != http.StatusBadRequest {
		t.Errorf("invalid email status = %d, want 400", status)
	}

	status, _ = send(t, http.MethodPost, baseURL, "not an object")
	if status != http.StatusBadRequest {
		t.Errorf("non-object body status = %d, want 400", status)
	}
}

func TestRejectsDuplicateEmailsRegardlessOfCase(t *testing.T) {
	baseURL := newTestServer(t)
	createCustomer(t, baseURL, "Ada", "Lovelace", "ada@example.com")
	if status, _ := createCustomer(t, baseURL, "Ada", "L", "ADA@example.com"); status != http.StatusConflict {
		t.Errorf("duplicate status = %d, want 409", status)
	}
}

func TestReturns404WhenDeletingAnUnknownCustomer(t *testing.T) {
	baseURL := newTestServer(t)
	for _, customerID := range []string{"999", "abc"} {
		if status, _ := send(t, http.MethodDelete, baseURL+"/"+customerID, nil); status != http.StatusNotFound {
			t.Errorf("delete %s status = %d, want 404", customerID, status)
		}
	}
}


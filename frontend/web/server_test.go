package web

import (
	"cmp"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
)

var (
	ada   = Customer{ID: 1, FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	grace = CustomerForm{FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com"}
)

// fakeAPI stands in for the REST API, the way the React tests stub fetch.
type fakeAPI struct {
	mutex              sync.Mutex
	customers          []Customer
	createErrorMessage string
	deletedCustomerIDs []int64
}

func (api *fakeAPI) BaseURL() string {
	return "http://127.0.0.1:9"
}

func (api *fakeAPI) ListCustomers() ([]Customer, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	sorted := slices.Clone(api.customers)
	slices.SortFunc(sorted, func(left Customer, right Customer) int {
		return cmp.Or(cmp.Compare(left.LastName, right.LastName), cmp.Compare(left.FirstName, right.FirstName))
	})
	return sorted, nil
}

func (api *fakeAPI) CreateCustomer(form CustomerForm) (Customer, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	if api.createErrorMessage != "" {
		return Customer{}, &APIError{Message: api.createErrorMessage}
	}
	created := Customer{ID: int64(len(api.customers) + 1), FirstName: form.FirstName, LastName: form.LastName, Email: form.Email}
	api.customers = append(api.customers, created)
	return created, nil
}

func (api *fakeAPI) DeleteCustomer(customerID int64) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.deletedCustomerIDs = append(api.deletedCustomerIDs, customerID)
	api.customers = slices.DeleteFunc(api.customers, func(customer Customer) bool {
		return customer.ID == customerID
	})
	return nil
}

func newTestServer(t *testing.T, api *fakeAPI) string {
	t.Helper()
	handler, err := NewHandler(api)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

// fetchPage GETs the page, or POSTs the form when one is given; redirects are followed.
func fetchPage(t *testing.T, pageURL string, form url.Values) (int, string) {
	t.Helper()
	var response *http.Response
	var err error
	if form == nil {
		response, err = http.Get(pageURL)
	} else {
		response, err = http.PostForm(pageURL, form)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	rawBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(rawBody)
}

func assertContains(t *testing.T, html string, expected string) {
	t.Helper()
	if !strings.Contains(html, expected) {
		t.Errorf("page does not contain %q", expected)
	}
}

func TestListsAddsAndRemovesCustomers(t *testing.T) {
	api := &fakeAPI{customers: []Customer{ada}}
	baseURL := newTestServer(t, api)

	status, html := fetchPage(t, baseURL+"/", nil)
	if status != http.StatusOK {
		t.Fatalf("home status = %d, want 200", status)
	}
	assertContains(t, html, "Lovelace")
	assertContains(t, html, "1 total")

	status, html = fetchPage(t, baseURL+"/customers", url.Values{"firstName": {grace.FirstName}, "lastName": {grace.LastName}, "email": {grace.Email}})
	if status != http.StatusOK {
		t.Fatalf("add status = %d, want 200", status)
	}
	assertContains(t, html, "2 total")
	if strings.Index(html, "Hopper") > strings.Index(html, "Lovelace") {
		t.Errorf("Hopper should be listed before Lovelace")
	}
	assertContains(t, html, `aria-label="Remove Ada Lovelace"`)
	assertContains(t, html, `data-confirm="Remove Ada Lovelace?"`)

	status, html = fetchPage(t, baseURL+"/customers/1/delete", url.Values{})
	if status != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", status)
	}
	if strings.Contains(html, "Lovelace") {
		t.Errorf("removed customer is still listed")
	}
	if !slices.Equal(api.deletedCustomerIDs, []int64{1}) {
		t.Errorf("deleted ids = %v, want [1]", api.deletedCustomerIDs)
	}
}

func TestShowsEmptyState(t *testing.T) {
	baseURL := newTestServer(t, &fakeAPI{})
	_, html := fetchPage(t, baseURL+"/", nil)
	assertContains(t, html, "No customers yet")
}

func TestShowsTheAPIErrorWhenAddingFailsAndKeepsTheInput(t *testing.T) {
	baseURL := newTestServer(t, &fakeAPI{createErrorMessage: "A customer with this email already exists"})
	status, html := fetchPage(t, baseURL+"/customers", url.Values{"firstName": {"Ada"}, "lastName": {"Lovelace"}, "email": {"ada@example.com"}})
	if status != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", status)
	}
	assertContains(t, html, `role="alert">A customer with this email already exists`)
	assertContains(t, html, `value="ada@example.com"`)
}

func TestEscapesCustomerFields(t *testing.T) {
	baseURL := newTestServer(t, &fakeAPI{customers: []Customer{{ID: 1, FirstName: "<b>Ada</b>", LastName: "Lovelace", Email: "ada@example.com"}}})
	_, html := fetchPage(t, baseURL+"/", nil)
	assertContains(t, html, "&lt;b&gt;Ada&lt;/b&gt;")
	if strings.Contains(html, "<b>Ada</b>") {
		t.Errorf("customer fields are not escaped")
	}
}

func TestServesTheStylesheet(t *testing.T) {
	baseURL := newTestServer(t, &fakeAPI{})
	status, css := fetchPage(t, baseURL+"/static/app.css", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	assertContains(t, css, ".customer-table")
}

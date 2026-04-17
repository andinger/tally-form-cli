package tally

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateForm(t *testing.T) {
	var gotReq CreateFormRequest
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/forms" {
			t.Errorf("Path = %q, want /forms", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotReq)
		json.NewEncoder(w).Encode(TallyForm{ID: "new123", Name: "Test"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	form, err := client.CreateForm(&CreateFormRequest{
		WorkspaceID: "ws1",
		Name:        "Test Form",
		Blocks:      []TallyBlock{{UUID: "u1", Type: "TEXT"}},
	})
	if err != nil {
		t.Fatalf("CreateForm error: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("Auth = %q", gotAuth)
	}
	if gotReq.Name != "Test Form" {
		t.Errorf("Name = %q", gotReq.Name)
	}
	if form.ID != "new123" {
		t.Errorf("Form ID = %q", form.ID)
	}
}

func TestUpdateForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/forms/abc123" {
			t.Errorf("Path = %q, want /forms/abc123", r.URL.Path)
		}
		json.NewEncoder(w).Encode(TallyForm{ID: "abc123"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	form, err := client.UpdateForm("abc123", &UpdateFormRequest{Blocks: []TallyBlock{}})
	if err != nil {
		t.Fatalf("UpdateForm error: %v", err)
	}
	if form.ID != "abc123" {
		t.Errorf("Form ID = %q", form.ID)
	}
}

func TestGetForm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		json.NewEncoder(w).Encode(TallyForm{
			ID:   "xyz",
			Name: "Fetched",
			Blocks: []TallyBlock{
				{UUID: "b1", Type: "TEXT"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	form, err := client.GetForm("xyz")
	if err != nil {
		t.Fatalf("GetForm error: %v", err)
	}
	if form.Name != "Fetched" {
		t.Errorf("Name = %q", form.Name)
	}
	if len(form.Blocks) != 1 {
		t.Errorf("Blocks = %d", len(form.Blocks))
	}
}

func TestGetSubmissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(SubmissionsResponse{
			Questions: []SubmissionQuestion{{ID: "q1", Name: "Test Q"}},
			Submissions: []Submission{
				{ID: "s1", SubmittedAt: "2026-03-28", Responses: []SubmissionResponse{
					{QuestionID: "q1", FormattedAnswer: "Answer 1"},
				}},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	subs, err := client.GetSubmissions("form1")
	if err != nil {
		t.Fatalf("GetSubmissions error: %v", err)
	}
	if len(subs.Submissions) != 1 {
		t.Errorf("Submissions = %d", len(subs.Submissions))
	}
}

func TestGetFormInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.GetForm("xyz")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestCreateFormInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.CreateForm(&CreateFormRequest{})
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

func TestUpdateFormInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.UpdateForm("id", &UpdateFormRequest{})
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

func TestGetSubmissionsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.GetSubmissions("id")
	if err == nil {
		t.Fatal("Expected error for invalid JSON response")
	}
}

func TestDeleteForm(t *testing.T) {
	var gotMethod, gotPath, gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	if err := client.DeleteForm("abc123"); err != nil {
		t.Fatalf("DeleteForm error: %v", err)
	}

	if gotMethod != "DELETE" {
		t.Errorf("Method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/forms/abc123" {
		t.Errorf("Path = %q, want /forms/abc123", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Auth = %q", gotAuth)
	}
}

func TestDeleteFormNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Form was not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	err := client.DeleteForm("missing")
	if err == nil {
		t.Fatal("Expected error for 404")
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token")
	_, err := client.GetForm("xyz")
	if err == nil {
		t.Fatal("Expected error for 403")
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nexnews/intelligence"
)

func TestAnalystPageIsRegistered(t *testing.T) {
	repo,err:=intelligence.NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil{t.Fatal(err)}
	handler:=newHandler(repo)
	request:=httptest.NewRequest(http.MethodGet,"/analyst",nil)
	response:=httptest.NewRecorder()
	handler.ServeHTTP(response,request)
	if response.Code!=http.StatusOK{t.Fatalf("expected HTTP 200, got %d",response.Code)}
	if !strings.Contains(response.Body.String(),"NexNews · Analizci"){t.Fatal("analyst page content missing")}
}

func TestLiveResearchRejectsMissingQuestion(t *testing.T) {
	repo,err:=intelligence.NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil{t.Fatal(err)}
	handler:=newHandler(repo)
	request:=httptest.NewRequest(http.MethodPost,"/api/live-research",strings.NewReader(`{"query":"   "}`))
	request.Header.Set("Content-Type","application/json")
	response:=httptest.NewRecorder()
	handler.ServeHTTP(response,request)
	if response.Code!=http.StatusBadRequest{t.Fatalf("expected HTTP 400, got %d: %s",response.Code,response.Body.String())}
	if !strings.Contains(response.Body.String(),"Araştırmak istediğin konuyu"){t.Fatalf("unexpected error response: %s",response.Body.String())}
}

func TestLiveResearchRejectsGET(t *testing.T) {
	repo,err:=intelligence.NewJSONRepository(filepath.Join(t.TempDir(),"intelligence.json"))
	if err!=nil{t.Fatal(err)}
	handler:=newHandler(repo)
	request:=httptest.NewRequest(http.MethodGet,"/api/live-research",nil)
	response:=httptest.NewRecorder()
	handler.ServeHTTP(response,request)
	if response.Code!=http.StatusMethodNotAllowed{t.Fatalf("expected HTTP 405, got %d",response.Code)}
}

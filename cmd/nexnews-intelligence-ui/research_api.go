package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"nexnews/intelligence"
)

const maxResearchRequestBytes = 1 << 20

func registerResearchRoutes(mux *http.ServeMux, repo *intelligence.JSONRepository) {
	mux.HandleFunc("/research-workspace", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/research-workspace" { http.NotFound(w, r); return }
		if !allowGET(w, r) { return }
		setSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(researchWorkspaceHTML))
	})

	mux.HandleFunc("/api/research-cases", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		switch r.Method {
		case http.MethodGet:
			items, err := repo.ListResearchCases()
			if err != nil { writeResearchError(w, http.StatusInternalServerError, "Research veritabanı okunamadı."); return }
			writeResearchJSON(w, http.StatusOK, map[string]any{"items": items})
		case http.MethodPost:
			var input intelligence.CreateResearchCaseInput
			if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
			item, err := repo.CreateResearchCase(input)
			if err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
			writeResearchJSON(w, http.StatusCreated, item)
		default:
			researchMethodNotAllowed(w, "GET, POST")
		}
	})

	mux.HandleFunc("/api/research-case", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if !allowGET(w, r) { return }
		id := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if id == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		item, err := repo.GetResearchCase(id)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") { writeResearchError(w, http.StatusNotFound, "Research case not found."); return }
			writeResearchError(w, http.StatusInternalServerError, "Research case could not be read.")
			return
		}
		writeResearchJSON(w, http.StatusOK, item)
	})

	mux.HandleFunc("/api/research-import", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if caseID == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		item, stats, err := repo.SyncResearchCaseFromNexNews(caseID)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusOK, map[string]any{"case":item,"stats":stats})
	})

	mux.HandleFunc("/api/research-analysis", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if !allowGET(w, r) { return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if caseID == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		item, err := repo.GetResearchCase(caseID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") { writeResearchError(w, http.StatusNotFound, "Research case not found."); return }
			writeResearchError(w, http.StatusInternalServerError, "Research case could not be read.")
			return
		}
		writeResearchJSON(w, http.StatusOK, intelligence.AnalyzeResearchCase(item))
	})

	mux.HandleFunc("/api/research-hypothesis", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if caseID == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		var input struct { Statement string `json:"statement"` }
		if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
		item, err := repo.AddResearchHypothesis(caseID, input.Statement)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("/api/research-evidence", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if caseID == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		var input intelligence.AddResearchEvidenceInput
		if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
		item, err := repo.AddResearchEvidence(caseID, input)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("/api/research-question", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		if caseID == "" { writeResearchError(w, http.StatusBadRequest, "caseId is required"); return }
		var input struct { Question string `json:"question"` }
		if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
		item, err := repo.AddResearchQuestion(caseID, input.Question)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("/api/research-question-status", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		questionID := strings.TrimSpace(r.URL.Query().Get("questionId"))
		if caseID == "" || questionID == "" { writeResearchError(w, http.StatusBadRequest, "caseId and questionId are required"); return }
		var input struct {
			Status string `json:"status"`
			Note string `json:"note"`
		}
		if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
		item, err := repo.SetResearchQuestionStatus(caseID, questionID, input.Status, input.Note)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusOK, item)
	})

	mux.HandleFunc("/api/research-hypothesis-status", func(w http.ResponseWriter, r *http.Request) {
		setResearchHeaders(w)
		if r.Method != http.MethodPost { researchMethodNotAllowed(w, "POST"); return }
		caseID := strings.TrimSpace(r.URL.Query().Get("caseId"))
		hypothesisID := strings.TrimSpace(r.URL.Query().Get("hypothesisId"))
		if caseID == "" || hypothesisID == "" { writeResearchError(w, http.StatusBadRequest, "caseId and hypothesisId are required"); return }
		var input struct {
			Status intelligence.ResearchHypothesisStatus `json:"status"`
			Note string `json:"note"`
		}
		if err := decodeResearchJSON(w, r, &input); err != nil { writeResearchError(w, http.StatusBadRequest, err.Error()); return }
		item, err := repo.SetResearchHypothesisStatus(caseID, hypothesisID, input.Status, input.Note)
		if err != nil { researchOperationError(w, err); return }
		writeResearchJSON(w, http.StatusOK, item)
	})
}

func setResearchHeaders(w http.ResponseWriter) {
	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
}

func decodeResearchJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxResearchRequestBytes)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) { return errors.New("JSON body is required") }
		return errors.New("JSON body is invalid or exceeds the 1 MiB limit")
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeResearchJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeResearchError(w http.ResponseWriter, status int, message string) {
	writeResearchJSON(w, status, map[string]string{"error": message})
}

func researchMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeResearchError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func researchOperationError(w http.ResponseWriter, err error) {
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		writeResearchError(w, http.StatusNotFound, err.Error())
		return
	}
	writeResearchError(w, http.StatusBadRequest, err.Error())
}

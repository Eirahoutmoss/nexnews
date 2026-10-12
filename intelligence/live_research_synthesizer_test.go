package intelligence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroundLiveSynthesisDropsUnknownSourceIDs(t *testing.T) {
	article:=LiveResearchArticle{ID:"source-1",Title:"Kaynak başlığı",Publisher:"Örnek",URL:"https://example.com/story"}
	draft:=liveSynthesisDraft{
		ExecutiveSummary:"Kaynakta bildirilen olay için bağımsız doğrulama kapsamı sınırlı.",
		OverallConfidence:"high",
		KeyFindings:[]liveSynthesisInsightDraft{
			{Title:"Kaynaklı bulgu",Assessment:"Kaynak bu gelişmeyi bildiriyor.",Confidence:"medium",ArticleIDs:[]string{"source-1","fabricated-id"}},
			{Title:"Kaynağı olmayan bulgu",Assessment:"Bu iddia girdide yok.",Confidence:"high",ArticleIDs:[]string{"invented-id"}},
		},
	}
	got,err:=groundLiveSynthesis(draft,[]LiveResearchArticle{article},"test-model")
	if err!=nil{t.Fatal(err)}
	if len(got.KeyFindings)!=1{t.Fatalf("expected only grounded finding, got %d",len(got.KeyFindings))}
	if len(got.KeyFindings[0].Sources)!=1{t.Fatalf("expected one validated citation, got %d",len(got.KeyFindings[0].Sources))}
	if got.KeyFindings[0].Sources[0].URL!=article.URL{t.Fatalf("unexpected citation URL: %s",got.KeyFindings[0].Sources[0].URL)}
	if got.OverallConfidence!="high"{t.Fatalf("unexpected normalized confidence: %s",got.OverallConfidence)}
}

func TestOpenAICompatibleSynthesizerUsesProvidedSourcesOnly(t *testing.T) {
	article:=LiveResearchArticle{ID:"source-1",Title:"Enerji altyapısına ilişkin haber",Publisher:"Örnek Yayın",Domain:"example.com",URL:"https://example.com/story"}
	draft:=liveSynthesisDraft{
		ExecutiveSummary:"Mevcut sonuçlar tek bir haberle sınırlı; geniş doğrulama yok.",
		OverallConfidence:"low",
		KeyFindings:[]liveSynthesisInsightDraft{{Title:"Bildirim bulundu",Assessment:"Yayın gelişmeyi bildiriyor; doğruluğu bağımsız teyit edilmiş değil.",Confidence:"low",ArticleIDs:[]string{"source-1","made-up"}}},
		OpenQuestions:[]string{"Birincil kaynak var mı?"},
	}
	draftJSON,err:=json.Marshal(draft);if err!=nil{t.Fatal(err)}
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Method!=http.MethodPost{t.Errorf("expected POST, got %s",r.Method)}
		if got:=r.Header.Get("Authorization");got!="Bearer test-key"{t.Errorf("expected auth header, got %q",got)}
		var request liveChatCompletionRequest
		if err:=json.NewDecoder(r.Body).Decode(&request);err!=nil{t.Errorf("decode request: %v",err)}
		if request.Model!="test-model"{t.Errorf("unexpected model: %s",request.Model)}
		if len(request.Messages)!=2{t.Errorf("expected system and user messages, got %d",len(request.Messages))}
		w.Header().Set("Content-Type","application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices":[]any{map[string]any{"message":map[string]string{"content":string(draftJSON)}}}})
	}))
	defer server.Close()
	synth:=&OpenAICompatibleSynthesizer{Endpoint:server.URL,APIKey:"test-key",Model:"test-model",Client:server.Client()}
	got,err:=synth.Synthesize(context.Background(),"Enerji altyapısı",[]LiveResearchArticle{article})
	if err!=nil{t.Fatal(err)}
	if got.Model!="test-model"{t.Fatalf("unexpected model: %s",got.Model)}
	if len(got.KeyFindings)!=1||len(got.KeyFindings[0].Sources)!=1{t.Fatalf("expected one finding with one grounded source: %#v",got.KeyFindings)}
	if got.KeyFindings[0].Sources[0].ArticleID!="source-1"{t.Fatalf("unexpected source id: %s",got.KeyFindings[0].Sources[0].ArticleID)}
}

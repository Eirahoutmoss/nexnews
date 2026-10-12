package intelligence

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeLiveSearchProvider struct {
	name string
	items []LiveResearchArticle
	err error
}
func (p fakeLiveSearchProvider) Name() string { return p.name }
func (p fakeLiveSearchProvider) Search(_ context.Context, _ string, _ int) ([]LiveResearchArticle,error) {
	return p.items,p.err
}

func TestLiveResearchEngineDeduplicatesAndReportsPartialCoverage(t *testing.T) {
	engine:=&LiveResearchEngine{
		Providers:[]LiveSearchProvider{
			fakeLiveSearchProvider{name:"feed-a",items:[]LiveResearchArticle{{
				Title:"Enerji altyapısı yatırımları açıklandı",
				URL:"https://example.com/story?utm_source=feed",
				Publisher:"Example",Domain:"example.com",Providers:[]string{"feed-a"},
			}}},
			fakeLiveSearchProvider{name:"feed-b",items:[]LiveResearchArticle{
				{Title:"Enerji altyapısı yatırımları açıklandı",URL:"https://example.com/story?utm_medium=copy",Publisher:"Example",Domain:"example.com",Providers:[]string{"feed-b"}},
				{Title:"Enerji yatırımları için yeni plan",URL:"https://other.example/report",Publisher:"Other",Domain:"other.example",Providers:[]string{"feed-b"}},
			}},
			fakeLiveSearchProvider{name:"feed-down",err:errors.New("provider unavailable")},
		},
		MaxDuration:time.Second,
		Now:func()time.Time{return time.Date(2026,10,12,12,0,0,0,time.UTC)},
	}
	report,err:=engine.Run(context.Background(),LiveResearchRequest{Query:"enerji yatırımları",MaxResults:10})
	if err!=nil{t.Fatalf("Run returned error: %v",err)}
	if report.Status!="partial"{t.Fatalf("expected partial status, got %q",report.Status)}
	if report.ArticleCount!=2{t.Fatalf("expected 2 deduplicated articles, got %d",report.ArticleCount)}
	if report.PublisherCount!=2{t.Fatalf("expected 2 publisher domains, got %d",report.PublisherCount)}
	if got:=len(report.Articles[0].Providers);got!=2{t.Fatalf("duplicate URL should retain both providers, got %d",got)}
	if len(report.Themes)==0{t.Fatal("expected repeated theme signals")}
	if len(report.ProviderStatuses)!=3{t.Fatalf("expected 3 provider statuses, got %d",len(report.ProviderStatuses))}
	if len(report.Limitations)==0{t.Fatal("expected methodological limitations")}
}

func TestLiveResearchEngineRejectsBlankQuery(t *testing.T) {
	engine:=&LiveResearchEngine{Providers:[]LiveSearchProvider{fakeLiveSearchProvider{name:"fake"}}}
	if _,err:=engine.Run(context.Background(),LiveResearchRequest{Query:"   "});err==nil{
		t.Fatal("expected empty query to be rejected")
	}
}

func TestLiveResearchNormalizeURLRemovesTrackingParameters(t *testing.T) {
	got:=normalizeLiveURL("HTTPS://EXAMPLE.COM/a?utm_source=news&x=1&fbclid=abc#section")
	want:="https://example.com/a?x=1"
	if got!=want{t.Fatalf("got %q, want %q",got,want)}
}

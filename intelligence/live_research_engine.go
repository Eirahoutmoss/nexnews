package intelligence

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

type LiveResearchRequest struct {
	Query string "json:\"query\""
	MaxResults int "json:\"maxResults,omitempty\""
}
type LiveResearchArticle struct {
	ID string "json:\"id\""
	Title string "json:\"title\""
	URL string "json:\"url\""
	Publisher string "json:\"publisher\""
	Domain string "json:\"domain\""
	Summary string "json:\"summary,omitempty\""
	PublishedAt *time.Time "json:\"publishedAt,omitempty\""
	RetrievedAt time.Time "json:\"retrievedAt\""
	Providers []string "json:\"providers\""
}
type LiveResearchProviderStatus struct {
	Name string "json:\"name\""
	Status string "json:\"status\""
	ResultCount int "json:\"resultCount\""
	Message string "json:\"message,omitempty\""
}
type LiveResearchTheme struct {
	Term string "json:\"term\""
	ArticleCount int "json:\"articleCount\""
	SourceCount int "json:\"sourceCount\""
	Examples []string "json:\"examples\""
}
type LiveResearchFinding struct {
	Kind string "json:\"kind\""
	Title string "json:\"title\""
	Detail string "json:\"detail\""
	SourceURLs []string "json:\"sourceUrls,omitempty\""
}
type LiveResearchReport struct {
	Query string "json:\"query\""
	GeneratedAt time.Time "json:\"generatedAt\""
	Status string "json:\"status\""
	Summary string "json:\"summary\""
	ArticleCount int "json:\"articleCount\""
	PublisherCount int "json:\"publisherCount\""
	Themes []LiveResearchTheme "json:\"themes\""
	Findings []LiveResearchFinding "json:\"findings\""
	Articles []LiveResearchArticle "json:\"articles\""
	ProviderStatuses []LiveResearchProviderStatus "json:\"providerStatuses\""
	OpenQuestions []string "json:\"openQuestions\""
	Limitations []string "json:\"limitations\""
}
type LiveSearchProvider interface {
	Name() string
	Search(context.Context, string, int) ([]LiveResearchArticle, error)
}
type LiveResearchEngine struct {
	Providers []LiveSearchProvider
	MaxDuration time.Duration
	Now func() time.Time
}
func NewLiveResearchEngine() *LiveResearchEngine {
	return &LiveResearchEngine{
		Providers: []LiveSearchProvider{
			GoogleNewsRSSProvider{Locale:"tr", Client:liveSearchClient()},
			GoogleNewsRSSProvider{Locale:"en", Client:liveSearchClient()},
			GDELTSearchProvider{Client:liveSearchClient()},
		},
		MaxDuration: 12*time.Second, Now:time.Now,
	}
}
func liveSearchClient() *http.Client { return &http.Client{Timeout:7*time.Second} }

func (e *LiveResearchEngine) Run(ctx context.Context, input LiveResearchRequest) (LiveResearchReport, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" { return LiveResearchReport{}, errors.New("araştırma sorusu boş olamaz") }
	if len([]rune(query)) > 500 { return LiveResearchReport{}, errors.New("araştırma sorusu en fazla 500 karakter olabilir") }
	if e == nil || len(e.Providers)==0 { return LiveResearchReport{}, errors.New("araştırma sağlayıcısı yapılandırılmamış") }
	limit := input.MaxResults
	if limit <= 0 { limit=24 }; if limit<5 { limit=5 }; if limit>50 { limit=50 }
	now:=time.Now; if e.Now!=nil { now=e.Now }
	duration:=e.MaxDuration; if duration<=0 { duration=12*time.Second }
	runCtx,cancel:=context.WithTimeout(ctx,duration); defer cancel()
	type result struct { name string; items []LiveResearchArticle; err error }
	results:=make([]result,len(e.Providers))
	var wg sync.WaitGroup
	for i,p:=range e.Providers { i,p:=i,p; wg.Add(1); go func(){ defer wg.Done(); results[i].name=p.Name(); results[i].items,results[i].err=p.Search(runCtx,query,limit) }() }
	wg.Wait()
	report:=LiveResearchReport{
		Query:query,GeneratedAt:now().UTC(),Status:"no_results",
		Themes:[]LiveResearchTheme{},Findings:[]LiveResearchFinding{},Articles:[]LiveResearchArticle{},ProviderStatuses:[]LiveResearchProviderStatus{},
		OpenQuestions:[]string{
			"Bulunan haberler aynı birincil belgeye veya haber ajansına mı dayanıyor?",
			"Resmî açıklama, ilk elden belge ya da ham veri haberlerdeki iddiaları doğruluyor mu?",
			"Sonradan yayımlanmış düzeltme, geri çekme veya karşı kanıt var mı?",
			"Hangi coğrafya, kaynak grubu veya karşı görüş arama sonuçlarında eksik?",
		},
		Limitations:[]string{
			"Bu sürüm herkese açık haber dizinlerini arar; makalelerin tam metnini indirmez. Analiz başlıklar ve varsa kısa özetlerle sınırlıdır.",
			"Yayıncı alan adı veya başlık benzerliği doğruluk ve bağımsız teyit anlamına gelmez. Ortak ajans/kopya kökeni bilinmiyorsa belirsiz kalır.",
			"Motor kural tabanlı kaynak keşfi ve kapsama analizi yapar; LLM kullanmaz, niyet veya nedensellik hakkında otomatik hüküm vermez.",
			"Sağlayıcı dizinleri ve yanıt süreleri değişebilir. Başarısız aramalar raporda açıkça işaretlenir.",
		},
	}
	var all []LiveResearchArticle
	okProviders:=0
	for _,r:=range results {
		s:=LiveResearchProviderStatus{Name:r.name}
		if r.err!=nil { s.Status="failed"; s.Message=liveSafeError(r.err) } else { s.Status="ok"; s.ResultCount=len(r.items); okProviders++ }
		report.ProviderStatuses=append(report.ProviderStatuses,s)
		all=append(all,r.items...)
	}
	if okProviders>0 { report.Status="completed"; if okProviders<len(results){report.Status="partial"} }
	report.Articles=normalizeLiveArticles(all,now().UTC())
	if len(report.Articles)>limit { report.Articles=report.Articles[:limit] }
	report.ArticleCount=len(report.Articles)
	domains:=map[string]bool{}
	for _,a:=range report.Articles { if a.Domain!="" {domains[a.Domain]=true} }
	report.PublisherCount=len(domains)
	report.Themes=buildLiveThemes(report.Articles)
	report.Findings=buildLiveFindings(report)
	if report.ArticleCount==0 {
		report.Summary="Bu sorguda kullanılabilir sonuç bulunamadı. Sağlayıcı durumlarını kontrol edin veya soruyu daha ayırt edici ifadelerle deneyin."
	} else {
		report.Summary=fmt.Sprintf("“%s” için %d benzersiz bağlantı ve %d farklı yayıncı alan adı bulundu. Bu, haber kapsamının bir fotoğrafıdır; iddiaların doğrulandığı anlamına gelmez.",query,report.ArticleCount,report.PublisherCount)
	}
	return report,nil
}
func liveSafeError(err error) string {
	if errors.Is(err,context.DeadlineExceeded){return "Zaman sınırı içinde yanıt alınamadı."}
	if errors.Is(err,context.Canceled){return "Araştırma isteği iptal edildi."}
	s:=strings.TrimSpace(err.Error()); if len(s)>160{s=s[:160]}; return s
}

type GoogleNewsRSSProvider struct { Locale string; Client *http.Client }
func (p GoogleNewsRSSProvider) Name() string { if strings.EqualFold(p.Locale,"en"){return "Google News · EN"}; return "Google News · TR" }
func (p GoogleNewsRSSProvider) Search(ctx context.Context, query string, limit int) ([]LiveResearchArticle,error) {
	hl,gl,ceid:="tr","TR","TR:tr"; if strings.EqualFold(p.Locale,"en"){hl,gl,ceid="en-US","US","US:en"}
	v:=url.Values{};v.Set("q",query);v.Set("hl",hl);v.Set("gl",gl);v.Set("ceid",ceid)
	client:=p.Client;if client==nil{client=liveSearchClient()}
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,"https://news.google.com/rss/search?"+v.Encode(),nil);if err!=nil{return nil,err}
	req.Header.Set("User-Agent","NexNews-Intelligence/1.0");req.Header.Set("Accept","application/rss+xml, application/xml, text/xml")
	resp,err:=client.Do(req);if err!=nil{return nil,err};defer resp.Body.Close()
	if resp.StatusCode<200||resp.StatusCode>=300{return nil,fmt.Errorf("Google News RSS HTTP %d",resp.StatusCode)}
	body,err:=io.ReadAll(io.LimitReader(resp.Body,4<<20));if err!=nil{return nil,err}
	var feed struct { Channel struct { Items []struct {
		Title string "xml:\"title\""; Link string "xml:\"link\""; PubDate string "xml:\"pubDate\""; Description string "xml:\"description\""
		Source struct { Title string "xml:\",chardata\""; URL string "xml:\"url,attr\"" } "xml:\"source\""
	} "xml:\"item\"" } "xml:\"channel\"" }
	if err=xml.Unmarshal(body,&feed);err!=nil{return nil,fmt.Errorf("Google News RSS XML: %w",err)}
	out:=make([]LiveResearchArticle,0,limit)
	for _,item:=range feed.Channel.Items {
		if len(out)>=limit{break};title:=cleanLiveText(item.Title);link:=normalizeLiveURL(item.Link);if title==""||link==""{continue}
		domain:=liveDomainOf(normalizeLiveURL(item.Source.URL));if domain==""{domain=liveDomainOf(link)}
		publisher:=cleanLiveText(item.Source.Title);if publisher==""{publisher=domain}
		var published *time.Time
		for _,layout:=range []string{time.RFC1123Z,time.RFC1123,time.RFC822Z,time.RFC822,time.RFC3339}{if t,e:=time.Parse(layout,strings.TrimSpace(item.PubDate));e==nil{u:=t.UTC();published=&u;break}}
		out=append(out,LiveResearchArticle{Title:title,URL:link,Publisher:publisher,Domain:domain,Summary:cleanLiveText(item.Description),PublishedAt:published,Providers:[]string{p.Name()}})
	}
	return out,nil
}
type GDELTSearchProvider struct { Client *http.Client }
func (p GDELTSearchProvider) Name() string { return "GDELT DOC" }
func (p GDELTSearchProvider) Search(ctx context.Context,query string,limit int)([]LiveResearchArticle,error){
	v:=url.Values{};v.Set("query",query);v.Set("mode","ArtList");v.Set("format","json");v.Set("maxrecords",fmt.Sprint(minLiveResearch(limit,50)));v.Set("sort","HybridRel")
	client:=p.Client;if client==nil{client=liveSearchClient()}
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,"https://api.gdeltproject.org/api/v2/doc/doc?"+v.Encode(),nil);if err!=nil{return nil,err}
	req.Header.Set("User-Agent","NexNews-Intelligence/1.0")
	resp,err:=client.Do(req);if err!=nil{return nil,err};defer resp.Body.Close()
	if resp.StatusCode<200||resp.StatusCode>=300{return nil,fmt.Errorf("GDELT DOC HTTP %d",resp.StatusCode)}
	body,err:=io.ReadAll(io.LimitReader(resp.Body,5<<20));if err!=nil{return nil,err}
	var payload struct { Articles []struct {
		URL string "json:\"url\""; Title string "json:\"title\""; Seendate string "json:\"seendate\""; Domain string "json:\"domain\""; SourceCountry string "json:\"sourcecountry\""
	} "json:\"articles\"" }
	if err=json.Unmarshal(body,&payload);err!=nil{return nil,fmt.Errorf("GDELT DOC JSON: %w",err)}
	out:=make([]LiveResearchArticle,0,len(payload.Articles))
	for _,row:=range payload.Articles { if len(out)>=limit{break};title:=cleanLiveText(row.Title);link:=normalizeLiveURL(row.URL);if title==""||link==""{continue}
		domain:=strings.ToLower(strings.TrimPrefix(strings.TrimSpace(row.Domain),"www."));if domain==""{domain=liveDomainOf(link)}
		var published *time.Time;if t,e:=time.Parse("20060102150405",strings.TrimSpace(row.Seendate));e==nil{u:=t.UTC();published=&u}
		publisher:=domain;if row.SourceCountry!=""{publisher+=" · "+strings.TrimSpace(row.SourceCountry)}
		out=append(out,LiveResearchArticle{Title:title,URL:link,Publisher:publisher,Domain:domain,PublishedAt:published,Providers:[]string{p.Name()}})
	}
	return out,nil
}

var liveTagPattern=regexp.MustCompile("<[^>]*>")
func cleanLiveText(s string) string { s=html.UnescapeString(s);s=liveTagPattern.ReplaceAllString(s," ");return strings.Join(strings.Fields(s)," ") }
func normalizeLiveURL(raw string) string {
	u,err:=url.Parse(strings.TrimSpace(raw));if err!=nil||u.Hostname()==""||(!strings.EqualFold(u.Scheme,"http")&&!strings.EqualFold(u.Scheme,"https")){return ""}
	u.Scheme=strings.ToLower(u.Scheme);u.Host=strings.ToLower(u.Host);u.Fragment=""
	q:=u.Query();for k:=range q{l:=strings.ToLower(k);if strings.HasPrefix(l,"utm_")||l=="fbclid"||l=="gclid"||l=="ved"||l=="usg"{q.Del(k)}};u.RawQuery=q.Encode();return u.String()
}
func liveDomainOf(raw string) string { u,e:=url.Parse(raw);if e!=nil{return ""};return strings.ToLower(strings.TrimPrefix(u.Hostname(),"www.")) }
func normalizeLiveArticles(input []LiveResearchArticle,at time.Time) []LiveResearchArticle {
	byURL:=map[string]int{};out:=make([]LiveResearchArticle,0,len(input))
	for _,a:=range input {
		a.Title=cleanLiveText(a.Title);a.URL=normalizeLiveURL(a.URL);if a.Title==""||a.URL==""{continue}
		if a.Domain==""{a.Domain=liveDomainOf(a.URL)};a.Domain=strings.ToLower(strings.TrimPrefix(strings.TrimSpace(a.Domain),"www."))
		if a.Publisher==""{a.Publisher=a.Domain};a.Summary=cleanLiveText(a.Summary);if len([]rune(a.Summary))>700{a.Summary=string([]rune(a.Summary)[:700])}
		if a.RetrievedAt.IsZero(){a.RetrievedAt=at};if a.ID==""{a.ID=stableID("live_article",a.URL)}
		a.Providers=uniqueLiveProviders(a.Providers)
		if i,ok:=byURL[a.URL];ok{out[i].Providers=uniqueLiveProviders(append(out[i].Providers,a.Providers...));if out[i].Summary==""{out[i].Summary=a.Summary};if out[i].PublishedAt==nil{out[i].PublishedAt=a.PublishedAt};continue}
		byURL[a.URL]=len(out);out=append(out,a)
	}
	sort.SliceStable(out,func(i,j int)bool{a,b:=out[i].PublishedAt,out[j].PublishedAt;if a!=nil&&b!=nil&&!a.Equal(*b){return a.After(*b)};if a!=nil&&b==nil{return true};if a==nil&&b!=nil{return false};return strings.ToLower(out[i].Title)<strings.ToLower(out[j].Title)})
	return out
}
func uniqueLiveProviders(values []string) []string {seen:=map[string]bool{};out:=[]string{};for _,s:=range values{s=strings.TrimSpace(s);if s!=""&&!seen[s]{seen[s]=true;out=append(out,s)}};sort.Strings(out);return out}

var liveStopwords=func()map[string]bool{m:=map[string]bool{};for _,s:=range strings.Fields("acaba ama ancak artık aslında az bazı böyle bu bunun çok çünkü da daha de defa diye en fakat gibi göre hangi hem hep her herkes hiç için ile ise kadar ki kim mi mu mü mı nasıl ne neden nerede nereye niçin niye o olan olarak olur onu veya ve ya yani yeni yine yok the a an and are as at be been being but by for from had has have he her him his i if in into is it its of on or our she that their them then there these they this to was we were what when where which who will with would you your about after also among before between can could did do does during each few further here how more most other over same some such than too under until up very not no nor") {m[s]=true};return m}()
func liveWords(s string) []string {var b strings.Builder;for _,r:=range strings.ToLower(s){if unicode.IsLetter(r)||unicode.IsDigit(r){b.WriteRune(r)}else{b.WriteByte(' ')}};return strings.Fields(b.String())}
func buildLiveThemes(articles []LiveResearchArticle) []LiveResearchTheme {
	type aggregate struct{count int;domains map[string]bool;examples []string}
	m:=map[string]*aggregate{}
	for _,a:=range articles {seen:=map[string]bool{};for _,word:=range liveWords(a.Title+" "+a.Summary){if len([]rune(word))<4||liveStopwords[word]||seen[word]{continue};seen[word]=true;x:=m[word];if x==nil{x=&aggregate{domains:map[string]bool{},examples:[]string{}};m[word]=x};x.count++;if a.Domain!=""{x.domains[a.Domain]=true};if len(x.examples)<2{x.examples=append(x.examples,a.Title)}}}
	out:=[]LiveResearchTheme{};for word,x:=range m{if x.count>=2{out=append(out,LiveResearchTheme{Term:word,ArticleCount:x.count,SourceCount:len(x.domains),Examples:x.examples})}}
	sort.Slice(out,func(i,j int)bool{if out[i].SourceCount!=out[j].SourceCount{return out[i].SourceCount>out[j].SourceCount};if out[i].ArticleCount!=out[j].ArticleCount{return out[i].ArticleCount>out[j].ArticleCount};return out[i].Term<out[j].Term});if len(out)>10{out=out[:10]};return out
}
func buildLiveFindings(r LiveResearchReport) []LiveResearchFinding {
	out:=[]LiveResearchFinding{{Kind:"coverage",Title:"Kaynak kapsamı",Detail:fmt.Sprintf("%d bağlantı, %d farklı yayıncı alan adı. Alan adı sayısı bağımsız doğrulama sayısı değildir.",r.ArticleCount,r.PublisherCount)}}
	for _,t:=range r.Themes{if len(out)>=6{break};urls:=[]string{};for _,a:=range r.Articles{for _,ex:=range t.Examples{if a.Title==ex&&len(urls)<2{urls=append(urls,a.URL)}}};out=append(out,LiveResearchFinding{Kind:"recurring-theme",Title:"Tekrarlanan konu: "+t.Term,Detail:fmt.Sprintf("Bu terim %d haber kaydında ve %d yayıncı alan adında görülüyor. Bu bir metin-kapsam sinyalidir, olayın kanıtı değildir.",t.ArticleCount,t.SourceCount),SourceURLs:urls})}
	for _,p:=range r.ProviderStatuses{if p.Status=="failed"{out=append(out,LiveResearchFinding{Kind:"provider-gap",Title:p.Name+" araması tamamlanamadı",Detail:p.Message+" Sonuç kümesi bu nedenle eksik olabilir."})}}
	if r.ArticleCount==0{out=append(out,LiveResearchFinding{Kind:"gap",Title:"İncelenecek kaynak bulunamadı",Detail:"Soru için dizin sonucu gelmedi. Farklı bir ifade, kişi/kurum adı veya zaman aralığıyla yeniden deneyin."})}
	return out
}
func minLiveResearch(a,b int)int{if a<b{return a};return b}

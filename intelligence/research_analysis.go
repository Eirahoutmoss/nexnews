package intelligence

import (
	"sort"
	"strings"
	"time"
)

// ResearchAnalysisReport is a deterministic evidence-coverage analysis. It does
// not determine factual truth or infer causal intent; it reports what evidence
// is linked to each hypothesis and where the research still has blind spots.
type ResearchAnalysisReport struct {
	CaseID                       string                       `json:"caseId"`
	GeneratedAt                  time.Time                    `json:"generatedAt"`
	EvidenceCount                int                          `json:"evidenceCount"`
	PublisherDomainCount         int                          `json:"publisherDomainCount"`
	SourceKindCounts             map[ResearchSourceKind]int   `json:"sourceKindCounts"`
	SourceOriginCounts           map[ResearchSourceOrigin]int `json:"sourceOriginCounts"`
	FullTextOrStructuredCount    int                          `json:"fullTextOrStructuredCount"`
	AttributedClaimCount         int                          `json:"attributedClaimCount"`
	UnknownSourceOriginCount     int                          `json:"unknownSourceOriginCount"`
	KnownSyndicationGroupCount   int                          `json:"knownSyndicationGroupCount"`
	OpenQuestionCount            int                          `json:"openQuestionCount"`
	Hypotheses                   []HypothesisEvidenceCoverage `json:"hypotheses"`
	Limitations                  []string                     `json:"limitations"`
}

type HypothesisEvidenceCoverage struct {
	HypothesisID                    string                      `json:"hypothesisId"`
	Statement                       string                      `json:"statement"`
	Status                          ResearchHypothesisStatus    `json:"status"`
	SupportingEvidenceCount         int                         `json:"supportingEvidenceCount"`
	ContradictingEvidenceCount      int                         `json:"contradictingEvidenceCount"`
	ContextEvidenceCount            int                         `json:"contextEvidenceCount"`
	SupportingPublisherDomains      []string                    `json:"supportingPublisherDomains"`
	ContradictingPublisherDomains   []string                    `json:"contradictingPublisherDomains"`
	ContextPublisherDomains         []string                    `json:"contextPublisherDomains"`
	UnknownOriginEvidenceCount      int                         `json:"unknownOriginEvidenceCount"`
	SharedSyndicationGroups         []string                    `json:"sharedSyndicationGroups"`
	CoverageLabel                   string                      `json:"coverageLabel"`
	Limitations                     []string                    `json:"limitations"`
}

func AnalyzeResearchCase(item ResearchCase) ResearchAnalysisReport {
	report := ResearchAnalysisReport{
		CaseID: item.ID, GeneratedAt: time.Now().UTC(),
		EvidenceCount: len(item.Evidence),
		SourceKindCounts: map[ResearchSourceKind]int{},
		SourceOriginCounts: map[ResearchSourceOrigin]int{},
		Hypotheses: []HypothesisEvidenceCoverage{},
		Limitations: []string{},
	}
	domains := map[string]bool{}
	syndicationGroups := map[string]bool{}
	fullTextOrData := 0
	unknownOrigins := 0
	attributedClaims := 0
	for _, evidence := range item.Evidence {
		domain := normalizedPublisherDomain(evidence.PublisherDomain)
		if domain != "" { domains[domain] = true }
		report.SourceKindCounts[evidence.SourceKind]++
		report.SourceOriginCounts[evidence.SourceOrigin]++
		if evidence.ContentLevel == ResearchContentFullText || evidence.ContentLevel == ResearchContentStructured {
			fullTextOrData++
		}
		if evidence.SourceOrigin == ResearchOriginUnknown { unknownOrigins++ }
		if evidence.Kind == ResearchEvidenceClaim { attributedClaims++ }
		if group := strings.TrimSpace(evidence.SyndicationGroup); group != "" { syndicationGroups[group] = true }
	}
	report.PublisherDomainCount = len(domains)
	report.FullTextOrStructuredCount = fullTextOrData
	report.AttributedClaimCount = attributedClaims
	report.UnknownSourceOriginCount = unknownOrigins
	report.KnownSyndicationGroupCount = len(syndicationGroups)
	for _, question := range item.OpenQuestions {
		if question.Status != "resolved" { report.OpenQuestionCount++ }
	}

	for _, hypothesis := range item.Hypotheses {
		coverage := HypothesisEvidenceCoverage{
			HypothesisID: hypothesis.ID, Statement: hypothesis.Statement, Status: hypothesis.Status,
			SupportingPublisherDomains: []string{}, ContradictingPublisherDomains: []string{},
			ContextPublisherDomains: []string{}, SharedSyndicationGroups: []string{}, Limitations: []string{},
		}
		supportDomains := map[string]bool{}
		contraDomains := map[string]bool{}
		contextDomains := map[string]bool{}
		groups := map[string]bool{}
		for _, evidence := range item.Evidence {
			for _, assessment := range evidence.HypothesisAssessments {
				if assessment.HypothesisID != hypothesis.ID { continue }
				if evidence.SourceOrigin == ResearchOriginUnknown { coverage.UnknownOriginEvidenceCount++ }
				if group := strings.TrimSpace(evidence.SyndicationGroup); group != "" { groups[group] = true }
				domain := normalizedPublisherDomain(evidence.PublisherDomain)
				switch assessment.Direction {
				case HypothesisEvidenceSupports:
					coverage.SupportingEvidenceCount++
					if domain != "" { supportDomains[domain] = true }
				case HypothesisEvidenceContradicts:
					coverage.ContradictingEvidenceCount++
					if domain != "" { contraDomains[domain] = true }
				case HypothesisEvidenceContext:
					coverage.ContextEvidenceCount++
					if domain != "" { contextDomains[domain] = true }
				}
			}
		}
		coverage.SupportingPublisherDomains = sortedKeys(supportDomains)
		coverage.ContradictingPublisherDomains = sortedKeys(contraDomains)
		coverage.ContextPublisherDomains = sortedKeys(contextDomains)
		coverage.SharedSyndicationGroups = sortedKeys(groups)
		coverage.CoverageLabel, coverage.Limitations = labelHypothesisCoverage(coverage)
		report.Hypotheses = append(report.Hypotheses, coverage)
	}

	if len(item.Evidence) == 0 {
		report.Limitations = append(report.Limitations, "Henüz kaynaklı kanıt kaydedilmedi. Bu dosyadaki hipotezlerin hiçbiri kanıta dayalı olarak değerlendirilemez.")
	}
	if unknownOrigins > 0 {
		report.Limitations = append(report.Limitations, "Bazı kaynakların özgün araştırma mı, resmî açıklama mı yoksa yeniden yayımlama mı olduğu bilinmiyor.")
	}
	if len(item.Evidence) > 0 && fullTextOrData == 0 {
		report.Limitations = append(report.Limitations, "Tam metin veya yapılandırılmış veri bulunmuyor. Kaynak kapsamı başlık/özet düzeyindeyse sonuçlar bu sınırı aşmamalı.")
	}
	if attributedClaims > 0 {
		report.Limitations = append(report.Limitations, "Atfedilmiş iddia kayıtları, olayın kendisinin bağımsız doğrulandığı anlamına gelmez.")
	}
	if len(domains) > 1 {
		report.Limitations = append(report.Limitations, "Farklı yayıncı alan adları bağımsız kaynak sayısını kanıtlamaz; kopyalama, ajans ve ortak kaynak kökeni ayrıca incelenmelidir.")
	}
	if len(syndicationGroups) > 0 {
		report.Limitations = append(report.Limitations, "Bilinen ortak yayımlama/kaynak kökeni etiketleri var. Bu gruplardaki yayınlar ayrı teyitler gibi sayılmamalıdır.")
	}
	if _, ok := report.SourceKindCounts[ResearchSourceLocalReport]; !ok {
		report.Limitations = append(report.Limitations, "Yerel/saha haberciliği kaydı henüz yok. Bu, sahada bilgi bulunmadığını değil, bu dosyaya eklenmediğini gösterir.")
	}
	if _, ok := report.SourceKindCounts[ResearchSourcePrimary]; !ok && len(item.Evidence) > 0 {
		report.Limitations = append(report.Limitations, "Birincil kaynak kaydı bulunmuyor. Tarafların doğrudan açıklamaları ve mümkün olan özgün belgeler araştırılmalı.")
	}
	if _, ok := report.SourceKindCounts[ResearchSourceDataset]; !ok {
		report.Limitations = append(report.Limitations, "Yapılandırılmış veri kümesi kaydedilmedi. Nicel örüntü iddiaları için olay listesi ve karşılaştırma evreni eksik olabilir.")
	}
	if report.OpenQuestionCount > 0 {
		report.Limitations = append(report.Limitations, "Araştırma dosyasında yanıt bekleyen açık sorular bulunuyor.")
	}
	return report
}

func labelHypothesisCoverage(c HypothesisEvidenceCoverage) (string, []string) {
	notes := []string{}
	total := c.SupportingEvidenceCount + c.ContradictingEvidenceCount + c.ContextEvidenceCount
	if total == 0 {
		return "no_linked_evidence", []string{"Bu hipoteze bağlı kaynak değerlendirmesi yok. Durum açık kalmalı."}
	}
	if c.SupportingEvidenceCount > 0 && c.ContradictingEvidenceCount > 0 {
		notes = append(notes, "Hipoteze hem destek hem karşı kanıt bağlandı; çelişki ortadan kaldırılmadan iki taraf da görünür tutulmalı.")
	} else if c.SupportingEvidenceCount > 0 && c.ContradictingEvidenceCount == 0 {
		notes = append(notes, "Şu an bağlı karşı kanıt yok. Karşı kanıt bulunmaması, hipotezin doğrulandığı anlamına gelmez.")
	} else if c.SupportingEvidenceCount == 0 && c.ContradictingEvidenceCount > 0 {
		notes = append(notes, "Şu an bağlı destekleyici kanıt yok. Karşı kanıt, hipotezin reddini otomatik olarak kanıtlamaz.")
	} else {
		notes = append(notes, "Bağlı kaynaklar yalnızca bağlam sağlıyor; hipotezi doğrudan destekleyen veya çürüten değerlendirme kaydedilmedi.")
	}
	domainCount := len(union(nil, append(append([]string{}, c.SupportingPublisherDomains...), c.ContradictingPublisherDomains...)))
	if domainCount <= 1 {
		notes = append(notes, "Destek/karşı kanıt en fazla bir yayıncı alan adına bağlı. Kaynak çeşitliliği sınırlı.")
	} else {
		notes = append(notes, "Birden çok yayıncı alan adı var; bu tek başına bağımsız doğrulama değildir.")
	}
	if c.UnknownOriginEvidenceCount > 0 {
		notes = append(notes, "Bu hipoteze bağlı kaynaklardan bazılarının kökeni bilinmiyor.")
	}
	if len(c.SharedSyndicationGroups) > 0 {
		notes = append(notes, "Bu hipoteze bağlı bazı kayıtlar ortak yayımlama/kaynak kökeni etiketi taşıyor; tekrarlar ayrı bağımsız kanıt sayılmamalı.")
	}
	switch {
	case c.SupportingEvidenceCount > 0 && c.ContradictingEvidenceCount > 0:
		return "mixed_evidence", notes
	case c.SupportingEvidenceCount > 0 && c.ContradictingEvidenceCount == 0:
		return "supporting_evidence_recorded", notes
	case c.SupportingEvidenceCount == 0 && c.ContradictingEvidenceCount > 0:
		return "contradictory_evidence_recorded", notes
	default:
		return "context_only", notes
	}
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values { out = append(out, value) }
	sort.Strings(out)
	return out
}

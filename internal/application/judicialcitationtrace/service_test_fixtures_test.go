package judicialcitationtrace_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type decisionDetailsOptions struct {
	fullText            bool
	reporterCitation    string
	lowerCourtName      string
	lowerCourtCase      string
	referencedProvision string
}

func mustDecisionDetails(
	t *testing.T,
	options decisionDetailsOptions,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	documents := []model.JudicialDocumentLink{}
	if options.fullText {
		document, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
			Kind:      model.JudicialDocumentKindFullText,
			Label:     "全文",
			MediaType: model.JudicialDocumentMediaTypePDF,
			URL:       "https://www.courts.go.jp/assets/hanrei/hanrei-pdf.pdf",
		})
		if err != nil {
			t.Fatalf("document = %v", err)
		}
		documents = append(documents, document)
	}
	ref := mustRef(t, "root:detail2")
	summary := mustSummary(
		t,
		ref,
		"root",
		"令和6年（受）第1号",
		"2026-08-26",
		model.JudicialPublicationCategorySupremeCourt,
		documents,
	)
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:                  summary,
		ReporterCitation:         optionalString(options.reporterCitation),
		LowerCourtName:           optionalString(options.lowerCourtName),
		LowerCourtCaseNumber:     optionalString(options.lowerCourtCase),
		ReferencedProvisionsText: optionalString(options.referencedProvision),
	})
	if err != nil {
		t.Fatalf("details = %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: []model.Provenance{mustDetailProvenance(t, ref.Key())},
		Data:       details,
	})
	if err != nil {
		t.Fatalf("resource = %v", err)
	}
	return resource
}

func mustSummary(
	t *testing.T,
	ref model.SourceResourceRef,
	decisionID string,
	caseNumber string,
	date string,
	category model.JudicialPublicationCategory,
	documents []model.JudicialDocumentLink,
) model.JudicialDecisionSummary {
	t.Helper()
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          decisionID,
		PublicationCategory: category,
		SourceCategoryLabel: "裁判例",
		CaseNumber:          caseNumber,
		DecisionDate:        mustDate(t, date),
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/" + decisionID + "/detail2/index.html",
		Documents:           documents,
		Source:              mustSource(t),
	})
	if err != nil {
		t.Fatalf("summary = %v", err)
	}
	_ = ref
	return summary
}

func mustRef(t *testing.T, resourceID string) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("key = %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref = %v", err)
	}
	return ref
}

func mustDetailProvenance(t *testing.T, key model.SourceResourceKey) model.Provenance {
	t.Helper()
	value, err := model.NewProvenance(model.ProvenanceValues{
		Source:         mustSource(t),
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/hanrei/root/detail2/index.html",
		RetrievedAt:    time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("detail provenance = %v", err)
	}
	return value
}

func mustDecisionMention(t *testing.T, identity, location string) model.JudicialCitationDecisionMention {
	return mustDecisionMentionWithReference(t, identity, identity, location)
}

func mustDecisionMentionWithReference(
	t *testing.T,
	referenceText, identity, location string,
) model.JudicialCitationDecisionMention {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision-document",
		ResourceID:   "root:full-text",
	})
	if err != nil {
		t.Fatalf("pdf key = %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         mustSource(t),
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/assets/hanrei/hanrei-pdf.pdf",
		RetrievedAt:    time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC),
		MediaType:      model.JudicialDocumentMediaTypePDF,
		Location:       location,
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-071",
		ContentDigest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatalf("pdf provenance = %v", err)
	}
	excerpt := referenceText
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
		Provenance:    provenance,
		Excerpt:       &excerpt,
	})
	if err != nil {
		t.Fatalf("evidence = %v", err)
	}
	mention, err := model.NewJudicialCitationDecisionMention(
		model.JudicialCitationDecisionMentionValues{
			ReferenceText:        referenceText,
			DecisionIdentityText: identity,
			Evidence:             evidence,
		},
	)
	if err != nil {
		t.Fatalf("mention = %v", err)
	}
	return mention
}

func mustExtractResult(
	t *testing.T,
	mentions []model.JudicialCitationDecisionMention,
	truncated bool,
) judicialcasecitationextract.Result {
	t.Helper()
	if mentions == nil {
		mentions = []model.JudicialCitationDecisionMention{}
	}
	result, err := judicialcasecitationextract.NewResult(judicialcasecitationextract.ResultValues{
		ConfirmedDecisionMentions: mentions,
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        judicialcasecitationextract.DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           len(mentions),
		Truncated:                 truncated,
	})
	if err != nil {
		t.Fatalf("extract result = %v", err)
	}
	return result
}

func mustCandidate(
	t *testing.T,
	decisionID string,
	date string,
	category model.JudicialPublicationCategory,
) judicialcitingcandidatesearch.Candidate {
	t.Helper()
	ref := mustRef(t, decisionID+":detail2")
	summary := mustSummary(t, ref, decisionID, "令和5年（受）第2号", date, category, []model.JudicialDocumentLink{})
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         mustSource(t),
		ResourceKey:    ref.Key(),
		URL:            "https://www.courts.go.jp/hanrei/" + decisionID + "/detail2/index.html",
		RetrievedAt:    time.Date(2026, 8, 27, 1, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-073",
	})
	if err != nil {
		t.Fatalf("candidate provenance = %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionSummary]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       summary,
	})
	if err != nil {
		t.Fatalf("candidate resource = %v", err)
	}
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialSearchCandidate,
		Provenance:    provenance,
	})
	if err != nil {
		t.Fatalf("candidate evidence = %v", err)
	}
	candidate, err := judicialcitingcandidatesearch.NewCandidate(
		judicialcitingcandidatesearch.CandidateValues{
			Decision: resource,
			Evidence: []model.JudicialCitationEvidence{evidence},
		},
	)
	if err != nil {
		t.Fatalf("candidate = %v", err)
	}
	return candidate
}

func mustCandidateResult(
	t *testing.T,
	items []judicialcitingcandidatesearch.Candidate,
) judicialcitingcandidatesearch.Result {
	t.Helper()
	coverage, err := judicialcitingcandidatesearch.NewCoverage(
		judicialcitingcandidatesearch.CoverageValues{
			Attempts: []judicialcitingcandidatesearch.CoverageAttempt{
				mustCoverageAttempt(t, judicialcitingcandidatesearch.SearchKindCaseNumber),
				mustCoverageAttempt(t, judicialcitingcandidatesearch.SearchKindReporterCitation),
			},
			ObservedItemCount:     len(items),
			DedupedCandidateCount: len(items),
		},
	)
	if err != nil {
		t.Fatalf("candidate coverage = %v", err)
	}
	result, err := judicialcitingcandidatesearch.NewResult(
		judicialcitingcandidatesearch.ResultValues{
			Status:   judicialcitingcandidatesearch.ResultStatusComplete,
			Items:    items,
			Coverage: coverage,
			Issues:   []judicialcitingcandidatesearch.Issue{},
		},
	)
	if err != nil {
		t.Fatalf("candidate result = %v", err)
	}
	return result
}

func mustCandidatePartialResult(
	t *testing.T,
	items []judicialcitingcandidatesearch.Candidate,
) judicialcitingcandidatesearch.Result {
	t.Helper()
	complete := mustCoverageAttempt(t, judicialcitingcandidatesearch.SearchKindCaseNumber)
	failed, err := judicialcitingcandidatesearch.NewCoverageAttempt(
		judicialcitingcandidatesearch.CoverageAttemptValues{
			SearchKind: judicialcitingcandidatesearch.SearchKindReporterCitation,
			Status:     judicialcitingcandidatesearch.AttemptStatusFailed,
		},
	)
	if err != nil {
		t.Fatalf("failed attempt = %v", err)
	}
	coverage, err := judicialcitingcandidatesearch.NewCoverage(
		judicialcitingcandidatesearch.CoverageValues{
			Attempts:              []judicialcitingcandidatesearch.CoverageAttempt{complete, failed},
			ObservedItemCount:     len(items),
			DedupedCandidateCount: len(items),
		},
	)
	if err != nil {
		t.Fatalf("partial coverage = %v", err)
	}
	issue, err := judicialcitingcandidatesearch.NewIssue(
		judicialcitingcandidatesearch.IssueValues{
			SearchKind: judicialcitingcandidatesearch.SearchKindReporterCitation,
			SourceError: mustTraceSourceError(
				t,
				judicialcitingcandidatesearch.CapabilityID,
				model.SourceErrorCodeSourceUnavailable,
			),
		},
	)
	if err != nil {
		t.Fatalf("candidate issue = %v", err)
	}
	result, err := judicialcitingcandidatesearch.NewResult(
		judicialcitingcandidatesearch.ResultValues{
			Status:   judicialcitingcandidatesearch.ResultStatusPartial,
			Items:    items,
			Coverage: coverage,
			Issues:   []judicialcitingcandidatesearch.Issue{issue},
		},
	)
	if err != nil {
		t.Fatalf("partial candidate result = %v", err)
	}
	return result
}

func mustTruncatedCandidateResult(
	t *testing.T,
	items []judicialcitingcandidatesearch.Candidate,
) judicialcitingcandidatesearch.Result {
	t.Helper()
	coverage, err := judicialcitingcandidatesearch.NewCoverage(
		judicialcitingcandidatesearch.CoverageValues{
			Attempts: []judicialcitingcandidatesearch.CoverageAttempt{
				mustCoverageAttempt(t, judicialcitingcandidatesearch.SearchKindCaseNumber),
			},
			ObservedItemCount:     len(items) + 1,
			DedupedCandidateCount: len(items) + 1,
			Truncated:             true,
		},
	)
	if err != nil {
		t.Fatalf("truncated coverage = %v", err)
	}
	result, err := judicialcitingcandidatesearch.NewResult(
		judicialcitingcandidatesearch.ResultValues{
			Status:   judicialcitingcandidatesearch.ResultStatusComplete,
			Items:    items,
			Coverage: coverage,
			Issues:   []judicialcitingcandidatesearch.Issue{},
		},
	)
	if err != nil {
		t.Fatalf("truncated candidate result = %v", err)
	}
	return result
}

type traceTestOperation string

const traceTestGET traceTestOperation = "GET /trace-test"

const traceTestProviderID = "trace-test-provider"

func (traceTestOperation) SourceOperationProviderID() string { return traceTestProviderID }
func (o traceTestOperation) SourceOperationName() string     { return string(o) }
func (o traceTestOperation) ValidateSourceOperation() error {
	if o != traceTestGET {
		return fmt.Errorf("試験用 operation が定義されていません")
	}
	return nil
}

func mustTraceSourceError(
	t *testing.T,
	capabilityID string,
	code model.SourceErrorCode,
) model.SourceError {
	t.Helper()
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           capabilityID,
		MajorVersion: 1,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("capability = %v", err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             traceTestProviderID,
		Source:                 mustSource(t),
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             mustDate(t, "2026-08-27"),
		InterfaceType:          model.InterfaceTypeHTML,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		t.Fatalf("descriptor = %v", err)
	}
	result, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   descriptor,
		Capability: capability,
		Operation:  traceTestGET,
	})
	if err != nil {
		t.Fatalf("source error = %v", err)
	}
	return result
}

func mustCoverageAttempt(
	t *testing.T,
	kind judicialcitingcandidatesearch.SearchKind,
) judicialcitingcandidatesearch.CoverageAttempt {
	t.Helper()
	attempt, err := judicialcitingcandidatesearch.NewCoverageAttempt(
		judicialcitingcandidatesearch.CoverageAttemptValues{
			SearchKind: kind,
			Status:     judicialcitingcandidatesearch.AttemptStatusComplete,
		},
	)
	if err != nil {
		t.Fatalf("attempt = %v", err)
	}
	return attempt
}

type lawEntry struct{ id, title string }

func mustLawResolver(
	t *testing.T,
	entries []lawEntry,
) judicialcitationnormalize.ExactLawAliasResolver {
	t.Helper()
	if len(entries) == 0 {
		entries = []lawEntry{{"dummy-law", "試験用法律"}}
	}
	values := make([]lawnamelexicon.Entry, len(entries))
	for index, entry := range entries {
		values[index] = lawnamelexicon.Entry{ResourceID: entry.id, Canonical: entry.title}
	}
	resolver, err := judicialcitationnormalize.NewExactLawAliasResolver(values)
	if err != nil {
		t.Fatalf("resolver = %v", err)
	}
	return resolver
}

func mustService(
	t *testing.T,
	reader judicialdecisionread.Port,
	extractor judicialcasecitationextract.Port,
	searcher judicialcitingcandidatesearch.Port,
	resolver judicialcitationnormalize.ExactLawAliasResolver,
) *judicialcitationtrace.Service {
	t.Helper()
	service, err := judicialcitationtrace.NewService(reader, extractor, searcher, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func mustTraceRequest(
	t *testing.T,
	ref model.SourceResourceRef,
	direction model.JudicialCitationRequestedDirection,
	limit *int,
) judicialcitationtrace.Request {
	t.Helper()
	request, err := judicialcitationtrace.NewRequest(judicialcitationtrace.RequestValues{
		Ref:           ref,
		Direction:     direction,
		IncomingLimit: limit,
	})
	if err != nil {
		t.Fatalf("request = %v", err)
	}
	return request
}

func mustSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("source = %v", err)
	}
	return source
}

func mustDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date = %v", err)
	}
	return date
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

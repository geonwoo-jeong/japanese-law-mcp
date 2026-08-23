package lawv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/requestpacing"
)

const (
	lawVersionCompareTestLawID          = "322CO0000000016"
	lawVersionCompareTestBeforeRevision = "322CO0000000016_20240101_506CO0000000001"
	lawVersionCompareTestAfterRevision  = "322CO0000000016_20250401_507CO0000000001"
)

func TestLawVersionCompareAdapterComparesTwoVersionsSequentially(t *testing.T) {
	t.Parallel()

	clock := newPacingTestClock()
	starts := make([]time.Time, 0, 2)
	requests := make([]*http.Request, 0, 2)
	adapter := newTestLawVersionCompareAdapter(t, clock, make(chan struct{}, 1), doerFunc(
		func(request *http.Request) (*http.Response, error) {
			requests = append(requests, request.Clone(request.Context()))
			starts = append(starts, clock.Now())
			switch {
			case strings.HasSuffix(request.URL.Path, "/"+lawVersionCompareTestBeforeRevision):
				return response(
					http.StatusOK,
					lawVersionCompareEnvelope(
						lawVersionCompareTestLawID,
						lawVersionCompareTestBeforeRevision,
						"試験法",
						lawVersionBeforeBody(),
					),
					map[string]string{"Content-Type": "application/xml"},
				), nil
			case strings.HasSuffix(request.URL.Path, "/"+lawVersionCompareTestLawID) &&
				request.URL.Query().Get("asof") == "2025-04-01":
				return response(
					http.StatusOK,
					lawVersionCompareEnvelope(
						lawVersionCompareTestLawID,
						lawVersionCompareTestAfterRevision,
						"試験法",
						lawVersionAfterBody(),
					),
					map[string]string{"Content-Type": "application/xml"},
				), nil
			default:
				return nil, fmt.Errorf("unexpected request: %s", request.URL.String())
			}
		},
	))
	request := mustLawVersionCompareRequest(
		t,
		lawVersionCompareTestLawID,
		mustLawVersionSelector(t, lawVersionCompareTestBeforeRevision, nil),
		mustLawVersionSelector(t, "", datePointer(mustDate("2025-04-01"))),
	)

	result, err := adapter.Compare(
		requestpacing.WithScope(context.Background()),
		request,
	)
	if err != nil {
		t.Fatalf("Compare() のエラー = %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("HTTP requests = %d", len(requests))
	}
	if requests[0].URL.Query().Get("asof") != "" ||
		requests[1].URL.Query().Get("asof") != "2025-04-01" {
		t.Fatalf("requests = %#v", requests)
	}
	if len(starts) != 2 || starts[1].Sub(starts[0]) != time.Second {
		t.Fatalf("HTTP start times = %v", starts)
	}

	comparison := result.Data()
	if comparison.LawID() != lawVersionCompareTestLawID ||
		comparison.BeforeArticleCount() != 2 ||
		comparison.AfterArticleCount() != 2 ||
		comparison.AddedCount() != 1 ||
		comparison.RemovedCount() != 1 ||
		comparison.ModifiedCount() != 1 ||
		comparison.UnchangedCount() != 0 ||
		comparison.TotalCount() != 3 {
		t.Fatalf("comparison counts = %#v", comparison)
	}
	items := comparison.Items()
	if len(items) != 3 ||
		items[0].ChangeKind() != model.LawVersionChangeKindModified ||
		items[1].ChangeKind() != model.LawVersionChangeKindAdded ||
		items[2].ChangeKind() != model.LawVersionChangeKindRemoved {
		t.Fatalf("items = %#v", items)
	}
	if reasons := items[0].ChangeReasons(); len(reasons) != 3 ||
		reasons[0] != model.LawVersionChangeReasonLocation ||
		reasons[1] != model.LawVersionChangeReasonText ||
		reasons[2] != model.LawVersionChangeReasonStructure {
		t.Fatalf("modified reasons = %#v", reasons)
	}
	if afterArticle, exists := items[1].After(); !exists {
		t.Fatal("added item に after がありません")
	} else if location, _ := afterArticle.Citation().Location(); location != "main:article=3" {
		t.Fatalf("added citation location = %q", location)
	}
	if beforeArticle, exists := items[2].Before(); !exists {
		t.Fatal("removed item に before がありません")
	} else if location, _ := beforeArticle.Citation().Location(); location != "supplementary:article=2" {
		t.Fatalf("removed citation location = %q", location)
	}

	provenance := result.Provenance()
	if len(provenance) != 3 {
		t.Fatalf("provenance = %#v", provenance)
	}
	final := provenance[len(provenance)-1]
	if final.Transformation() != model.ProvenanceTransformationDerived {
		t.Fatalf("final provenance = %#v", final)
	}
	inputKeys, exists := final.InputKeys()
	if !exists || len(inputKeys) != 2 {
		t.Fatalf("inputKeys = %#v", final)
	}
}

func TestLawVersionCompareAdapterUsesSharedBusyGate(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	attempts := 0
	adapter := newTestLawVersionCompareAdapter(t, newPacingTestClock(), gate, doerFunc(
		func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("呼び出してはならない HTTP")
		},
	))
	request := mustLawVersionCompareRequest(
		t,
		lawVersionCompareTestLawID,
		mustLawVersionSelector(t, lawVersionCompareTestBeforeRevision, nil),
		mustLawVersionSelector(t, lawVersionCompareTestAfterRevision, nil),
	)

	_, err := adapter.Compare(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("HTTP attempts = %d", attempts)
	}
}

func TestLawVersionCompareAdapterTreatsCrossLawRevisionAsNotFound(t *testing.T) {
	t.Parallel()

	adapter := newTestLawVersionCompareAdapter(t, newPacingTestClock(), make(chan struct{}, 1), doerFunc(
		func(request *http.Request) (*http.Response, error) {
			if strings.HasSuffix(request.URL.Path, "/"+lawVersionCompareTestBeforeRevision) {
				return response(
					http.StatusOK,
					lawVersionCompareEnvelope("other-law", lawVersionCompareTestBeforeRevision, "別法", lawVersionBeforeBody()),
					map[string]string{"Content-Type": "application/xml"},
				), nil
			}
			return response(
				http.StatusOK,
				lawVersionCompareEnvelope(lawVersionCompareTestLawID, lawVersionCompareTestAfterRevision, "試験法", lawVersionAfterBody()),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		},
	))
	request := mustLawVersionCompareRequest(
		t,
		lawVersionCompareTestLawID,
		mustLawVersionSelector(t, lawVersionCompareTestBeforeRevision, nil),
		mustLawVersionSelector(t, lawVersionCompareTestAfterRevision, nil),
	)

	_, err := adapter.Compare(requestpacing.WithScope(context.Background()), request)
	if !errors.Is(err, lawversioncompare.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestLawVersionCompareAdapterReturnsEmptyResultForSameRevision(t *testing.T) {
	t.Parallel()

	clock := newPacingTestClock()
	body := lawVersionCompareEnvelope(
		lawVersionCompareTestLawID,
		lawVersionCompareTestBeforeRevision,
		"試験法",
		`<MainProvision><Article Num="1"><Sentence>同一</Sentence></Article></MainProvision>`,
	)
	calls := 0
	adapter := newTestLawVersionCompareAdapter(t, clock, make(chan struct{}, 1), doerFunc(
		func(*http.Request) (*http.Response, error) {
			calls++
			return response(
				http.StatusOK,
				body,
				map[string]string{"Content-Type": "application/xml"},
			), nil
		},
	))
	selector := mustLawVersionSelector(t, lawVersionCompareTestBeforeRevision, nil)
	request := mustLawVersionCompareRequest(
		t,
		lawVersionCompareTestLawID,
		selector,
		selector,
	)

	result, err := adapter.Compare(requestpacing.WithScope(context.Background()), request)
	if err != nil {
		t.Fatalf("Compare() のエラー = %v", err)
	}
	comparison := result.Data()
	if calls != 2 || comparison.TotalCount() != 0 ||
		comparison.UnchangedCount() != 1 || len(comparison.Items()) != 0 {
		t.Fatalf("同版比較 = calls:%d result:%#v", calls, comparison)
	}
}

func TestLawVersionCompareAdapterAppliesProcessingDeadline(t *testing.T) {
	t.Parallel()

	limits := defaultLawVersionCompareLimits()
	limits.processingTimeout = time.Nanosecond
	clock := newPacingTestClock()
	call := 0
	adapter := newTestLawVersionCompareAdapter(
		t,
		clock,
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			revisionID := lawVersionCompareTestBeforeRevision
			if call == 1 {
				revisionID = lawVersionCompareTestAfterRevision
			}
			call++
			return response(
				http.StatusOK,
				lawVersionCompareEnvelope(
					lawVersionCompareTestLawID,
					revisionID,
					"試験法",
					`<MainProvision><Article Num="1"><Sentence>本文</Sentence></Article></MainProvision>`,
				),
				map[string]string{"Content-Type": "application/xml"},
			), nil
		}),
		limits,
	)
	request := mustLawVersionCompareRequest(
		t,
		lawVersionCompareTestLawID,
		mustLawVersionSelector(t, lawVersionCompareTestBeforeRevision, nil),
		mustLawVersionSelector(t, lawVersionCompareTestAfterRevision, nil),
	)

	_, err := adapter.Compare(requestpacing.WithScope(context.Background()), request)
	assertLawVersionCompareSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)
}

func newTestLawVersionCompareAdapter(
	t *testing.T,
	clock *pacingTestClock,
	gate chan struct{},
	doer httpDoer,
	customLimits ...lawVersionCompareLimits,
) *LawVersionCompareAdapter {
	t.Helper()

	client := mustTestClient(t, clientDependencies{
		doer: doer,
		now:  clock.Now,
		sleep: func(ctx context.Context, delay time.Duration) error {
			return clock.Sleep(ctx, delay)
		},
	})
	limits := defaultLawVersionCompareLimits()
	if len(customLimits) != 0 {
		limits = customLimits[0]
	}
	adapter, err := newLawVersionCompareAdapter(lawVersionCompareAdapterDependencies{
		client: client,
		gate:   gate,
		limits: limits,
	})
	if err != nil {
		t.Fatalf("newLawVersionCompareAdapter() のエラー = %v", err)
	}
	return adapter
}

func mustLawVersionCompareRequest(
	t *testing.T,
	lawID string,
	before lawversioncompare.Selector,
	after lawversioncompare.Selector,
) lawversioncompare.Request {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: "law",
		ResourceID:   lawID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できません: %v", err)
	}
	request, err := lawversioncompare.NewRequest(lawversioncompare.RequestValues{
		Resource: ref,
		Before:   before,
		After:    after,
	})
	if err != nil {
		t.Fatalf("lawversioncompare.NewRequest() のエラー = %v", err)
	}
	return request
}

func mustLawVersionSelector(
	t *testing.T,
	revisionID string,
	asOf *model.Date,
) lawversioncompare.Selector {
	t.Helper()

	selector, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: revisionID,
		AsOf:       asOf,
	})
	if err != nil {
		t.Fatalf("lawversioncompare.NewSelector() のエラー = %v", err)
	}
	return selector
}

func datePointer(value model.Date) *model.Date {
	return &value
}

func lawVersionCompareEnvelope(
	lawID string,
	revisionID string,
	title string,
	lawBody string,
) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<law_data_response>
  <law_info>
    <law_type>Act</law_type>
    <law_id>%s</law_id>
    <law_num>令和六年法律第一号</law_num>
    <promulgation_date>2024-01-01</promulgation_date>
  </law_info>
  <revision_info>
    <law_revision_id>%s</law_revision_id>
    <law_title>%s</law_title>
    <amendment_enforcement_date>2024-04-01</amendment_enforcement_date>
  </revision_info>
  <law_full_text>
    <Law Era="Reiwa" Lang="ja" LawType="Act" Num="1" PromulgateDay="01" PromulgateMonth="01" Year="6">
      <LawNum>令和六年法律第一号</LawNum>
      <LawBody>
        <LawTitle>%s</LawTitle>
        %s
      </LawBody>
    </Law>
  </law_full_text>
</law_data_response>`, lawID, revisionID, title, title, lawBody)
}

func lawVersionBeforeBody() string {
	return `<MainProvision>
  <Chapter Num="1">
    <Article Num="1">
      <ArticleCaption>（目的）</ArticleCaption>
      <ArticleTitle>第一条</ArticleTitle>
      <Paragraph Num="1"><ParagraphSentence><Sentence>旧 本文</Sentence></ParagraphSentence></Paragraph>
      <Paragraph Num="2"><TableStruct><Table><TableRow><Article Num="99"><Paragraph Num="1"/></Article></TableRow></Table></TableStruct></Paragraph>
    </Article>
  </Chapter>
</MainProvision>
<SupplProvision>
  <Article Num="2">
    <ArticleTitle>第二条</ArticleTitle>
    <Paragraph Num="1"><ParagraphSentence><Sentence>原始附則</Sentence></ParagraphSentence></Paragraph>
  </Article>
</SupplProvision>
<SupplProvision AmendLawNum="令和六年法律第二号">
  <Article Num="9">
    <ArticleTitle>第九条</ArticleTitle>
    <Paragraph Num="1"><ParagraphSentence><Sentence>改正附則</Sentence></ParagraphSentence></Paragraph>
  </Article>
</SupplProvision>`
}

func lawVersionAfterBody() string {
	return `<MainProvision>
  <Chapter Num="2">
    <Article Num="1">
      <ArticleCaption>（目的）</ArticleCaption>
      <ArticleTitle>第一条</ArticleTitle>
      <Paragraph Num="1"><ParagraphSentence><Sentence>新 本文</Sentence></ParagraphSentence></Paragraph>
    </Article>
  </Chapter>
  <Article Num="3">
    <ArticleTitle>第三条</ArticleTitle>
    <Paragraph Num="1"><ParagraphSentence><Sentence>追加条文</Sentence></ParagraphSentence></Paragraph>
    <Paragraph Num="2"><QuoteStruct><Article Num="77"><Paragraph Num="1"/></Article></QuoteStruct></Paragraph>
  </Article>
</MainProvision>`
}

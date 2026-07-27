package judicialdecisionread_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServicePropagatesTimeoutAndCancellation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		parent  func() (context.Context, context.CancelFunc)
		timeout time.Duration
		want    error
	}{
		"request timeout": {
			parent: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			timeout: 10 * time.Millisecond,
			want:    context.DeadlineExceeded,
		},
		"parent cancellation": {
			parent: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			timeout: time.Second,
			want:    context.Canceled,
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			reader := &recordingJudicialDecisionReader{
				read: func(
					ctx context.Context,
					_ judicialdecisionread.Request,
				) (model.SourcedResource[model.JudicialDecisionDetails], error) {
					<-ctx.Done()
					return model.SourcedResource[model.JudicialDecisionDetails]{},
						ctx.Err()
				},
			}
			service := newJudicialReadService(
				t,
				"courts-provider",
				"courts-source",
				true,
				reader,
				test.timeout,
			)
			request := newJudicialReadRequest(
				t,
				"courts-provider",
				"courts-source",
			)
			ctx, cancel := test.parent()
			defer cancel()

			_, err := service.Read(ctx, request)
			if !errors.Is(err, test.want) {
				t.Fatalf(
					"SOT-ENG-010/SOT-IF-042: error = %v、期待値 = %v",
					err,
					test.want,
				)
			}
		})
	}
}

func TestServicePreservesProviderError(t *testing.T) {
	t.Parallel()

	providerError := errors.New("正規化済みの provider error")
	reader := &recordingJudicialDecisionReader{
		read: func(
			context.Context,
			judicialdecisionread.Request,
		) (model.SourcedResource[model.JudicialDecisionDetails], error) {
			return model.SourcedResource[model.JudicialDecisionDetails]{},
				providerError
		},
	}
	service := newJudicialReadService(
		t,
		"courts-provider",
		"courts-source",
		true,
		reader,
		time.Second,
	)
	request := newJudicialReadRequest(t, "courts-provider", "courts-source")

	_, err := service.Read(context.Background(), request)
	if !errors.Is(err, providerError) {
		t.Fatalf("SOT-IF-042: error = %v", err)
	}
}

func TestServiceReturnsValidatedResultWithExactReferenceRoundTrip(t *testing.T) {
	t.Parallel()

	request := newJudicialReadRequest(
		t,
		"courts-provider",
		"courts-source",
	)
	want := newJudicialDecisionResource(
		t,
		request.Ref(),
		"courts-source",
	)
	calls := 0
	reader := &recordingJudicialDecisionReader{
		read: func(
			ctx context.Context,
			gotRequest judicialdecisionread.Request,
		) (model.SourcedResource[model.JudicialDecisionDetails], error) {
			calls++
			if gotRequest.Ref() != request.Ref() {
				t.Fatalf(
					"SOT-IF-042: provider に渡した ref = %#v",
					gotRequest.Ref(),
				)
			}
			deadline, exists := ctx.Deadline()
			if !exists || time.Until(deadline) <= 0 {
				t.Fatal("SOT-ENG-010: provider context に有効な期限がありません")
			}
			return want, nil
		},
	}
	service := newJudicialReadService(
		t,
		"courts-provider",
		"courts-source",
		true,
		reader,
		time.Second,
	)

	got, err := service.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-042: Read() のエラー = %v", err)
	}
	if calls != 1 {
		t.Fatalf("SOT-IF-042: provider の呼出し回数 = %d", calls)
	}
	if got.Ref() != request.Ref() || got.Ref() != want.Ref() {
		t.Fatalf("SOT-IF-042: 出力 ref = %#v", got.Ref())
	}
	if got.Data().Summary().Source().ID() != request.Ref().Key().SourceID() {
		t.Fatalf(
			"SOT-IF-042: summary.source.id = %q",
			got.Data().Summary().Source().ID(),
		)
	}
}

func TestServiceRejectsProviderResultReferenceMismatch(t *testing.T) {
	t.Parallel()

	request := newJudicialReadRequest(
		t,
		"courts-provider",
		"courts-source",
	)
	mismatchedRef := newJudicialDecisionRef(
		t,
		"courts-provider",
		"courts-source",
		"96746",
		"",
	)
	reader := &recordingJudicialDecisionReader{
		read: func(
			context.Context,
			judicialdecisionread.Request,
		) (model.SourcedResource[model.JudicialDecisionDetails], error) {
			return newJudicialDecisionResource(
				t,
				mismatchedRef,
				"courts-source",
			), nil
		},
	}
	service := newJudicialReadService(
		t,
		"courts-provider",
		"courts-source",
		true,
		reader,
		time.Second,
	)

	if _, err := service.Read(context.Background(), request); err == nil {
		t.Fatal("SOT-IF-042: 入力と異なる出力 ref を受理しました")
	}
}

func TestServiceRejectsProviderResultSummarySourceMismatch(t *testing.T) {
	t.Parallel()

	request := newJudicialReadRequest(
		t,
		"courts-provider",
		"courts-source",
	)
	reader := &recordingJudicialDecisionReader{
		read: func(
			context.Context,
			judicialdecisionread.Request,
		) (model.SourcedResource[model.JudicialDecisionDetails], error) {
			return newJudicialDecisionResource(
				t,
				request.Ref(),
				"different-courts-source",
			), nil
		},
	}
	service := newJudicialReadService(
		t,
		"courts-provider",
		"courts-source",
		true,
		reader,
		time.Second,
	)

	if _, err := service.Read(context.Background(), request); err == nil {
		t.Fatal("SOT-IF-042: ref と異なる summary.source.id を受理しました")
	}
}

func TestServiceRejectsInvalidDependenciesAndInputs(t *testing.T) {
	t.Parallel()

	if _, err := judicialdecisionread.NewService(nil, time.Second); err == nil {
		t.Fatal("nil resolver を受理しました")
	}
	var typedNil *judicialdecisionread.Resolver
	if _, err := judicialdecisionread.NewService(typedNil, time.Second); err == nil {
		t.Fatal("typed nil resolver を受理しました")
	}
	resolver, err := judicialdecisionread.NewResolver(nil)
	if err != nil {
		t.Fatalf("空の Resolver を作成できません: %v", err)
	}
	if _, err := judicialdecisionread.NewService(resolver, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理しました")
	}
	service, err := judicialdecisionread.NewService(resolver, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	var nilContext context.Context
	if _, err := service.Read(nilContext, judicialdecisionread.Request{}); err == nil {
		t.Fatal("nil context を受理しました")
	}
	if _, err := service.Read(
		context.Background(),
		judicialdecisionread.Request{},
	); err == nil {
		t.Fatal("Request のゼロ値を受理しました")
	}

	var typedNilPort *recordingJudicialDecisionReader
	invalidResolver := fixedPortResolver{port: typedNilPort}
	invalidService, err := judicialdecisionread.NewService(
		invalidResolver,
		time.Second,
	)
	if err != nil {
		t.Fatalf("typed nil port 用 Service を作成できません: %v", err)
	}
	request := newJudicialReadRequest(
		t,
		"courts-provider",
		"courts-source",
	)
	if _, err := invalidService.Read(
		context.Background(),
		request,
	); err == nil {
		t.Fatal("resolver が返した typed nil port を呼び出しました")
	}
}

func TestServiceDoesNotCallAnotherProviderAfterResolutionFailure(t *testing.T) {
	t.Parallel()

	calls := 0
	fallback := &recordingJudicialDecisionReader{
		read: func(
			context.Context,
			judicialdecisionread.Request,
		) (model.SourcedResource[model.JudicialDecisionDetails], error) {
			calls++
			return model.SourcedResource[model.JudicialDecisionDetails]{}, nil
		},
	}
	service := newJudicialReadService(
		t,
		"fallback-provider",
		"fallback-source",
		true,
		fallback,
		time.Second,
	)
	request := newJudicialReadRequest(
		t,
		"missing-provider",
		"missing-source",
	)

	_, err := service.Read(context.Background(), request)
	var resolutionError judicialdecisionread.ResolutionError
	if !errors.As(err, &resolutionError) ||
		resolutionError.Code() != model.ErrorCodeInvalidArgument {
		t.Fatalf("SOT-IF-042: error = %T %v", err, err)
	}
	if calls != 0 {
		t.Fatalf("SOT-IF-042: fallback provider の呼出し回数 = %d", calls)
	}
}

func newJudicialReadService(
	t *testing.T,
	providerID string,
	sourceID string,
	enabled bool,
	port judicialdecisionread.Port,
	timeout time.Duration,
) *judicialdecisionread.Service {
	t.Helper()

	resolver, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{
			{
				Descriptor: newJudicialProviderDescriptor(
					t,
					providerID,
					sourceID,
					true,
				),
				Enabled: enabled,
				Port:    port,
			},
		},
	)
	if err != nil {
		t.Fatalf("Resolver を作成できません: %v", err)
	}
	service, err := judicialdecisionread.NewService(resolver, timeout)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	return service
}

func newJudicialDecisionResource(
	t *testing.T,
	ref model.SourceResourceRef,
	summarySourceID string,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()

	summarySource := newJudicialInformationSource(t, summarySourceID)
	decisionDate, err := model.NewDate("2025-03-03")
	if err != nil {
		t.Fatalf("Date を作成できません: %v", err)
	}
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          ref.Key().ResourceID(),
			PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例",
			CaseNumber:          "令和6年（受）第1号",
			DecisionDate:        decisionDate,
			CourtName:           "最高裁判所",
			DetailURL: "https://www.courts.go.jp/hanrei/" +
				ref.Key().ResourceID() +
				"/detail2/index.html",
			Documents: []model.JudicialDocumentLink{},
			Source:    summarySource,
		},
	)
	if err != nil {
		t.Fatalf("JudicialDecisionSummary を作成できません: %v", err)
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: summary},
	)
	if err != nil {
		t.Fatalf("JudicialDecisionDetails を作成できません: %v", err)
	}

	provenanceSource := newJudicialInformationSource(
		t,
		ref.Key().SourceID(),
	)
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         provenanceSource,
		ResourceKey:    ref.Key(),
		URL:            summary.DetailURL(),
		RetrievedAt:    time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       details,
		},
	)
	if err != nil {
		t.Fatalf("SourcedResource を作成できません: %v", err)
	}
	return resource
}

func newJudicialInformationSource(
	t *testing.T,
	sourceID string,
) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	return source
}

type fixedPortResolver struct {
	port judicialdecisionread.Port
	err  error
}

func (r fixedPortResolver) Resolve(
	judicialdecisionread.Request,
) (judicialdecisionread.Port, error) {
	return r.port, r.err
}

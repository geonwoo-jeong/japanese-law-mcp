package legalquerycorpus

import (
	"context"
	"strings"
	"testing"
)

type setSeparationTestRequestValues struct {
	query        string
	hasRef       bool
	providerID   string
	sourceID     string
	resourceType string
	resourceID   string
	hasVersion   bool
	versionID    string
	hasLimit     bool
	limit        int
}

func TestSetSeparationは区別されるDevelopmentとHoldoutを受理する(
	t *testing.T,
) {
	development := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-distinct",
			"development-group",
			setSeparationTestRequest(t, setSeparationTestRequestValues{
				query:    "行政手続法を検索",
				hasLimit: true,
				limit:    10,
			}),
		),
	}
	holdout := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetHoldout,
			"holdout-distinct",
			"holdout-group",
			setSeparationTestRequest(t, setSeparationTestRequestValues{
				query:    "行政事件訴訟法を確認",
				hasLimit: true,
				limit:    20,
			}),
		),
	}

	if err := validateSemanticSetSeparation(development, holdout); err != nil {
		t.Fatalf("SOT-ENG-026: 区別される二集合を拒否した: %v", err)
	}
}

func TestSetSeparationは完全Requestの交差重複を拒否する(t *testing.T) {
	values := setSeparationTestRequestValues{
		query:        "行政手続法の第一条",
		hasRef:       true,
		providerID:   "provider-a",
		sourceID:     "source-a",
		resourceType: "law",
		resourceID:   "resource-a",
		hasVersion:   true,
		versionID:    "version-a",
		hasLimit:     true,
		limit:        12,
	}
	development := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-request-duplicate",
			"development-request-group",
			setSeparationTestRequest(t, values),
		),
	}
	holdout := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetHoldout,
			"holdout-request-duplicate",
			"holdout-request-group",
			setSeparationTestRequest(t, values),
		),
	}

	err := validateSemanticSetSeparation(development, holdout)
	if err == nil {
		t.Fatal("SOT-ENG-026: 完全 request が同じ二集合を受理した")
	}
	if !strings.Contains(err.Error(), "完全 request") {
		t.Fatalf("SOT-ENG-026: 完全 request の衝突として分類しなかった: %v", err)
	}
}

func TestSetSeparationはComparisonKeyの交差重複を拒否する(t *testing.T) {
	tests := []struct {
		name        string
		development string
		holdout     string
	}{
		{
			name:        "表記とUnicode外側空白",
			development: "\u3000民 法（第一条）。\u00a0",
			holdout:     "\u2003民法第一条\u2002",
		},
		{
			name:        "空の比較key",
			development: "\u3000。、\u00a0",
			holdout:     "\u2003・・\u2002",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			development := []SemanticCase{
				setSeparationTestCase(
					t,
					ManifestSetDevelopment,
					"development-comparison-collision",
					"development-comparison-group",
					setSeparationTestRequest(
						t,
						setSeparationTestRequestValues{
							query: test.development,
						},
					),
				),
			}
			holdout := []SemanticCase{
				setSeparationTestCase(
					t,
					ManifestSetHoldout,
					"holdout-comparison-collision",
					"holdout-comparison-group",
					setSeparationTestRequest(
						t,
						setSeparationTestRequestValues{
							query: test.holdout,
						},
					),
				),
			}

			err := validateSemanticSetSeparation(
				development,
				holdout,
			)
			if err == nil {
				t.Fatal("SOT-ENG-026: 同じ ComparisonKey の二集合を受理した")
			}
			if !strings.Contains(err.Error(), "比較キー") {
				t.Fatalf(
					"SOT-ENG-026: ComparisonKey の衝突として分類しなかった: %v",
					err,
				)
			}
		})
	}
}

func TestSetSeparationはLeakageGroupの交差重複を拒否する(t *testing.T) {
	development := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-leakage",
			"shared-leakage-group",
			setSeparationTestRequest(
				t,
				setSeparationTestRequestValues{query: "行政手続法を検索"},
			),
		),
	}
	holdout := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetHoldout,
			"holdout-leakage",
			"shared-leakage-group",
			setSeparationTestRequest(
				t,
				setSeparationTestRequestValues{query: "民法を確認"},
			),
		),
	}

	err := validateSemanticSetSeparation(development, holdout)
	if err == nil {
		t.Fatal("SOT-ENG-026: 同じ leakageGroupId の二集合を受理した")
	}
	if !strings.Contains(err.Error(), "leakageGroupId") {
		t.Fatalf("SOT-ENG-026: leakage group の衝突として分類しなかった: %v", err)
	}
}

func TestManifestIntegrityはSetSeparationを実行する(t *testing.T) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	fixtures[1] = manifestIntegrityTestSemanticFixture(
		t,
		ManifestSetHoldout,
		"holdout-a",
		" 行政 手続法を検索 ",
	)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	filesystem := filesystemReadTestOpen(t, layout)

	_ = manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		filesystem,
		schema,
		manifest,
	)
}

func TestSetSeparationは同じ集合内の重複を許可する(t *testing.T) {
	requestValues := setSeparationTestRequestValues{
		query:    "行政手続法を検索",
		hasLimit: true,
		limit:    10,
	}
	development := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-duplicate-a",
			"development-duplicate-group",
			setSeparationTestRequest(t, requestValues),
		),
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-duplicate-b",
			"development-duplicate-group",
			setSeparationTestRequest(t, requestValues),
		),
	}
	holdout := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetHoldout,
			"holdout-other",
			"holdout-other-group",
			setSeparationTestRequest(
				t,
				setSeparationTestRequestValues{query: "民法を確認"},
			),
		),
	}

	if err := validateSemanticSetSeparation(development, holdout); err != nil {
		t.Fatalf("SOT-ENG-026: 同じ集合内だけの重複を拒否した: %v", err)
	}
}

func TestRawRequestEqualityは全項目と存在情報を比較する(t *testing.T) {
	base := setSeparationTestRequestValues{
		query:        "行政手続法の第一条",
		hasRef:       true,
		providerID:   "provider-a",
		sourceID:     "source-a",
		resourceType: "law",
		resourceID:   "resource-a",
		hasVersion:   true,
		versionID:    "version-a",
		hasLimit:     true,
		limit:        12,
	}
	if !rawRequestsEqual(
		setSeparationTestRequest(t, base),
		setSeparationTestRequest(t, base),
	) {
		t.Fatal("SOT-ENG-026: 全項目が同じ独立 request を不一致とした")
	}
	withoutOptionals := setSeparationTestRequestValues{query: "民法"}
	if !rawRequestsEqual(
		setSeparationTestRequest(t, withoutOptionals),
		setSeparationTestRequest(t, withoutOptionals),
	) {
		t.Fatal("SOT-ENG-026: optional が共に欠落した request を不一致とした")
	}

	tests := map[string]func(setSeparationTestRequestValues) setSeparationTestRequestValues{
		"query原値": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.query = " 行政手続法の第一条 "
			return values
		},
		"ref有無": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.hasRef = false
			return values
		},
		"providerId": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.providerID = "provider-b"
			return values
		},
		"sourceId": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.sourceID = "source-b"
			return values
		},
		"resourceType": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.resourceType = "judicial-decision"
			return values
		},
		"resourceId": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.resourceID = "resource-b"
			return values
		},
		"versionId有無": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.hasVersion = false
			return values
		},
		"versionId値": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.versionID = "version-b"
			return values
		},
		"limitPerAttempt有無": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.hasLimit = false
			return values
		},
		"limitPerAttempt値": func(values setSeparationTestRequestValues) setSeparationTestRequestValues {
			values.limit = 13
			return values
		},
	}

	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			if rawRequestsEqual(
				setSeparationTestRequest(t, base),
				setSeparationTestRequest(t, mutate(base)),
			) {
				t.Fatal("SOT-ENG-026: request の項目差または存在差を見落とした")
			}
		})
	}
}

func TestSetSeparationのErrorはQueryとRef原文を含まない(t *testing.T) {
	const (
		developmentQuery = "secret-development-query"
		holdoutQuery     = "secret-holdout-query"
		providerID       = "secret-provider-ref"
		sourceID         = "secret-source-ref"
		resourceID       = "secret-resource-ref"
		versionID        = "secret-version-ref"
	)
	requestValues := func(query string) setSeparationTestRequestValues {
		return setSeparationTestRequestValues{
			query:        query,
			hasRef:       true,
			providerID:   providerID,
			sourceID:     sourceID,
			resourceType: "law",
			resourceID:   resourceID,
			hasVersion:   true,
			versionID:    versionID,
		}
	}
	development := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetDevelopment,
			"development-secret",
			"shared-secret-group",
			setSeparationTestRequest(t, requestValues(developmentQuery)),
		),
	}
	holdout := []SemanticCase{
		setSeparationTestCase(
			t,
			ManifestSetHoldout,
			"holdout-secret",
			"shared-secret-group",
			setSeparationTestRequest(t, requestValues(holdoutQuery)),
		),
	}

	err := validateSemanticSetSeparation(development, holdout)
	if err == nil {
		t.Fatal("SOT-ENG-026: leakage group 衝突を受理した")
	}
	for _, secret := range []string{
		developmentQuery,
		holdoutQuery,
		providerID,
		sourceID,
		resourceID,
		versionID,
	} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("SOT-ENG-026: error が query/ref 原文を含む: %v", err)
		}
	}
}

func setSeparationTestRequest(
	t *testing.T,
	values setSeparationTestRequestValues,
) Request {
	t.Helper()

	var ref *RequestRef
	if values.hasRef {
		var version *string
		if values.hasVersion {
			versionValue := values.versionID
			version = &versionValue
		}
		key, err := NewRequestKey(RequestKeyValues{
			SourceID:     values.sourceID,
			ResourceType: values.resourceType,
			ResourceID:   values.resourceID,
			VersionID:    version,
		})
		if err != nil {
			t.Fatalf("SOT-ENG-026: test RequestKey error = %v", err)
		}
		refValue, err := NewRequestRef(RequestRefValues{
			ProviderID: values.providerID,
			Key:        key,
		})
		if err != nil {
			t.Fatalf("SOT-ENG-026: test RequestRef error = %v", err)
		}
		ref = &refValue
	}
	var limit *int
	if values.hasLimit {
		limitValue := values.limit
		limit = &limitValue
	}
	request, err := NewRequest(RequestValues{
		Query:           values.query,
		Ref:             ref,
		LimitPerAttempt: limit,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: test Request error = %v", err)
	}
	return request
}

func setSeparationTestCase(
	t *testing.T,
	set ManifestSetKind,
	caseID string,
	leakageGroupID string,
	request Request,
) SemanticCase {
	t.Helper()

	values := validSemanticCaseValues(t)
	values.CaseID = caseID
	values.LeakageGroupID = leakageGroupID
	values.Request = request
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf(
			"SOT-ENG-026: %s の test SemanticCase error = %v",
			set,
			err,
		)
	}
	return semanticCase
}

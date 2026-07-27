package legalquerycorpus

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFixtureIntegrityは原byteのSHA256不一致を拒否する(t *testing.T) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	fixtures[0].declaredSHA = strings.Repeat("0", 64)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)

	_ = manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		fs,
		schema,
		manifest,
	)
}

func TestFixtureIntegrityはChecksumをJSONより先に検証する(t *testing.T) {
	const secret = "secret-invalid-fixture-body"

	fixtures := manifestIntegrityTestBaseFixtures(t)
	fixtures[0].data = []byte(`{"query":"` + secret)
	fixtures[0].declaredSHA = strings.Repeat("f", 64)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)

	err := manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		fs,
		schema,
		manifest,
	)
	lowerError := strings.ToLower(err.Error())
	if !strings.Contains(lowerError, "checksum") &&
		!strings.Contains(lowerError, "sha256") {
		t.Fatalf("SOT-ENG-026: checksum より先に JSON を検証した error = %v", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("SOT-ENG-026: error が不正 fixture の内容を含む: %v", err)
	}
}

func TestFixtureIntegrityは集合ごとのArtifactKindを強制する(t *testing.T) {
	tests := []struct {
		set  ManifestSetKind
		data func(t *testing.T) []byte
	}{
		{
			set: ManifestSetDevelopment,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestExecutionData(t, "execution-wrong-kind")
			},
		},
		{
			set: ManifestSetHoldout,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestExecutionData(t, "execution-wrong-kind")
			},
		},
		{
			set: ManifestSetExecution,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestSemanticData(
					t,
					"development-wrong-kind",
					"kind mismatch",
				)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(string(test.set), func(t *testing.T) {
			fixtures := manifestIntegrityTestBaseFixtures(t)
			fixture := fixtureIntegrityTestFixturePointer(
				t,
				fixtures,
				test.set,
			)
			fixture.data = test.data(t)
			fixture.declaredSHA = ""
			layout, schema, manifest := manifestIntegrityTestPrepare(
				t,
				fixtures,
				"corpus-v1",
				"",
			)
			fs := filesystemReadTestOpen(t, layout)
			_ = manifestIntegrityTestRequireFailure(
				t,
				context.Background(),
				fs,
				schema,
				manifest,
			)
		})
	}
}

func TestFixtureIntegrityはDecodedIdentity不一致を拒否する(t *testing.T) {
	tests := []struct {
		name string
		set  ManifestSetKind
		data func(t *testing.T) []byte
	}{
		{
			name: "schemaVersion",
			set:  ManifestSetDevelopment,
			data: func(t *testing.T) []byte {
				source := validSemanticCase(validLawSearchStep())
				source["caseId"] = "development-a"
				source["schemaVersion"] = float64(2)
				return mustJSONBytes(t, source)
			},
		},
		{
			name: "manifestとfile名",
			set:  ManifestSetDevelopment,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestSemanticData(
					t,
					"development-other",
					"filename mismatch",
				)
			},
		},
		{
			name: "所属集合",
			set:  ManifestSetHoldout,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestSemanticData(
					t,
					"development-other",
					"set mismatch",
				)
			},
		},
		{
			name: "execution file名",
			set:  ManifestSetExecution,
			data: func(t *testing.T) []byte {
				return fixtureIntegrityTestExecutionData(
					t,
					"execution-other",
				)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fixtures := manifestIntegrityTestBaseFixtures(t)
			fixture := fixtureIntegrityTestFixturePointer(
				t,
				fixtures,
				test.set,
			)
			fixture.data = test.data(t)
			fixture.declaredSHA = ""
			layout, schema, manifest := manifestIntegrityTestPrepare(
				t,
				fixtures,
				"corpus-v1",
				"",
			)
			fs := filesystemReadTestOpen(t, layout)
			_ = manifestIntegrityTestRequireFailure(
				t,
				context.Background(),
				fs,
				schema,
				manifest,
			)
		})
	}
}

func TestFixtureIntegrityは取消時にZeroResultを返す(t *testing.T) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := manifestIntegrityTestRequireFailure(
		t,
		ctx,
		fs,
		schema,
		manifest,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-026: context cancellation error = %v", err)
	}
}

func TestFixtureIntegrityのErrorはQueryとFixture本文を含まない(t *testing.T) {
	const secret = "secret-query-must-not-leak"

	fixtures := manifestIntegrityTestBaseFixtures(t)
	fixture := fixtureIntegrityTestFixturePointer(
		t,
		fixtures,
		ManifestSetDevelopment,
	)
	fixture.data = fixtureIntegrityTestSemanticData(
		t,
		"development-other",
		secret,
	)
	fixture.declaredSHA = ""
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)

	err := manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		fs,
		schema,
		manifest,
	)
	if strings.Contains(err.Error(), secret) ||
		strings.Contains(err.Error(), string(fixture.data)) {
		t.Fatalf("SOT-ENG-026: error が query または fixture 本文を含む: %v", err)
	}
}

func fixtureIntegrityTestFixturePointer(
	t *testing.T,
	fixtures []manifestIntegrityTestFixture,
	set ManifestSetKind,
) *manifestIntegrityTestFixture {
	t.Helper()

	for index := range fixtures {
		if fixtures[index].set == set {
			return &fixtures[index]
		}
	}
	t.Fatalf("SOT-ENG-026: test fixture に %s がない", set)
	return nil
}

func fixtureIntegrityTestSemanticData(
	t *testing.T,
	caseID string,
	query string,
) []byte {
	t.Helper()

	source := validSemanticCase(validLawSearchStep())
	source["caseId"] = caseID
	source["leakageGroupId"] = "group-" + caseID
	source["request"].(map[string]any)["query"] = query
	return mustJSONBytes(t, source)
}

func fixtureIntegrityTestExecutionData(
	t *testing.T,
	caseID string,
) []byte {
	t.Helper()

	source := validExecutionCase(
		map[string]any{
			"kind":            "collection_success",
			"sourceItemCount": float64(1),
		},
		validResultExpectation("completed"),
	)
	source["caseId"] = caseID
	return mustJSONBytes(t, source)
}

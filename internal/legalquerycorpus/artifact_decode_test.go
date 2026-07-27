package legalquerycorpus

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestCorpusArtifactDecodeは三種の成果物を正しいvariantへ復元する(
	t *testing.T,
) {
	t.Parallel()

	schema := artifactDecodeTestSchema(t)
	tests := []struct {
		name   string
		source map[string]any
		assert func(t *testing.T, artifact decodedCorpusArtifact)
	}{
		{
			name:   "manifest",
			source: validManifest(),
			assert: func(t *testing.T, artifact decodedCorpusArtifact) {
				t.Helper()
				if artifact.kind != ArtifactKindCorpusManifest {
					t.Fatalf("SOT-ENG-026: decoded kind = %q", artifact.kind)
				}
				if err := artifact.manifest.Validate(); err != nil {
					t.Fatalf("SOT-ENG-026: manifest variant = %v", err)
				}
				artifactDecodeTestRequireZeroSemanticAndExecution(t, artifact)
			},
		},
		{
			name:   "semantic_case",
			source: validSemanticCase(validLawSearchStep()),
			assert: func(t *testing.T, artifact decodedCorpusArtifact) {
				t.Helper()
				if artifact.kind != ArtifactKindSemanticCase {
					t.Fatalf("SOT-ENG-026: decoded kind = %q", artifact.kind)
				}
				if err := artifact.semanticCase.Validate(); err != nil {
					t.Fatalf("SOT-ENG-026: semantic case variant = %v", err)
				}
				artifactDecodeTestRequireZeroManifestAndExecution(t, artifact)
			},
		},
		{
			name: "execution_case",
			source: validExecutionCase(
				map[string]any{
					"kind":            "collection_success",
					"sourceItemCount": float64(1),
				},
				validResultExpectation("completed"),
			),
			assert: func(t *testing.T, artifact decodedCorpusArtifact) {
				t.Helper()
				if artifact.kind != ArtifactKindExecutionCase {
					t.Fatalf("SOT-ENG-026: decoded kind = %q", artifact.kind)
				}
				if err := artifact.executionCase.Validate(); err != nil {
					t.Fatalf("SOT-ENG-026: execution case variant = %v", err)
				}
				artifactDecodeTestRequireZeroManifestAndSemantic(t, artifact)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			artifact := artifactDecodeTestDecode(t, schema, test.source)
			test.assert(t, artifact)
		})
	}
}

func TestCorpusArtifactDecodeは未知のversionとkindを拒否する(t *testing.T) {
	t.Parallel()

	schema := artifactDecodeTestSchema(t)
	tests := map[string]func(map[string]any){
		"未知schemaVersion": func(source map[string]any) {
			source["schemaVersion"] = float64(2)
		},
		"未知artifactKind": func(source map[string]any) {
			source["artifactKind"] = "future_artifact"
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := validSemanticCase(validLawSearchStep())
			mutate(source)
			data := mustJSONBytes(t, source)
			header, err := inspectJSONDocument(data)
			if err != nil {
				t.Fatalf("SOT-ENG-026: 未知 header の構造検査 error = %v", err)
			}
			artifact, err := schema.validateAndDecode(data, header)
			if err == nil {
				t.Fatal("SOT-ENG-026: 未知の version または kind を受理した")
			}
			artifactDecodeTestRequireZeroArtifact(t, artifact)
		})
	}
}

func TestCorpusArtifactDecodeはSchema違反で部分的な成果物を返さない(
	t *testing.T,
) {
	t.Parallel()

	schema := artifactDecodeTestSchema(t)
	tests := map[string]func(map[string]any){
		"未知項目": func(source map[string]any) {
			source["normalizedQuery"] = "schema-before-typed-decode"
		},
		"variant条件不一致": func(source map[string]any) {
			expected := source["expected"].(map[string]any)
			expected["kind"] = "request_error"
			expected["errorCode"] = "invalid_argument"
			expected["field"] = "query"
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := validSemanticCase(validLawSearchStep())
			mutate(source)
			data := mustJSONBytes(t, source)
			header, err := inspectJSONDocument(data)
			if err != nil {
				t.Fatalf("SOT-ENG-026: schema 違反文書の構造検査 error = %v", err)
			}
			artifact, err := schema.validateAndDecode(data, header)
			if err == nil {
				t.Fatal("SOT-ENG-026: JSON Schema が拒否すべき成果物を受理した")
			}
			artifactDecodeTestRequireZeroArtifact(t, artifact)
		})
	}
}

func TestCorpusArtifactDecodeはSchema受理後のConstructor違反を拒否する(
	t *testing.T,
) {
	t.Parallel()

	source := validSemanticCase(validLawSearchStep())
	source["coverageIds"] = []any{
		"typo-insertion",
		"intent-law-search",
	}
	_, resolved := resolvedCorpusSchemaV1(t)
	if err := resolved.Validate(source); err != nil {
		t.Fatalf(
			"SOT-ENG-026: constructor 用の非昇順 fixture を schema が拒否した: %v",
			err,
		)
	}

	schema := artifactDecodeTestSchema(t)
	data := mustJSONBytes(t, source)
	header, err := inspectJSONDocument(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 非昇順 fixture の構造検査 error = %v", err)
	}
	artifact, err := schema.validateAndDecode(data, header)
	if err == nil {
		t.Fatal("SOT-ENG-026: schema 後の typed constructor 違反を受理した")
	}
	artifactDecodeTestRequireZeroArtifact(t, artifact)
}

func TestCorpusArtifactDecodeは同じSchemaから並行decodeできる(t *testing.T) {
	t.Parallel()

	schema := artifactDecodeTestSchema(t)
	sources := []map[string]any{
		validManifest(),
		validSemanticCase(validLawSearchStep()),
		validExecutionCase(
			map[string]any{
				"kind":            "collection_success",
				"sourceItemCount": float64(1),
			},
			validResultExpectation("completed"),
		),
	}
	type decodeInput struct {
		data   []byte
		header jsonDocumentHeader
	}
	inputs := make([]decodeInput, 0, len(sources))
	for _, source := range sources {
		data := mustJSONBytes(t, source)
		header, err := inspectJSONDocument(data)
		if err != nil {
			t.Fatalf("SOT-ENG-026: 並行 decode 用 header error = %v", err)
		}
		inputs = append(inputs, decodeInput{data: data, header: header})
	}

	const goroutines = 32
	const repetitions = 25
	errs := make(chan error, goroutines)
	var wait sync.WaitGroup
	for worker := 0; worker < goroutines; worker++ {
		worker := worker
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iteration := 0; iteration < repetitions; iteration++ {
				input := inputs[(worker+iteration)%len(inputs)]
				artifact, err := schema.validateAndDecode(
					input.data,
					input.header,
				)
				if err != nil {
					errs <- err
					return
				}
				if artifact.kind != input.header.artifactKind {
					errs <- fmt.Errorf(
						"kind = %q, want %q",
						artifact.kind,
						input.header.artifactKind,
					)
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("SOT-ENG-026: 同じ schema の並行 decode error = %v", err)
	}
}

func artifactDecodeTestSchema(t *testing.T) corpusSchemaV1 {
	t.Helper()

	schema, err := newCorpusSchemaV1(schemaRuntimeTestFixedSchema(t))
	if err != nil {
		t.Fatalf("SOT-ENG-026: decode 用 v1 schema を作成できない: %v", err)
	}
	return schema
}

func artifactDecodeTestDecode(
	t *testing.T,
	schema corpusSchemaV1,
	source map[string]any,
) decodedCorpusArtifact {
	t.Helper()

	data := mustJSONBytes(t, source)
	header, err := inspectJSONDocument(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 成果物 header の検査 error = %v", err)
	}
	artifact, err := schema.validateAndDecode(data, header)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 成果物 decode error = %v", err)
	}
	return artifact
}

func artifactDecodeTestRequireZeroArtifact(
	t *testing.T,
	artifact decodedCorpusArtifact,
) {
	t.Helper()

	if !reflect.DeepEqual(artifact, decodedCorpusArtifact{}) {
		t.Fatal("SOT-ENG-026: error と同時に部分的な成果物を返した")
	}
}

func artifactDecodeTestRequireZeroSemanticAndExecution(
	t *testing.T,
	artifact decodedCorpusArtifact,
) {
	t.Helper()

	if !reflect.DeepEqual(artifact.semanticCase, SemanticCase{}) ||
		!reflect.DeepEqual(artifact.executionCase, ExecutionCase{}) {
		t.Fatal("SOT-ENG-026: manifest 以外の variant も復元された")
	}
}

func artifactDecodeTestRequireZeroManifestAndExecution(
	t *testing.T,
	artifact decodedCorpusArtifact,
) {
	t.Helper()

	if !reflect.DeepEqual(artifact.manifest, Manifest{}) ||
		!reflect.DeepEqual(artifact.executionCase, ExecutionCase{}) {
		t.Fatal("SOT-ENG-026: semantic case 以外の variant も復元された")
	}
}

func artifactDecodeTestRequireZeroManifestAndSemantic(
	t *testing.T,
	artifact decodedCorpusArtifact,
) {
	t.Helper()

	if !reflect.DeepEqual(artifact.manifest, Manifest{}) ||
		!reflect.DeepEqual(artifact.semanticCase, SemanticCase{}) {
		t.Fatal("SOT-ENG-026: execution case 以外の variant も復元された")
	}
}

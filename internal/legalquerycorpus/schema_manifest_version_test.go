package legalquerycorpus

import "testing"

func TestCorpusSchemaV1はCorpusVersionに対応する必須ExecutionScenario一覧だけを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := []struct {
		name        string
		version     string
		scenarioIDs []string
	}{
		{
			name:        "corpus-v1は従来の七件",
			version:     "corpus-v1",
			scenarioIDs: legacyRequiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v2は従来の七件",
			version:     "corpus-v2",
			scenarioIDs: legacyRequiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v3は従来の七件",
			version:     "corpus-v3",
			scenarioIDs: legacyRequiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v4は新しい八件",
			version:     "corpus-v4",
			scenarioIDs: requiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v10は新しい八件",
			version:     "corpus-v10",
			scenarioIDs: requiredExecutionScenarioIDs(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			instance := validManifest()
			instance["corpusVersion"] = test.version
			instance["requiredExecutionScenarioIds"] = stringValues(
				test.scenarioIDs...,
			)
			assertSchemaAccepts(t, schema, instance)
		})
	}
}

func TestCorpusSchemaV1はCorpusVersionと不一致の必須ExecutionScenario一覧を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := []struct {
		name        string
		version     string
		scenarioIDs []string
	}{
		{
			name:        "corpus-v1で新しい八件",
			version:     "corpus-v1",
			scenarioIDs: requiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v2で新しい八件",
			version:     "corpus-v2",
			scenarioIDs: requiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v3で新しい八件",
			version:     "corpus-v3",
			scenarioIDs: requiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v4で従来の七件",
			version:     "corpus-v4",
			scenarioIDs: legacyRequiredExecutionScenarioIDs(),
		},
		{
			name:        "corpus-v10で従来の七件",
			version:     "corpus-v10",
			scenarioIDs: legacyRequiredExecutionScenarioIDs(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			instance := validManifest()
			instance["corpusVersion"] = test.version
			instance["requiredExecutionScenarioIds"] = stringValues(
				test.scenarioIDs...,
			)
			assertSchemaRejects(t, schema, instance)
		})
	}
}

func legacyRequiredExecutionScenarioIDs() []string {
	return []string{
		"execution-all-failed",
		"execution-empty",
		"execution-item-budget",
		"execution-nonempty",
		"execution-partial-failure",
		"execution-reversed-completion",
		"execution-timeout",
	}
}

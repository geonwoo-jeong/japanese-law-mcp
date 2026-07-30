package legalquery_test

import (
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCueTaskRelationConstructorKeepsValidatedImmutableValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		query         string
		subjectText   string
		predicateText string
		subjectRole   legalquery.CueSyntaxRole
		predicateRole legalquery.CueSyntaxRole
		kind          legalquery.CueTaskRelationKind
	}{
		"完結した task 表現": {
			query:         "検索してください",
			subjectText:   "検索してください",
			predicateText: "検索してください",
			subjectRole:   legalquery.CueSyntaxRoleTaskExpression,
			predicateRole: legalquery.CueSyntaxRoleTaskExpression,
			kind:          legalquery.CueTaskRelationDirectTask,
		},
		"目的語と述語": {
			query:         "影響グラフを作成してください",
			subjectText:   "影響グラフ",
			predicateText: "作成してください",
			subjectRole:   legalquery.CueSyntaxRoleTaskObject,
			predicateRole: legalquery.CueSyntaxRoleTaskPredicate,
			kind:          legalquery.CueTaskRelationObjectPredicate,
		},
		"短縮された単独 task": {
			query:         " 比較 ",
			subjectText:   "比較",
			predicateText: "比較",
			subjectRole:   legalquery.CueSyntaxRoleTaskObject,
			predicateRole: legalquery.CueSyntaxRoleTaskObject,
			kind:          legalquery.CueTaskRelationStandaloneTask,
		},
	}
	for name, test := range tests {
		name := name
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			subject := mustCueMention(
				t,
				spanForSurface(t, test.query, test.subjectText),
				test.subjectText,
				"core",
				"subject",
			)
			predicate := subject
			if test.predicateText != test.subjectText {
				predicate = mustCueMention(
					t,
					spanForSurface(t, test.query, test.predicateText),
					test.predicateText,
					"core",
					"predicate",
				)
			}
			clauseSpan := mustQuerySpan(t, 0, len(test.query))
			relation, err := legalquery.NewCueTaskRelation(
				legalquery.CueTaskRelationValues{
					Query:         test.query,
					Subject:       subject,
					Predicate:     predicate,
					SubjectRole:   test.subjectRole,
					PredicateRole: test.predicateRole,
					ClauseSpan:    clauseSpan,
					Kind:          test.kind,
				},
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-029: relation のエラー = %v", err)
			}

			subjectRef := relation.Subject()
			predicateRef := relation.Predicate()
			if subjectRef.ProfileID() != "core" ||
				subjectRef.CueID() != "subject" ||
				subjectRef.Span() != subject.Span() ||
				predicateRef.ProfileID() != predicate.ProfileID() ||
				predicateRef.CueID() != predicate.CueID() ||
				predicateRef.Span() != predicate.Span() ||
				relation.ClauseSpan() != clauseSpan ||
				relation.Kind() != test.kind {
				t.Fatalf("SOT-MODEL-029: relation = %#v", relation)
			}
			if err := relation.Validate(); err != nil {
				t.Fatalf("SOT-MODEL-029: Validate() のエラー = %v", err)
			}

			subjectRef = legalquery.CueTaskRelationRef{}
			predicateRef = legalquery.CueTaskRelationRef{}
			if relation.Subject().CueID() != "subject" ||
				relation.Predicate().CueID() != predicate.CueID() {
				t.Fatal("SOT-MODEL-029: getter の参照から relation を変更できました")
			}
		})
	}
}

func TestCueTaskRelationRefConstructorRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	validSpan := mustQuerySpan(t, 0, len("比較"))
	validValues := legalquery.CueTaskRelationRefValues{
		ProfileID: "core",
		CueID:     "task-compare",
		Span:      validSpan,
	}
	ref, err := legalquery.NewCueTaskRelationRef(validValues)
	if err != nil {
		t.Fatalf("SOT-MODEL-029: relation ref のエラー = %v", err)
	}
	if ref.ProfileID() != validValues.ProfileID ||
		ref.CueID() != validValues.CueID ||
		ref.Span() != validSpan {
		t.Fatalf("SOT-MODEL-029: relation ref = %#v", ref)
	}

	tests := map[string]legalquery.CueTaskRelationRefValues{
		"profile ID 欠落": {
			CueID: "task-compare",
			Span:  validSpan,
		},
		"cue ID 欠落": {
			ProfileID: "core",
			Span:      validSpan,
		},
		"span のゼロ値": {
			ProfileID: "core",
			CueID:     "task-compare",
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewCueTaskRelationRef(values); err == nil {
				t.Fatal("SOT-MODEL-029: 不正な relation ref を受理しました")
			}
		})
	}
}

func TestCueTaskRelationConstructorRejectsInvalidRoleKindAndSpans(t *testing.T) {
	t.Parallel()

	objectQuery := "影響グラフを作成してください"
	object := mustCueMention(
		t,
		spanForSurface(t, objectQuery, "影響グラフ"),
		"影響グラフ",
		"core",
		"task-graph",
	)
	predicate := mustCueMention(
		t,
		spanForSurface(t, objectQuery, "作成してください"),
		"作成してください",
		"core",
		"predicate-create",
	)
	valid := legalquery.CueTaskRelationValues{
		Query:         objectQuery,
		Subject:       object,
		Predicate:     predicate,
		SubjectRole:   legalquery.CueSyntaxRoleTaskObject,
		PredicateRole: legalquery.CueSyntaxRoleTaskPredicate,
		ClauseSpan:    mustQuerySpan(t, 0, len(objectQuery)),
		Kind:          legalquery.CueTaskRelationObjectPredicate,
	}
	differentProfilePredicate := mustCueMention(
		t,
		predicate.Span(),
		predicate.Surface(),
		"judicial",
		predicate.CueID(),
	)
	runeMiddleSubject := mustCueMention(
		t,
		mustQuerySpan(t, 1, 1+len("影")),
		"影",
		"core",
		"task-graph",
	)

	tests := map[string]func() legalquery.CueTaskRelationValues{
		"未知の subject role": func() legalquery.CueTaskRelationValues {
			values := valid
			values.SubjectRole = legalquery.CueSyntaxRole("unknown")
			return values
		},
		"none role": func() legalquery.CueTaskRelationValues {
			values := valid
			values.SubjectRole = legalquery.CueSyntaxRoleNone
			return values
		},
		"未知の kind": func() legalquery.CueTaskRelationValues {
			values := valid
			values.Kind = legalquery.CueTaskRelationKind("unknown")
			return values
		},
		"object_predicate の role 不一致": func() legalquery.CueTaskRelationValues {
			values := valid
			values.PredicateRole = legalquery.CueSyntaxRoleTaskExpression
			return values
		},
		"異なる profile": func() legalquery.CueTaskRelationValues {
			values := valid
			values.Predicate = differentProfilePredicate
			return values
		},
		"subject と predicate の逆順": func() legalquery.CueTaskRelationValues {
			values := valid
			values.Subject = predicate
			values.Predicate = object
			return values
		},
		"助詞が を ではない": func() legalquery.CueTaskRelationValues {
			query := "影響グラフに作成してください"
			values := valid
			values.Query = query
			values.Subject = mustCueMention(
				t,
				spanForSurface(t, query, "影響グラフ"),
				"影響グラフ",
				"core",
				"task-graph",
			)
			values.Predicate = mustCueMention(
				t,
				spanForSurface(t, query, "作成してください"),
				"作成してください",
				"core",
				"predicate-create",
			)
			values.ClauseSpan = mustQuerySpan(t, 0, len(query))
			return values
		},
		"predicate の後ろに非空白": func() legalquery.CueTaskRelationValues {
			query := objectQuery + "規定"
			values := valid
			values.Query = query
			values.ClauseSpan = mustQuerySpan(t, 0, len(query))
			return values
		},
		"clause が subject を含まない": func() legalquery.CueTaskRelationValues {
			values := valid
			values.ClauseSpan = predicate.Span()
			return values
		},
		"query の UTF-8 rune 途中": func() legalquery.CueTaskRelationValues {
			values := valid
			values.Subject = runeMiddleSubject
			return values
		},
		"cue が query の範囲外": func() legalquery.CueTaskRelationValues {
			values := valid
			values.Subject = mustCueMention(
				t,
				mustQuerySpan(t, 0, len(objectQuery)+1),
				objectQuery+"x",
				"core",
				"task-graph",
			)
			return values
		},
		"clause が query の範囲外": func() legalquery.CueTaskRelationValues {
			values := valid
			values.ClauseSpan = mustQuerySpan(t, 0, len(objectQuery)+1)
			return values
		},
	}
	for name, makeValues := range tests {
		name := name
		makeValues := makeValues
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewCueTaskRelation(makeValues()); err == nil {
				t.Fatal("SOT-MODEL-029: 不正な relation を受理しました")
			}
		})
	}
}

func TestCueTaskRelationGetterは並行呼出しで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	query := "影響グラフを作成してください"
	subject := mustCueMention(
		t,
		spanForSurface(t, query, "影響グラフ"),
		"影響グラフ",
		"core",
		"task-graph",
	)
	predicate := mustCueMention(
		t,
		spanForSurface(t, query, "作成してください"),
		"作成してください",
		"core",
		"predicate-create",
	)
	relation := mustCueTaskRelation(
		t,
		query,
		subject,
		predicate,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate,
		mustQuerySpan(t, 0, len(query)),
		legalquery.CueTaskRelationObjectPredicate,
	)

	const workers = 16
	start := make(chan struct{})
	errors := make(chan string, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			<-start
			if err := relation.Validate(); err != nil {
				errors <- err.Error()
				return
			}
			subjectRef := relation.Subject()
			subjectRef = legalquery.CueTaskRelationRef{}
			if subjectRef.CueID() != "" || relation.Subject().CueID() != "task-graph" {
				errors <- "getter から relation が変更されました"
			}
		}()
	}
	close(start)
	group.Wait()
	close(errors)

	for message := range errors {
		t.Fatalf("SOT-MODEL-029: %s", message)
	}
}

func TestCueTaskRelationConstructorRejectsInvalidDirectAndStandaloneTask(t *testing.T) {
	t.Parallel()

	directQuery := "検索してください"
	direct := mustCueMention(
		t,
		spanForSurface(t, directQuery, directQuery),
		directQuery,
		"core",
		"task-search",
	)
	otherDirect := mustCueMention(
		t,
		direct.Span(),
		direct.Surface(),
		"core",
		"task-read",
	)
	standaloneQuery := " 比較 "
	standalone := mustCueMention(
		t,
		spanForSurface(t, standaloneQuery, "比較"),
		"比較",
		"core",
		"task-compare",
	)

	tests := map[string]legalquery.CueTaskRelationValues{
		"direct_task の異なる参照": {
			Query: directQuery, Subject: direct, Predicate: otherDirect,
			SubjectRole:   legalquery.CueSyntaxRoleTaskExpression,
			PredicateRole: legalquery.CueSyntaxRoleTaskExpression,
			ClauseSpan:    mustQuerySpan(t, 0, len(directQuery)),
			Kind:          legalquery.CueTaskRelationDirectTask,
		},
		"direct_task の role 不一致": {
			Query: directQuery, Subject: direct, Predicate: direct,
			SubjectRole:   legalquery.CueSyntaxRoleTaskObject,
			PredicateRole: legalquery.CueSyntaxRoleTaskObject,
			ClauseSpan:    mustQuerySpan(t, 0, len(directQuery)),
			Kind:          legalquery.CueTaskRelationDirectTask,
		},
		"standalone_task の role 不一致": {
			Query: standaloneQuery, Subject: standalone, Predicate: standalone,
			SubjectRole:   legalquery.CueSyntaxRoleTaskExpression,
			PredicateRole: legalquery.CueSyntaxRoleTaskExpression,
			ClauseSpan:    mustQuerySpan(t, 0, len(standaloneQuery)),
			Kind:          legalquery.CueTaskRelationStandaloneTask,
		},
		"standalone_task の節外側に非空白": {
			Query: "規定を比較", Subject: mustCueMention(
				t,
				spanForSurface(t, "規定を比較", "比較"),
				"比較",
				"core",
				"task-compare",
			), Predicate: mustCueMention(
				t,
				spanForSurface(t, "規定を比較", "比較"),
				"比較",
				"core",
				"task-compare",
			),
			SubjectRole:   legalquery.CueSyntaxRoleTaskObject,
			PredicateRole: legalquery.CueSyntaxRoleTaskObject,
			ClauseSpan:    mustQuerySpan(t, 0, len("規定を比較")),
			Kind:          legalquery.CueTaskRelationStandaloneTask,
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewCueTaskRelation(values); err == nil {
				t.Fatal("SOT-MODEL-029: 不正な direct/standalone relation を受理しました")
			}
		})
	}
}

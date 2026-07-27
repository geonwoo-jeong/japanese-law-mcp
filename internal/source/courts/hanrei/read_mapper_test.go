package hanrei

import "testing"

func TestReadFieldValueRejectsConflictingAliases(t *testing.T) {
	t.Parallel()
	_, _, err := readFieldValue(
		[]readField{
			{label: "裁判所名", value: "最高裁判所"},
			{label: "法廷名", value: "第一小法廷"},
		},
		"裁判所名",
		"法廷名",
	)
	if err == nil {
		t.Fatal("相反する別名を受理した")
	}
}

func TestOptionalReadDateFieldParsesAndRejectsInvalidValue(t *testing.T) {
	t.Parallel()
	value, err := optionalReadDateField(
		[]readField{{label: "原審裁判年月日", value: "令和7年10月16日"}},
		"原審裁判年月日",
	)
	if err != nil || value == nil || value.String() != "2025-10-16" {
		t.Fatalf("lowerCourtDecisionDate = %#v", value)
	}
	value, err = optionalReadDateField(
		[]readField{{label: "原審裁判年月日", value: "不明"}},
		"原審裁判年月日",
	)
	if err == nil || value != nil {
		t.Fatalf("不正日付を受理した: %#v, %v", value, err)
	}
}

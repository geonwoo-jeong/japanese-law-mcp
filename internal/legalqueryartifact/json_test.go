package legalqueryartifact

import "testing"

func TestInspectJSONObjectRejectsDuplicateNullTrailingAndLimits(t *testing.T) {
	t.Parallel()

	limits := JSONLimits{Depth: 4, Values: 8, RejectNull: true}
	if err := InspectJSONObject([]byte("{\"a\":1}"), limits); err != nil {
		t.Fatalf("valid JSON を拒否しました: %v", err)
	}
	tests := map[string][]byte{
		"duplicate": []byte("{\"a\":1,\"a\":2}"),
		"null":      []byte("{\"a\":null}"),
		"trailing":  []byte("{\"a\":1}[]"),
		"top array": []byte("[1]"),
		"invalid":   {0xff},
	}
	for name, data := range tests {
		if err := InspectJSONObject(data, limits); err == nil {
			t.Fatalf("%s を受理しました", name)
		}
	}
	if err := InspectJSONObject([]byte("{\"a\":{\"b\":{\"c\":{\"d\":1}}}}"), JSONLimits{
		Depth:      3,
		Values:     8,
		RejectNull: true,
	}); err == nil {
		t.Fatal("depth 超過を受理しました")
	}
	if err := InspectJSONObject([]byte("{\"a\":[1,2,3,4]}"), JSONLimits{
		Depth:      4,
		Values:     4,
		RejectNull: true,
	}); err == nil {
		t.Fatal("value 数超過を受理しました")
	}
}

func TestDecodeClosedRejectsUnknownAndTrailing(t *testing.T) {
	t.Parallel()

	type document struct {
		Name string `json:"name"`
	}
	var valid document
	if err := DecodeClosed([]byte("{\"name\":\"ok\"}"), &valid); err != nil {
		t.Fatalf("valid JSON を decode できません: %v", err)
	}
	if valid.Name != "ok" {
		t.Fatalf("name = %q", valid.Name)
	}
	for _, data := range [][]byte{
		[]byte("{\"name\":\"ok\",\"unknown\":true}"),
		[]byte("{\"name\":\"ok\"}{}"),
	} {
		var dst document
		if err := DecodeClosed(data, &dst); err == nil {
			t.Fatalf("invalid decode を受理しました: %s", data)
		}
	}
}

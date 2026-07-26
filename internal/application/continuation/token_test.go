package continuation

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestIssueAndVerifyGoldenV1Token(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016: golden token の発行エラー = %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		t.Fatalf("SOT-IF-016: token envelope = %q", token)
	}
	payload, _ := decodeTokenPayload(t, token)
	wantPayload := []byte(
		`{"adapterContractVersion":"1.0.0","capabilityId":"law.search",` +
			`"conditionFingerprint":"` + goldenConditionFingerprint + `",` +
			`"configFingerprint":"` + goldenConfigFingerprint + `",` +
			`"expiresAt":1767226500,"issuedAt":1767225600,"limit":20,"majorVersion":1,` +
			`"position":{"cursor":"next-20"},"providerId":"` + goldenProviderID + `",` +
			`"snapshot":"2026-07-26T00:00:00Z","sort":["lawId","asc"]}`,
	)
	if !bytes.Equal(payload, wantPayload) {
		t.Fatalf("SOT-IF-016: golden payload = %s、期待値 = %s", payload, wantPayload)
	}
	if parts[2] != goldenEnvelopeMAC {
		t.Fatalf("SOT-IF-016: golden MAC = %s、期待値 = %s", parts[2], goldenEnvelopeMAC)
	}
	independentMAC := hmacForDomain(
		fixedContinuationKey(),
		"continuation-token-v1",
		[]byte(parts[0]+"."+parts[1]),
	)
	if got := base64.RawURLEncoding.EncodeToString(independentMAC); got != parts[2] {
		t.Fatalf("SOT-IF-016: 独立計算した token MAC = %s、実値 = %s", got, parts[2])
	}

	cursor, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
	if err != nil {
		t.Fatalf("SOT-IF-016: golden token の検証エラー = %v", err)
	}
	if got := string(cursor.Position().Bytes()); got != `{"cursor":"next-20"}` {
		t.Fatalf("SOT-IF-016: cursor position = %s", got)
	}
	snapshot, ok := cursor.Snapshot()
	if !ok || string(snapshot.Bytes()) != `"2026-07-26T00:00:00Z"` {
		t.Fatalf("SOT-IF-016: cursor snapshot = %s, %t", snapshot.Bytes(), ok)
	}
	sortMarker, ok := cursor.Sort()
	if !ok || string(sortMarker.Bytes()) != `["lawId","asc"]` {
		t.Fatalf("SOT-IF-016: cursor sort = %s, %t", sortMarker.Bytes(), ok)
	}
	if !cursor.IssuedAt().Equal(goldenNow) ||
		!cursor.ExpiresAt().Equal(goldenNow.Add(15*time.Minute)) {
		t.Fatalf(
			"SOT-IF-016: cursor の時刻 = %s, %s",
			cursor.IssuedAt(),
			cursor.ExpiresAt(),
		)
	}
}

func TestOptionalMarkersAreOmittedAndAbsentFromCursor(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	input := goldenIssueInput(t, fixture)
	input.Position = mustJSONValue(t, []byte(`20`))
	input.Snapshot = nil
	input.Sort = nil
	input.TTL = input.MaxTTL

	token, err := fixture.manager.Issue(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: optional marker なしの token 発行エラー = %v", err)
	}
	_, payload := decodeTokenPayload(t, token)
	if len(payload) != 10 {
		t.Fatalf("SOT-IF-016: optional marker なしの payload key 数 = %d", len(payload))
	}
	if _, exists := payload["snapshot"]; exists {
		t.Fatalf("SOT-IF-016: 未指定の snapshot が payload に存在する")
	}
	if _, exists := payload["sort"]; exists {
		t.Fatalf("SOT-IF-016: 未指定の sort が payload に存在する")
	}

	cursor, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
	if err != nil {
		t.Fatalf("SOT-IF-016: optional marker なしの token 検証エラー = %v", err)
	}
	if string(cursor.Position().Bytes()) != `20` {
		t.Fatalf("SOT-IF-016: scalar position = %s", cursor.Position().Bytes())
	}
	if _, ok := cursor.Snapshot(); ok {
		t.Fatalf("SOT-IF-016: 未指定の snapshot が cursor に存在する")
	}
	if _, ok := cursor.Sort(); ok {
		t.Fatalf("SOT-IF-016: 未指定の sort が cursor に存在する")
	}
}

func TestCursorValuesAreImmutableAndSafeToFormat(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	token, err := fixture.manager.Issue(goldenIssueInput(t, fixture))
	if err != nil {
		t.Fatalf("SOT-IF-016: token の発行エラー = %v", err)
	}
	cursor, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
	if err != nil {
		t.Fatalf("SOT-IF-016: token の検証エラー = %v", err)
	}

	position := cursor.Position()
	positionBytes := position.Bytes()
	positionBytes[0] = '['
	snapshot, _ := cursor.Snapshot()
	snapshotBytes := snapshot.Bytes()
	snapshotBytes[0] = '['
	sortMarker, _ := cursor.Sort()
	sortBytes := sortMarker.Bytes()
	sortBytes[0] = '{'
	if string(cursor.Position().Bytes()) != `{"cursor":"next-20"}` {
		t.Fatalf("SOT-IF-016: position が getter 経由で変更された")
	}
	gotSnapshot, _ := cursor.Snapshot()
	if string(gotSnapshot.Bytes()) != `"2026-07-26T00:00:00Z"` {
		t.Fatalf("SOT-IF-016: snapshot が getter 経由で変更された")
	}
	gotSort, _ := cursor.Sort()
	if string(gotSort.Bytes()) != `["lawId","asc"]` {
		t.Fatalf("SOT-IF-016: sort が getter 経由で変更された")
	}

	for _, format := range []string{"%v", "%+v", "%#v"} {
		formatted := fmt.Sprintf(format, cursor)
		if strings.Contains(formatted, "next-20") ||
			strings.Contains(formatted, "2026-07-26T00:00:00Z") {
			t.Fatalf("SOT-IF-016: cursor の formatting が marker を公開した: %s", formatted)
		}
	}
}

func TestIssueRejectsInvalidBindingLifetimeAndPosition(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	tests := map[string]func(*IssueInput){
		"provider の zero value": func(input *IssueInput) {
			input.Provider = model.ProviderDescriptor{}
		},
		"capability の zero value": func(input *IssueInput) {
			input.Capability = model.ProviderCapability{}
		},
		"宣言されていない capability": func(input *IssueInput) {
			input.Capability = newTestCapability(t, "law.document.read", 1)
		},
		"limit が zero": func(input *IssueInput) {
			input.Limit = 0
		},
		"limit が負": func(input *IssueInput) {
			input.Limit = -1
		},
		"position の zero value": func(input *IssueInput) {
			input.Position = JSONValue{}
		},
		"TTL が zero": func(input *IssueInput) {
			input.TTL = 0
		},
		"TTL が負": func(input *IssueInput) {
			input.TTL = -time.Second
		},
		"TTL が上限超過": func(input *IssueInput) {
			input.TTL = input.MaxTTL + time.Second
		},
		"有効期限上限が zero": func(input *IssueInput) {
			input.MaxTTL = 0
		},
	}
	for name, change := range tests {
		change := change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := goldenIssueInput(t, fixture)
			change(&input)
			token, err := fixture.manager.Issue(input)
			if err == nil || token != "" {
				t.Fatalf("SOT-IF-016: 不正な発行入力の結果 = %q, %v", token, err)
			}
			if errors.Is(err, ErrInvalidToken) {
				t.Fatalf("SOT-IF-016: 内部の発行入力エラーを token 起因として返した: %v", err)
			}
		})
	}
}

func TestIssueRejectsTokenLargerThan4096BytesWithoutLeakingPosition(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	secretPosition := "外部取得位置を公開しない-" + strings.Repeat("x", 5000)
	input := goldenIssueInput(t, fixture)
	input.Position = mustJSONValue(t, []byte(`"`+secretPosition+`"`))
	token, err := fixture.manager.Issue(input)
	if err == nil || token != "" {
		t.Fatalf("SOT-IF-016: 4096 byte を超える token の発行結果 = %d byte, %v", len(token), err)
	}
	if strings.Contains(err.Error(), secretPosition) {
		t.Fatalf("SOT-IF-016: token 発行エラーが position を公開した")
	}
}

func TestBindingUsesCapabilityIDAndMajorVersionOnly(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	metadataVariant, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           fixture.capability.ID(),
		MajorVersion: fixture.capability.MajorVersion(),
		Level:        fixture.capability.Level(),
		Stability:    model.CapabilityStabilityExperimental,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-013: metadata variant の作成エラー = %v", err)
	}
	input := goldenIssueInput(t, fixture)
	input.Capability = metadataVariant

	token, err := fixture.manager.Issue(input)
	if err != nil {
		t.Fatalf("SOT-ARCH-012/SOT-IF-016: 同じ capability binding の発行エラー = %v", err)
	}
	verifyInput := goldenVerifyInput(fixture, token)
	verifyInput.Capability = metadataVariant
	if _, err := fixture.manager.Verify(verifyInput); err != nil {
		t.Fatalf("SOT-ARCH-012/SOT-IF-016: 同じ capability binding の検証エラー = %v", err)
	}
}

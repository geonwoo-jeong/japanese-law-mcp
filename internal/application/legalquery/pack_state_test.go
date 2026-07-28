package legalquery

import "testing"

func TestStaticPackStateは採用と有効状態を区別する(t *testing.T) {
	t.Parallel()

	state, err := NewStaticPackState(
		[]string{"judicial-cases", "tax"},
		[]string{"judicial-cases"},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-019: pack state を作成できません: %v", err)
	}
	var port PackState = state

	tests := []struct {
		name        string
		packID      string
		wantEnabled bool
		wantAdopted bool
	}{
		{
			name:        "採用済みかつ有効",
			packID:      "judicial-cases",
			wantEnabled: true,
			wantAdopted: true,
		},
		{
			name:        "採用済みだが無効",
			packID:      "tax",
			wantEnabled: false,
			wantAdopted: true,
		},
		{
			name:        "未知のpack",
			packID:      "unknown-pack",
			wantEnabled: false,
			wantAdopted: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			enabled, adopted := port.State(test.packID)
			if enabled != test.wantEnabled || adopted != test.wantAdopted {
				t.Fatalf(
					"State(%q) = enabled:%t adopted:%t; want enabled:%t adopted:%t",
					test.packID,
					enabled,
					adopted,
					test.wantEnabled,
					test.wantAdopted,
				)
			}
		})
	}
}

func TestStaticPackStateは構成矛盾をfailClosedで拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		adopted []string
		enabled []string
	}{
		{
			name:    "未採用packの有効化",
			adopted: []string{"judicial-cases"},
			enabled: []string{"tax"},
		},
		{
			name:    "採用packの重複",
			adopted: []string{"tax", "tax"},
		},
		{
			name:    "有効packの重複",
			adopted: []string{"tax"},
			enabled: []string{"tax", "tax"},
		},
		{
			name:    "不正なpack識別子",
			adopted: []string{"Judicial Cases"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewStaticPackState(test.adopted, test.enabled); err == nil {
				t.Fatal("SOT-ARCH-019: 矛盾する pack state を受理しました")
			}
		})
	}
}

func TestStaticPackStateは入力sliceから変更できない(t *testing.T) {
	t.Parallel()

	adopted := []string{"judicial-cases", "tax"}
	enabled := []string{"judicial-cases"}
	state, err := NewStaticPackState(adopted, enabled)
	if err != nil {
		t.Fatalf("pack state を作成できません: %v", err)
	}
	adopted[0] = "labor"
	enabled[0] = "labor"

	enabledJudicial, adoptedJudicial := state.State("judicial-cases")
	enabledLabor, adoptedLabor := state.State("labor")
	if !enabledJudicial || !adoptedJudicial ||
		enabledLabor || adoptedLabor {
		t.Fatal("SOT-ENG-025: 構築後の入力 slice から pack state を変更できました")
	}
}

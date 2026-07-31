package judicialcases

import "fmt"

func newJudicialEvidenceProfile(profile *Profile) (*Profile, error) {
	if profile == nil {
		return nil, fmt.Errorf("judicial-cases profile は必須です")
	}
	next, err := newCueTaskRelationV2Profile(profile)
	if err != nil {
		return nil, err
	}
	margin, present := next.metadata.Selection().BranchRetentionMargin()
	if next.metadata.SchemaVersion() != 2 || !present || margin <= 0 {
		return nil, fmt.Errorf(
			"judicial evidence profile は schema version 2 と branchRetentionMargin が必要です",
		)
	}
	next.intentEvidenceMode = cueIntentEvidenceJudicial
	return next, nil
}

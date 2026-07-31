package defaultprofile

import "fmt"

// ProfileIdentity は、composition root 順の profile identity を保持する。
type ProfileIdentity struct {
	profileID      string
	profileVersion string
	cueSetVersion  string
}

// ProfileID は profile ID を返す。
func (p ProfileIdentity) ProfileID() string { return p.profileID }

// ProfileVersion は profile の意味版を返す。
func (p ProfileIdentity) ProfileVersion() string { return p.profileVersion }

// CueSetVersion は cue 成果物版を返す。
func (p ProfileIdentity) CueSetVersion() string { return p.cueSetVersion }

// Identity は、標準 evaluator が実際に構成した profile set identity を保持する。
type Identity struct {
	profileSetID       string
	profileSetVersion  string
	rankingVersion     string
	compositionVersion string
	profiles           []ProfileIdentity
}

// ProfileSetID は固定 set ID を返す。
func (i Identity) ProfileSetID() string { return i.profileSetID }

// ProfileSetVersion は構成済み set の不透明な版を返す。
func (i Identity) ProfileSetVersion() string { return i.profileSetVersion }

// RankingVersion は profile 共通の順位校正版を返す。
func (i Identity) RankingVersion() string { return i.rankingVersion }

// CompositionVersion は候補合成規則版を返す。
func (i Identity) CompositionVersion() string { return i.compositionVersion }

// Profiles は composition root の固定順を複製して返す。
func (i Identity) Profiles() []ProfileIdentity {
	return append([]ProfileIdentity(nil), i.profiles...)
}

// Identity は、production と共有する組込み composition root の identity を返す。
func (e *Evaluator) Identity() (Identity, error) {
	if e == nil {
		return Identity{}, fmt.Errorf("default profile evaluator は nil にできません")
	}
	profiles := e.planning.ProfileMetadata()
	identities := make([]ProfileIdentity, 0, len(profiles))
	for _, profile := range profiles {
		identities = append(identities, ProfileIdentity{
			profileID:      profile.ProfileID(),
			profileVersion: profile.ProfileVersion(),
			cueSetVersion:  profile.CueSetVersion(),
		})
	}
	profileSet := e.planning.Profiles()
	if err := profileSet.Validate(); err != nil {
		return Identity{}, fmt.Errorf("default profile set が不正です: %w", err)
	}
	return Identity{
		profileSetID:       "default",
		profileSetVersion:  profileSet.ProfileVersion(),
		rankingVersion:     profileSet.RankingVersion(),
		compositionVersion: profileSet.CompositionVersion(),
		profiles:           identities,
	}, nil
}

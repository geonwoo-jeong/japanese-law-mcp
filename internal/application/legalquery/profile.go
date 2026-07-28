package legalquery

import "fmt"

// QueryProfile は、位置付き前処理事実から候補 contribution を生成する。
type QueryProfile interface {
	Metadata() QueryProfileMetadata
	CueVocabulary() []CueVocabularyEntry
	Generate(
		CandidateGenerationInput,
		CandidateIDScope,
	) (CandidateGeneration, error)
}

// CollectProfileCandidates は、前処理結果を profile 入力へ変換して候補を回収する。
func CollectProfileCandidates(
	profile QueryProfile,
	preprocessed PreprocessResult,
	scope CandidateIDScope,
) (CandidateGeneration, error) {
	if isNilInterfaceValue(profile) {
		return CandidateGeneration{}, fmt.Errorf("profile は必須です")
	}
	metadata := profile.Metadata()
	return collectProfileCandidatesForMetadata(
		profile,
		metadata,
		preprocessed,
		scope,
	)
}

func collectProfileCandidatesForMetadata(
	profile QueryProfile,
	metadata QueryProfileMetadata,
	preprocessed PreprocessResult,
	scope CandidateIDScope,
) (CandidateGeneration, error) {
	if err := metadata.Validate(); err != nil {
		return CandidateGeneration{}, fmt.Errorf("profile metadata が有効ではありません: %w", err)
	}
	input, err := NewCandidateGenerationInput(preprocessed)
	if err != nil {
		return CandidateGeneration{}, err
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		return CandidateGeneration{}, err
	}
	if err := generation.Validate(); err != nil {
		return CandidateGeneration{}, fmt.Errorf(
			"profile generation が有効ではありません: %w",
			err,
		)
	}
	if generation.ProfileID() != metadata.ProfileID() {
		return CandidateGeneration{}, fmt.Errorf(
			"profile が metadata と異なる profileId を返しました",
		)
	}
	if generation.ProfileVersion() != metadata.ProfileVersion() {
		return CandidateGeneration{}, fmt.Errorf(
			"profile が metadata と異なる profileVersion を返しました",
		)
	}
	if generation.RankingVersion() != metadata.RankingVersion() {
		return CandidateGeneration{}, fmt.Errorf(
			"profile が metadata と異なる rankingVersion を返しました",
		)
	}
	currentMetadata := profile.Metadata()
	if err := currentMetadata.Validate(); err != nil {
		return CandidateGeneration{}, fmt.Errorf(
			"profile metadata が候補生成中に無効になりました: %w",
			err,
		)
	}
	if queryProfileMetadataSignature(currentMetadata) !=
		queryProfileMetadataSignature(metadata) {
		return CandidateGeneration{}, fmt.Errorf(
			"profile metadata が候補生成中に変更されました",
		)
	}
	return generation, nil
}

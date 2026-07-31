package legalquerycandidateeval

import "fmt"

func validateCandidateContent(document CandidateContentManifest) error {
	if document.ArtifactKind != ArtifactKindCandidateContent ||
		document.SchemaVersion != SchemaVersionV2 ||
		!candidateContentIDPattern.MatchString(document.CandidateContentID) {
		return fmt.Errorf("candidate content manifest の版または ID が不正です")
	}
	if err := validateProfileSet(document.ProfileSet); err != nil {
		return err
	}
	if err := validateProfileArtifacts(document.ProfileArtifacts); err != nil {
		return err
	}
	if err := validateLexiconArtifacts(document.LexiconArtifacts); err != nil {
		return err
	}
	if err := validateComposition(document.Composition, document.ProfileSet, document.ProfileArtifacts); err != nil {
		return err
	}
	if err := validateSemanticSourceSet(document.SemanticSourceSet); err != nil {
		return err
	}
	expectedID, err := CanonicalCandidateContentID(document)
	if err != nil || document.CandidateContentID != expectedID {
		return fmt.Errorf("candidateContentId が canonical tuple digest と一致しません")
	}
	return nil
}

func validateProfileSet(profileSet ProfileSetIdentity) error {
	fields := []struct{ name, value string }{
		{name: "profileSetId", value: profileSet.ProfileSetID},
		{name: "profileSetVersion", value: profileSet.ProfileSetVersion},
		{name: "rankingVersion", value: profileSet.RankingVersion},
	}
	for _, field := range fields {
		if err := validateMachineString(field.name, field.value, 128); err != nil {
			return err
		}
	}
	return nil
}

func validateProfileArtifacts(profiles []ProfileArtifact) error {
	//nolint:staticcheck // SOT-ENG-038: JSON null と空配列を別状態として閉じて検証する。
	if profiles == nil || len(profiles) < 1 || len(profiles) > 16 {
		return fmt.Errorf("profileArtifacts の件数が不正です")
	}
	allowed := []string{"core", "judicial-cases"}
	lastAllowed := -1
	for _, profile := range profiles {
		position := stringPosition(allowed, profile.ProfileID)
		if position < 0 || position <= lastAllowed {
			return fmt.Errorf("profileArtifacts の profileId または順序が不正です")
		}
		lastAllowed = position
		if err := validateProfileArtifact(profile); err != nil {
			return err
		}
	}
	return nil
}

func validateProfileArtifact(profile ProfileArtifact) error {
	if err := validateMachineString("profileVersion", profile.ProfileVersion, 128); err != nil {
		return err
	}
	if profile.MetadataSchemaVersion < 1 {
		return fmt.Errorf("metadataSchemaVersion が不正です")
	}
	if err := validateSHA256("metadataCanonicalSha256", profile.MetadataCanonicalSHA256); err != nil {
		return err
	}
	if err := validateMachineString("cueSetVersion", profile.CueSetVersion, 128); err != nil {
		return err
	}
	return validateSHA256("cueArtifactSha256", profile.CueArtifactSHA256)
}

func validateLexiconArtifacts(artifacts []LexiconArtifact) error {
	//nolint:staticcheck // SOT-ENG-038: JSON null と空配列を別状態として閉じて検証する。
	if artifacts == nil || len(artifacts) < 1 || len(artifacts) > 16 {
		return fmt.Errorf("lexiconArtifacts の件数が不正です")
	}
	previous := ""
	for index, artifact := range artifacts {
		if index > 0 && previous >= artifact.LexiconID {
			return fmt.Errorf("lexiconArtifacts は lexiconId の byte 昇順でなければなりません")
		}
		if err := validateLexiconArtifact(artifact); err != nil {
			return err
		}
		previous = artifact.LexiconID
	}
	return nil
}

func validateLexiconArtifact(artifact LexiconArtifact) error {
	if err := validateMachineString("lexiconVersion", artifact.LexiconVersion, 128); err != nil {
		return err
	}
	expectedPaths, exists := allowedLexiconPaths()[artifact.LexiconID]
	if !exists || len(artifact.Files) != len(expectedPaths) {
		return fmt.Errorf("lexiconId または file 集合が閉じた対応にありません")
	}
	if err := validateFileDigests(artifact.Files, len(expectedPaths), len(expectedPaths)); err != nil {
		return err
	}
	for index, expected := range expectedPaths {
		if artifact.Files[index].Path != expected {
			return fmt.Errorf("lexicon file が閉じた対応と一致しません")
		}
	}
	if artifact.AggregateSHA256 != LexiconAggregateSHA256(artifact.Files) {
		return fmt.Errorf("aggregateSha256 が file tuple と一致しません")
	}
	return nil
}

func allowedLexiconPaths() map[string][]string {
	return map[string][]string{
		"lawNames": {
			"internal/lawnamelexicon/data/egov-current.json",
			"internal/lawnamelexicon/data/supplemental.json",
		},
		"legalConcepts": {"internal/legalconceptlexicon/data/current.json"},
	}
}

func validateComposition(
	descriptor CompositionDescriptor,
	profileSet ProfileSetIdentity,
	profiles []ProfileArtifact,
) error {
	if descriptor.DescriptorSchemaVersion < 1 ||
		descriptor.ProfileSetID != profileSet.ProfileSetID ||
		descriptor.ProfileSetVersion != profileSet.ProfileSetVersion ||
		descriptor.RankingVersion != profileSet.RankingVersion {
		return fmt.Errorf("composition と profileSet の identity が一致しません")
	}
	if err := validateMachineString("compositionVersion", descriptor.CompositionVersion, 128); err != nil {
		return err
	}
	if err := validateCompositionComponents(descriptor.Components, profiles); err != nil {
		return err
	}
	expected, err := CanonicalCompositionSHA256(descriptor)
	if err != nil || descriptor.DescriptorSHA256 != expected {
		return fmt.Errorf("descriptorSha256 が canonical tuple と一致しません")
	}
	return nil
}

func validateCompositionComponents(
	components []CompositionComponent,
	profiles []ProfileArtifact,
) error {
	if components == nil || len(components) != len(profiles)+3 {
		return fmt.Errorf("composition components の件数が不正です")
	}
	expected := []compositionIdentity{{"preprocessor", "query-preprocessor", "internal/querypreprocess"}}
	for _, profile := range profiles {
		expected = append(expected, compositionIdentity{
			role: "profile", componentID: profile.ProfileID, packageRoot: profilePackageRoot(profile.ProfileID),
		})
	}
	expected = append(expected,
		compositionIdentity{"composer", "candidate-composer", "internal/application/legalquery"},
		compositionIdentity{"selector", "legal-query-selector", "internal/application/legalquery"},
	)
	for index, component := range components {
		if component.Role != expected[index].role || component.ComponentID != expected[index].componentID ||
			component.PackageRoot != expected[index].packageRoot {
			return fmt.Errorf("composition component が閉じた対応または固定順と一致しません")
		}
		if err := validateMachineString("semanticVersion", component.SemanticVersion, 128); err != nil {
			return err
		}
	}
	return nil
}

type compositionIdentity struct {
	role        string
	componentID string
	packageRoot string
}

func profilePackageRoot(profileID string) string {
	if profileID == "core" {
		return "internal/queryprofile/core"
	}
	if profileID == "judicial-cases" {
		return "internal/queryprofile/judicialcases"
	}
	return ""
}

func validateSemanticSourceSet(sourceSet SemanticSourceSet) error {
	if err := validateSourceBuildContext(sourceSet); err != nil {
		return err
	}
	if err := validateGoDebugSettings(sourceSet.GoDebugSettings); err != nil {
		return err
	}
	if err := validateFileDigests(sourceSet.Files, 1, 8192); err != nil {
		return err
	}
	if err := validateModuleDependencies(sourceSet.ModuleDependencies); err != nil {
		return err
	}
	expected, err := CanonicalSourceSetSHA256(sourceSet)
	if err != nil || sourceSet.SourceSetSHA256 != expected {
		return fmt.Errorf("sourceSetSha256 が canonical tuple と一致しません")
	}
	return nil
}

func validateSourceBuildContext(sourceSet SemanticSourceSet) error {
	if err := validateMachineString("mainModulePath", sourceSet.MainModulePath, 256); err != nil {
		return err
	}
	if !goLanguageVersionPattern.MatchString(sourceSet.GoLanguageVersion) {
		return fmt.Errorf("goLanguageVersion が正規形ではありません")
	}
	if !goToolchainVersionPattern.MatchString(sourceSet.GoToolchainVersion) {
		return fmt.Errorf("goToolchainVersion が正規形ではありません")
	}
	if sourceSet.GOOS != "linux" || sourceSet.GOARCH != "amd64" ||
		sourceSet.GOAMD64 != "v1" || sourceSet.GOEXPERIMENT != "" || sourceSet.CGOEnabled != 0 ||
		sourceSet.BuildTags == nil || len(sourceSet.BuildTags) != 0 {
		return fmt.Errorf("semantic source の build context が固定値と一致しません")
	}
	return nil
}

func validateGoDebugSettings(settings []GoDebugSetting) error {
	if settings == nil || len(settings) > 128 {
		return fmt.Errorf("goDebugSettings は空配列または値を持つ配列でなければなりません")
	}
	previous := ""
	for index, setting := range settings {
		if index > 0 && previous >= setting.Name {
			return fmt.Errorf("goDebugSettings は name の byte 昇順でなければなりません")
		}
		if !goDebugNamePattern.MatchString(setting.Name) || len(setting.Name) > 64 {
			return fmt.Errorf("GODEBUG name が正規形ではありません")
		}
		if err := validateMachineString("GODEBUG value", setting.Value, 64); err != nil {
			return err
		}
		previous = setting.Name
	}
	return nil
}

func validateModuleDependencies(dependencies []ModuleDependency) error {
	if dependencies == nil || len(dependencies) > 1024 {
		return fmt.Errorf("moduleDependencies の件数が不正です")
	}
	previous := ""
	for _, dependency := range dependencies {
		key := dependency.ModulePath + "\x00" + dependency.Version
		if previous != "" && previous >= key {
			return fmt.Errorf("moduleDependencies は modulePath と version の byte 昇順でなければなりません")
		}
		if err := validateModuleDependency(dependency); err != nil {
			return err
		}
		previous = key
	}
	return nil
}

func validateModuleDependency(dependency ModuleDependency) error {
	if err := validateMachineString("modulePath", dependency.ModulePath, 512); err != nil {
		return err
	}
	if err := validateMachineString("module version", dependency.Version, 128); err != nil {
		return err
	}
	if !moduleSumPattern.MatchString(dependency.ModuleZipSum) {
		return fmt.Errorf("moduleZipSum が h1 checksum ではありません")
	}
	if err := validateSHA256("moduleZipRawSha256", dependency.ModuleZipRawSHA256); err != nil {
		return err
	}
	if dependency.ModuleZipByteLength < 1 || dependency.ModuleZipByteLength > 64<<20 ||
		dependency.ModuleZipEntryCount < 1 || dependency.ModuleZipEntryCount > 16384 ||
		dependency.ModuleExpandedByteLength < 1 || dependency.ModuleExpandedByteLength > 128<<20 {
		return fmt.Errorf("module archive の count または byte length が不正です")
	}
	if !moduleSumPattern.MatchString(dependency.ModuleGoModSum) {
		return fmt.Errorf("moduleGoModSum が h1 checksum ではありません")
	}
	return validateSHA256("moduleGoModRawSha256", dependency.ModuleGoModRawSHA256)
}

func stringPosition(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

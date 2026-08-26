package lawv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var errLawVersionCompareBudget = errors.New("law version compare budget exceeded")

type parsedLawVersionArticle struct {
	article              model.LawVersionArticle
	structureFingerprint string
}

type parsedLawVersionDocument struct {
	snapshot   model.LawVersionSnapshot
	ref        model.SourceResourceRef
	provenance []model.Provenance
	articles   map[string]parsedLawVersionArticle
	order      []string
	textBytes  int
}

type compareElementFrame struct {
	name   xml.Name
	number string
}

type compareArticleBuilder struct {
	provision        model.LawArticleProvision
	articleNumber    string
	partNumber       string
	chapterNumber    string
	sectionNumber    string
	subsectionNumber string
	divisionNumber   string
	documentOrder    int
	titleBuilder     strings.Builder
	captionBuilder   strings.Builder
	textBuilder      strings.Builder
	structure        hash.Hash
	titleSeen        bool
	captionSeen      bool
}

type compareDocumentState struct {
	limits         lawVersionCompareLimits
	stack          []compareElementFrame
	elementCount   int
	rootCount      int
	provision      model.LawArticleProvision
	provisionDepth int
	skipProvision  bool
	current        *compareArticleBuilder
	currentDepth   int
	titleDepth     int
	captionDepth   int
	articles       map[string]parsedLawVersionArticle
	order          []string
	baseCitation   model.Citation
	totalTextBytes int
}

func parseLawVersionDocument(
	ctx context.Context,
	resource model.SourcedResource[model.LawDocumentRepresentation],
	limits lawVersionCompareLimits,
) (parsedLawVersionDocument, error) {
	if ctx == nil {
		return parsedLawVersionDocument{}, fmt.Errorf("context は必須です")
	}
	if err := limits.validate(); err != nil {
		return parsedLawVersionDocument{}, err
	}
	if err := ctx.Err(); err != nil {
		return parsedLawVersionDocument{}, err
	}
	if err := resource.Validate(); err != nil {
		return parsedLawVersionDocument{}, invalidLawVersionCompareResponse()
	}
	document := resource.Data()
	if err := document.Validate(); err != nil {
		return parsedLawVersionDocument{}, invalidLawVersionCompareResponse()
	}
	if document.Format() != model.LawDocumentFormatXML {
		return parsedLawVersionDocument{}, invalidLawVersionCompareResponse()
	}
	content := document.Content()
	if len(content) > lawDocumentParserInputBytes {
		return parsedLawVersionDocument{}, newLawVersionCompareSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if !utf8.ValidString(content) {
		return parsedLawVersionDocument{}, newLawVersionCompareSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}

	snapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      document.Law(),
		AsOf:     cloneOptionalDateFromRepresentation(document),
		Citation: document.Citation(),
	})
	if err != nil {
		return parsedLawVersionDocument{}, invalidLawVersionCompareResponse()
	}

	state := compareDocumentState{
		limits:       limits,
		articles:     make(map[string]parsedLawVersionArticle),
		baseCitation: document.Citation(),
	}
	decoder := xml.NewDecoder(&contextReader{
		ctx:    ctx,
		reader: bytes.NewReader([]byte(content)),
	})
	decoder.Strict = true
	for {
		token, tokenErr := decoder.RawToken()
		if errors.Is(tokenErr, io.EOF) {
			break
		}
		if tokenErr != nil {
			return parsedLawVersionDocument{}, classifyLawVersionCompareXMLError(ctx, tokenErr)
		}
		if consumeErr := state.consume(token); consumeErr != nil {
			return parsedLawVersionDocument{}, classifyLawVersionCompareXMLError(ctx, consumeErr)
		}
	}
	if err := ctx.Err(); err != nil {
		return parsedLawVersionDocument{}, err
	}
	if len(state.stack) != 0 || state.rootCount != 1 || state.current != nil {
		return parsedLawVersionDocument{}, invalidLawVersionCompareResponse()
	}

	return parsedLawVersionDocument{
		snapshot:   snapshot,
		ref:        resource.Ref(),
		provenance: resource.Provenance(),
		articles:   state.articles,
		order:      slices.Clone(state.order),
		textBytes:  state.totalTextBytes,
	}, nil
}

func (s *compareDocumentState) consume(token xml.Token) error {
	switch value := token.(type) {
	case xml.StartElement:
		return s.consumeStart(value)
	case xml.EndElement:
		return s.consumeEnd(value)
	case xml.CharData:
		if s.current == nil {
			if len(s.stack) == 0 && strings.TrimSpace(string(value)) != "" {
				return lawDocumentXMLErrorStructure
			}
			return nil
		}
		text := string(value)
		if s.titleDepth != 0 {
			_, _ = s.current.titleBuilder.WriteString(text)
		}
		if s.captionDepth != 0 {
			_, _ = s.current.captionBuilder.WriteString(text)
		}
		_, _ = s.current.textBuilder.WriteString(text)
		return nil
	case xml.Directive, xml.ProcInst:
		return lawDocumentXMLErrorUnsafe
	default:
		return nil
	}
}

func (s *compareDocumentState) consumeStart(element xml.StartElement) error {
	if err := validateCompareElementSafety(element); err != nil {
		return err
	}
	depth := len(s.stack) + 1
	s.elementCount++
	if s.elementCount > lawDocumentXMLElements {
		return lawDocumentXMLErrorElements
	}
	if depth > lawDocumentXMLDepth {
		return lawDocumentXMLErrorDepth
	}
	if depth == 1 {
		s.rootCount++
		if s.rootCount != 1 || !isXMLName(element.Name, "Law") {
			return lawDocumentXMLErrorStructure
		}
	}

	number, err := compareHierarchyNumber(element)
	if err != nil {
		return err
	}
	if isDirectCompareProvision(s.stack, element.Name) {
		_, hasAmendLawNum, attributeErr := unqualifiedAttribute(element, "AmendLawNum")
		if attributeErr != nil {
			return attributeErr
		}
		s.provisionDepth = depth
		s.skipProvision = false
		switch element.Name.Local {
		case "MainProvision":
			s.provision = model.LawArticleProvisionMain
		case "SupplProvision":
			s.provision = model.LawArticleProvisionSupplementary
			s.skipProvision = hasAmendLawNum
		}
	}

	if s.current == nil && s.provisionDepth != 0 && !s.skipProvision &&
		isXMLName(element.Name, "Article") {
		path := s.stack[s.provisionDepth:]
		locationValues, pathErr := compareArticleLocationValues(s.provision, element, path)
		if pathErr != nil {
			return pathErr
		}
		if locationValues != nil {
			if len(s.order) >= s.limits.articlesPerVersion {
				return errLawVersionCompareBudget
			}
			s.current = &compareArticleBuilder{
				provision:        locationValues.Provision,
				articleNumber:    locationValues.ArticleNumber,
				partNumber:       locationValues.PartNumber,
				chapterNumber:    locationValues.ChapterNumber,
				sectionNumber:    locationValues.SectionNumber,
				subsectionNumber: locationValues.SubsectionNumber,
				divisionNumber:   locationValues.DivisionNumber,
				documentOrder:    len(s.order) + 1,
				structure:        sha256.New(),
			}
			s.currentDepth = depth
		}
	}

	s.stack = append(s.stack, compareElementFrame{name: element.Name, number: number})
	if s.current != nil {
		writeCompareStructureStart(s.current.structure, element)
		if depth == s.currentDepth+1 && isXMLName(element.Name, "ArticleTitle") {
			if s.current.titleSeen {
				return lawDocumentXMLErrorStructure
			}
			s.current.titleSeen = true
			s.titleDepth = depth
		}
		if depth == s.currentDepth+1 && isXMLName(element.Name, "ArticleCaption") {
			if s.current.captionSeen {
				return lawDocumentXMLErrorStructure
			}
			s.current.captionSeen = true
			s.captionDepth = depth
		}
	}
	return nil
}

func (s *compareDocumentState) consumeEnd(element xml.EndElement) error {
	depth := len(s.stack)
	if element.Name.Space != "" || depth == 0 || s.stack[depth-1].name != element.Name {
		return lawDocumentXMLErrorStructure
	}
	if s.current != nil {
		writeCompareStructureEnd(s.current.structure, element)
		if s.titleDepth == depth && isXMLName(element.Name, "ArticleTitle") {
			s.titleDepth = 0
		}
		if s.captionDepth == depth && isXMLName(element.Name, "ArticleCaption") {
			s.captionDepth = 0
		}
		if s.currentDepth == depth && isXMLName(element.Name, "Article") {
			article, fingerprint, err := s.current.build(s.baseCitation)
			if err != nil {
				return err
			}
			identity := lawVersionArticleIdentityKey(article.Location())
			if _, duplicate := s.articles[identity]; duplicate {
				return lawDocumentXMLErrorStructure
			}
			s.articles[identity] = parsedLawVersionArticle{
				article:              article,
				structureFingerprint: fingerprint,
			}
			s.order = append(s.order, identity)
			s.totalTextBytes += len(article.Text())
			if s.totalTextBytes > s.limits.combinedTextBytes {
				return errLawVersionCompareBudget
			}
			s.current = nil
			s.currentDepth = 0
		}
	}
	if s.provisionDepth == depth {
		s.provision = ""
		s.provisionDepth = 0
		s.skipProvision = false
	}
	s.stack = s.stack[:depth-1]
	return nil
}

func (b compareArticleBuilder) build(
	base model.Citation,
) (model.LawVersionArticle, string, error) {
	location, err := model.NewLawVersionArticleLocation(model.LawVersionArticleLocationValues{
		Provision:        b.provision,
		ArticleNumber:    b.articleNumber,
		PartNumber:       b.partNumber,
		ChapterNumber:    b.chapterNumber,
		SectionNumber:    b.sectionNumber,
		SubsectionNumber: b.subsectionNumber,
		DivisionNumber:   b.divisionNumber,
	})
	if err != nil {
		return model.LawVersionArticle{}, "", err
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     base.Source(),
		LawID:      base.LawID(),
		RevisionID: base.RevisionID(),
		Location:   string(location.Provision()) + ":article=" + location.ArticleNumber(),
		URL:        base.URL(),
	})
	if err != nil {
		return model.LawVersionArticle{}, "", err
	}
	fingerprint := hex.EncodeToString(b.structure.Sum(nil))
	article, err := model.NewLawVersionArticle(model.LawVersionArticleValues{
		Location:             location,
		ArticleTitle:         normalizeCompareText(b.titleBuilder.String()),
		ArticleCaption:       normalizeCompareText(b.captionBuilder.String()),
		Text:                 normalizeCompareText(b.textBuilder.String()),
		Citation:             citation,
		DocumentOrder:        b.documentOrder,
		StructureFingerprint: fingerprint,
	})
	if err != nil {
		return model.LawVersionArticle{}, "", err
	}
	return article, fingerprint, nil
}

func compareLawVersionDocuments(
	ctx context.Context,
	before parsedLawVersionDocument,
	after parsedLawVersionDocument,
	limits lawVersionCompareLimits,
) (model.LawVersionComparison, error) {
	if ctx == nil {
		return model.LawVersionComparison{}, fmt.Errorf("context は必須です")
	}
	if err := limits.validate(); err != nil {
		return model.LawVersionComparison{}, err
	}
	if err := ctx.Err(); err != nil {
		return model.LawVersionComparison{}, err
	}
	if before.snapshot.Law().LawID() != after.snapshot.Law().LawID() {
		return model.LawVersionComparison{}, invalidLawVersionCompareResponse()
	}
	if err := validateLawVersionCompareTextBudget(before, after, limits); err != nil {
		return model.LawVersionComparison{}, err
	}

	capacity := len(before.articles) + len(after.articles)
	if capacity > limits.changes {
		capacity = limits.changes
	}
	changes := make([]model.LawVersionChange, 0, capacity)
	addedCount, removedCount, modifiedCount, unchangedCount := 0, 0, 0, 0
	appendChange := func(change model.LawVersionChange) error {
		if len(changes) >= limits.changes {
			return errLawVersionCompareBudget
		}
		changes = append(changes, change)
		return nil
	}

	for _, key := range after.order {
		if err := ctx.Err(); err != nil {
			return model.LawVersionComparison{}, err
		}
		afterArticle := after.articles[key]
		beforeArticle, exists := before.articles[key]
		if !exists {
			change, err := model.NewLawVersionChange(model.LawVersionChangeValues{
				ChangeKind: model.LawVersionChangeKindAdded,
				After:      &afterArticle.article,
			})
			if err != nil {
				return model.LawVersionComparison{}, invalidLawVersionCompareResponse()
			}
			if err := appendChange(change); err != nil {
				return model.LawVersionComparison{}, lawVersionCompareBudgetError(err)
			}
			addedCount++
			continue
		}
		if lawVersionArticlesEqual(beforeArticle, afterArticle) {
			unchangedCount++
			continue
		}
		reasons := make([]model.LawVersionChangeReason, 0, 3)
		if !beforeArticle.article.Location().HasSameAuxiliaryLocation(afterArticle.article.Location()) {
			reasons = append(reasons, model.LawVersionChangeReasonLocation)
		}
		if beforeArticle.article.Text() != afterArticle.article.Text() {
			reasons = append(reasons, model.LawVersionChangeReasonText)
		}
		if beforeArticle.structureFingerprint != afterArticle.structureFingerprint {
			reasons = append(reasons, model.LawVersionChangeReasonStructure)
		}
		change, err := model.NewLawVersionChange(model.LawVersionChangeValues{
			ChangeKind:    model.LawVersionChangeKindModified,
			ChangeReasons: reasons,
			Before:        &beforeArticle.article,
			After:         &afterArticle.article,
		})
		if err != nil {
			return model.LawVersionComparison{}, invalidLawVersionCompareResponse()
		}
		if err := appendChange(change); err != nil {
			return model.LawVersionComparison{}, lawVersionCompareBudgetError(err)
		}
		modifiedCount++
	}

	for _, key := range before.order {
		if err := ctx.Err(); err != nil {
			return model.LawVersionComparison{}, err
		}
		if _, exists := after.articles[key]; exists {
			continue
		}
		beforeArticle := before.articles[key]
		change, err := model.NewLawVersionChange(model.LawVersionChangeValues{
			ChangeKind: model.LawVersionChangeKindRemoved,
			Before:     &beforeArticle.article,
		})
		if err != nil {
			return model.LawVersionComparison{}, invalidLawVersionCompareResponse()
		}
		if err := appendChange(change); err != nil {
			return model.LawVersionComparison{}, lawVersionCompareBudgetError(err)
		}
		removedCount++
	}

	comparison, err := model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              before.snapshot.Law().LawID(),
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             before.snapshot,
		After:              after.snapshot,
		BeforeArticleCount: len(before.articles),
		AfterArticleCount:  len(after.articles),
		AddedCount:         addedCount,
		RemovedCount:       removedCount,
		ModifiedCount:      modifiedCount,
		UnchangedCount:     unchangedCount,
		TotalCount:         len(changes),
		Items:              changes,
	})
	if err != nil {
		return model.LawVersionComparison{}, invalidLawVersionCompareResponse()
	}
	return comparison, nil
}

func lawVersionArticlesEqual(before, after parsedLawVersionArticle) bool {
	return before.article.Location().HasSameAuxiliaryLocation(after.article.Location()) &&
		before.article.Text() == after.article.Text() &&
		before.structureFingerprint == after.structureFingerprint
}

func normalizeCompareText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cloneOptionalDateFromRepresentation(document model.LawDocumentRepresentation) *model.Date {
	asOf, exists := document.AsOf()
	if !exists {
		return nil
	}
	return &asOf
}

func isDirectCompareProvision(stack []compareElementFrame, name xml.Name) bool {
	if len(stack) != 2 || !isXMLName(stack[0].name, "Law") ||
		!isXMLName(stack[1].name, "LawBody") {
		return false
	}
	return isXMLName(name, "MainProvision") || isXMLName(name, "SupplProvision")
}

func compareHierarchyNumber(element xml.StartElement) (string, error) {
	switch element.Name.Local {
	case "Part", "Chapter", "Section", "Subsection", "Division":
		value, _, err := unqualifiedAttribute(element, "Num")
		return value, err
	default:
		return "", nil
	}
}

func compareArticleLocationValues(
	provision model.LawArticleProvision,
	article xml.StartElement,
	path []compareElementFrame,
) (*model.LawVersionArticleLocationValues, error) {
	if !allowedCompareArticlePath(path) {
		return nil, nil
	}
	articleNumber, exists, err := unqualifiedAttribute(article, "Num")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, lawDocumentXMLErrorStructure
	}
	values := &model.LawVersionArticleLocationValues{
		Provision:     provision,
		ArticleNumber: articleNumber,
	}
	for _, frame := range path {
		switch frame.name.Local {
		case "Part":
			values.PartNumber = frame.number
		case "Chapter":
			values.ChapterNumber = frame.number
		case "Section":
			values.SectionNumber = frame.number
		case "Subsection":
			values.SubsectionNumber = frame.number
		case "Division":
			values.DivisionNumber = frame.number
		}
	}
	return values, nil
}

func allowedCompareArticlePath(path []compareElementFrame) bool {
	ranks := map[string]int{
		"Part": 1, "Chapter": 2, "Section": 3, "Subsection": 4, "Division": 5,
	}
	previous := 0
	for _, frame := range path {
		if frame.name.Space != "" {
			return false
		}
		rank, exists := ranks[frame.name.Local]
		if !exists || rank <= previous {
			return false
		}
		previous = rank
	}
	return true
}

func validateCompareElementSafety(element xml.StartElement) error {
	if element.Name.Space != "" {
		return lawDocumentXMLErrorUnsafe
	}
	seen := make(map[string]struct{}, len(element.Attr))
	for _, attribute := range element.Attr {
		if attribute.Name.Local == "xmlns" || attribute.Name.Space == "xmlns" {
			if attribute.Value != "" {
				return lawDocumentXMLErrorUnsafe
			}
			continue
		}
		if attribute.Name.Space != "" {
			return lawDocumentXMLErrorUnsafe
		}
		if _, duplicate := seen[attribute.Name.Local]; duplicate {
			return lawDocumentXMLErrorStructure
		}
		seen[attribute.Name.Local] = struct{}{}
	}
	return nil
}

type compareCanonicalAttribute struct {
	name  string
	value string
}

func writeCompareStructureStart(destination hash.Hash, element xml.StartElement) {
	attributes := make([]compareCanonicalAttribute, 0, len(element.Attr))
	for _, attribute := range element.Attr {
		if attribute.Name.Local == "xmlns" || attribute.Name.Space == "xmlns" {
			continue
		}
		attributes = append(attributes, compareCanonicalAttribute{
			name: attribute.Name.Local, value: attribute.Value,
		})
	}
	sort.Slice(attributes, func(left, right int) bool {
		if attributes[left].name != attributes[right].name {
			return attributes[left].name < attributes[right].name
		}
		return attributes[left].value < attributes[right].value
	})
	writeCompareFingerprintField(destination, 'S', element.Name.Local)
	var count [4]byte
	binary.BigEndian.PutUint32(
		count[:],
		uint32(len(attributes)), //nolint:gosec // SOT-ENG-019: 比較 fingerprint の属性数は公式 XML の現実的上限内であり、仕様上 32-bit field へ固定しても安全である。
	)
	_, _ = destination.Write(count[:])
	for _, attribute := range attributes {
		writeCompareFingerprintField(destination, 'A', attribute.name, attribute.value)
	}
}

func writeCompareStructureEnd(destination hash.Hash, element xml.EndElement) {
	writeCompareFingerprintField(destination, 'E', element.Name.Local)
}

func writeCompareFingerprintField(destination hash.Hash, kind byte, values ...string) {
	_, _ = destination.Write([]byte{kind})
	var length [4]byte
	for _, value := range values {
		binary.BigEndian.PutUint32(
			length[:],
			uint32(len(value)), //nolint:gosec // SOT-ENG-019: 比較 fingerprint の文字列長は仕様上 32-bit field へ固定し、実入力もその上限内に収まる。
		)
		_, _ = destination.Write(length[:])
		_, _ = destination.Write([]byte(value))
	}
}

func lawVersionArticleIdentityKey(location model.LawVersionArticleLocation) string {
	return string(location.Provision()) + "\x00" + location.ArticleNumber()
}

func classifyLawVersionCompareXMLError(ctx context.Context, err error) error {
	if contextErr := ctx.Err(); contextErr != nil {
		return contextErr
	}
	switch {
	case errors.Is(err, errLawVersionCompareBudget):
		return lawVersionCompareBudgetError(err)
	case errors.Is(err, lawDocumentXMLErrorElements):
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	case errors.Is(err, lawDocumentXMLErrorDepth),
		errors.Is(err, lawDocumentXMLErrorUnsafe),
		strings.Contains(err.Error(), "invalid character entity"):
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	default:
		return invalidLawVersionCompareResponse()
	}
}

func lawVersionCompareBudgetError(_ error) error {
	return newLawVersionCompareSourceError(
		model.SourceErrorCodeSourceProcessingLimit,
		"",
	)
}

func invalidLawVersionCompareResponse() error {
	return newLawVersionCompareSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}

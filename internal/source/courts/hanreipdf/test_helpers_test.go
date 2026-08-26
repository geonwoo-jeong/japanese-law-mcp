package hanreipdf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/tsawler/tabula/reader"
)

func judicialcasecitationextracttestRequest(
	t *testing.T,
) (model.SourcedResource[model.JudicialDecisionDetails], model.JudicialDocumentLink) {
	return judicialcasecitationextracttestRequestWithSourceID(
		t,
		sourceID,
		"https://www.courts.go.jp/assets/hanrei/00001.pdf",
	)
}

func judicialcasecitationextracttestRequestWithURL(
	t *testing.T,
	documentURL string,
) (model.SourcedResource[model.JudicialDecisionDetails], model.JudicialDocumentLink) {
	return judicialcasecitationextracttestRequestWithSourceID(t, sourceID, documentURL)
}

func judicialcasecitationextracttestRequestWithSourceID(
	t *testing.T,
	currentSourceID string,
	documentURL string,
) (model.SourcedResource[model.JudicialDecisionDetails], model.JudicialDocumentLink) {
	return judicialcasecitationextracttestRequestWithOptions(t, extractRequestFixtureOptions{
		sourceID:    currentSourceID,
		documentURL: documentURL,
	})
}

type extractRequestFixtureOptions struct {
	sourceID         string
	documentURL      string
	reporterCitation *string
	divisionName     *string
	decisionType     *string
}

func judicialcasecitationextracttestRequestWithOptions(
	t *testing.T,
	options extractRequestFixtureOptions,
) (model.SourcedResource[model.JudicialDecisionDetails], model.JudicialDocumentLink) {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         options.sourceID,
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: serviceURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	document, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "判決文",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       options.documentURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "00001",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6(受)123",
		DecisionDate:        mustDate(t, "2025-01-02"),
		CourtName:           "最高裁判所",
		DivisionName:        options.divisionName,
		DecisionType:        options.decisionType,
		DetailURL:           "https://www.courts.go.jp/app/hanrei_jp/detail2?id=00001",
		Documents:           []model.JudicialDocumentLink{document},
		Source:              source,
	})
	if err != nil {
		t.Fatal(err)
	}
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:          summary,
		ReporterCitation: options.reporterCitation,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     options.sourceID,
		ResourceType: "judicial-decision",
		ResourceID:   "00001",
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            summary.DetailURL(),
		RetrievedAt:    mustRetrievedAt(t),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		t.Fatal(err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       details,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resource, document
}

func mustDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func mustRetrievedAt(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2026, 8, 26, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
}

func mustExtractRequest(t *testing.T) judicialcasecitationextract.Request {
	t.Helper()
	decision, document := judicialcasecitationextracttestRequest(t)
	request, err := judicialcasecitationextract.NewRequest(
		judicialcasecitationextract.RequestValues{Decision: decision, Document: document},
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func syntheticJapanesePDF(text string) []byte {
	runes := []rune(text)
	var encoded strings.Builder
	var cmap strings.Builder
	for index, current := range runes {
		code := index + 1
		fmt.Fprintf(&encoded, "%04X", code)
		units := utf16.Encode([]rune{current})
		fmt.Fprintf(&cmap, "<%04X> <", code)
		for _, unit := range units {
			fmt.Fprintf(&cmap, "%04X", unit)
		}
		cmap.WriteString(">\n")
	}
	cmapBody := fmt.Sprintf(
		"/CIDInit /ProcSet findresource begin\n"+
			"12 dict begin\nbegincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n"+
			"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n"+
			"1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n"+
			"%d beginbfchar\n%sendbfchar\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend",
		len(runes),
		cmap.String(),
	)
	content := "BT /F1 12 Tf 20 100 Td <" + encoded.String() + "> Tj ET"
	return buildSyntheticPDF([][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 500 200] /Resources << /Font << /F1 4 0 R >> >> /Contents 6 0 R >>"),
		[]byte("<< /Type /Font /Subtype /Type0 /BaseFont /HeiseiMin-W3 /Encoding /Identity-H /DescendantFonts [5 0 R] /ToUnicode 7 0 R >>"),
		[]byte("<< /Type /Font /Subtype /CIDFontType0 /BaseFont /HeiseiMin-W3 /CIDSystemInfo << /Registry (Adobe) /Ordering (Japan1) /Supplement 2 >> /DW 1000 >>"),
		streamObject([]byte(content)),
		streamObject([]byte(cmapBody)),
	})
}

func syntheticImageOnlyPDF() []byte {
	return buildSyntheticPDF([][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"),
		[]byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << /XObject << /Im1 4 0 R >> >> /Contents 5 0 R >>"),
		[]byte("<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceGray /BitsPerComponent 8 /Length 1 >>\nstream\n\x00\nendstream"),
		streamObject([]byte("q 1 0 0 1 0 0 cm /Im1 Do Q")),
	})
}

func syntheticOversizedPageTreePDF() []byte {
	return buildSyntheticPDF([][]byte{
		[]byte("<< /Type /Catalog /Pages 2 0 R >>"),
		[]byte("<< /Type /Pages /Kids [] /Count 301 >>"),
	})
}

func mustSyntheticPDFReader(t *testing.T, pdf []byte) *reader.Reader {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "fixture.pdf")
	if err := os.WriteFile(filePath, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filePath) //nolint:gosec // SOT-IF-071: t.TempDir 配下へ test が作成した固定 PDF fixture だけを読む。
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	pdfReader, err := reader.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	return pdfReader
}

func streamObject(data []byte) []byte {
	return []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(data), data))
}

func buildSyntheticPDF(objects [][]byte) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for index, object := range objects {
		offsets[index] = buffer.Len()
		fmt.Fprintf(&buffer, "%d 0 obj\n", index+1)
		buffer.Write(object)
		buffer.WriteString("\nendobj\n")
	}
	xref := buffer.Len()
	fmt.Fprintf(&buffer, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&buffer, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(
		&buffer,
		"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF",
		len(objects)+1,
		xref,
	)
	return buffer.Bytes()
}

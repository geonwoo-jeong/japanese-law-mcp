package lawv2

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestEGovLawDataRuntimeResponseClassification(t *testing.T) {
	t.Parallel()
	t.Run("egov-law-data-runtime-response-classification", func(t *testing.T) {
		t.Parallel()

		location := mustLawArticleLocation(
			t,
			model.LawArticleProvisionMain,
			"1",
			0,
		)
		valid := minimumLawDataXML()
		bodies := []string{
			strings.Replace(valid, `<law_info><law_id>322CO0000000016</law_id></law_info>`, "", 1),
			strings.Replace(valid, `<revision_info>`, `<revision_info><law_title>重複</law_title>`, 1),
			strings.TrimSuffix(valid, `</law_data_response>`),
		}
		for _, body := range bodies {
			_, documentErr := parseLawDocumentResponse(
				context.Background(),
				[]byte(body),
			)
			assertLawDocumentSourceError(
				t,
				documentErr,
				model.SourceErrorCodeInvalidSourceResponse,
			)
			_, articleErr := parseLawArticleResponse(
				context.Background(),
				[]byte(body),
				location,
			)
			assertLawArticleSourceError(
				t,
				articleErr,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		}

		client := mustTestClient(t, clientDependencies{
			doer: doerFunc(func(*http.Request) (*http.Response, error) {
				return response(
					http.StatusOK,
					`{"unexpected":true}`,
					map[string]string{"Content-Type": "application/json"},
				), nil
			}),
			now:   time.Now,
			sleep: sleepWithContext,
		})
		request := lawDocumentRequest{identifier: "322CO0000000016"}
		_, documentErr := client.fetchLawDocument(context.Background(), request)
		assertLawDocumentSourceError(
			t,
			documentErr,
			model.SourceErrorCodeInvalidSourceResponse,
		)
		_, articleErr := client.fetchLawArticle(context.Background(), request)
		assertLawArticleSourceError(
			t,
			articleErr,
			model.SourceErrorCodeInvalidSourceResponse,
		)
	})
}

func TestEGovLawDataContractChangeSeparation(t *testing.T) {
	t.Parallel()
	t.Run("egov-law-data-contract-change-separation", func(t *testing.T) {
		t.Parallel()

		recorded := fixture(t, "fixtures/law-data-contract.json")
		for _, sourceError := range []sourceErrorFactory{
			newLawDocumentSourceError,
			newLawArticleSourceError,
		} {
			if err := verifyRecordedLawDataContract(recorded, sourceError); err != nil {
				t.Fatalf("保存済み公式契約を拒否しました: %v", err)
			}
			changed := bytes.Replace(
				recorded,
				[]byte(`"successMediaType": "application/xml"`),
				[]byte(`"successMediaType": "text/html"`),
				1,
			)
			if bytes.Equal(changed, recorded) {
				t.Fatal("公式契約 fixture を変更できませんでした")
			}
			assertSourceErrorCode(
				t,
				verifyRecordedLawDataContract(changed, sourceError),
				model.SourceErrorCodeSourceContractChanged,
			)
		}

		missingLawInfo := strings.Replace(
			minimumLawDataXML(),
			`<law_info><law_id>322CO0000000016</law_id></law_info>`,
			"",
			1,
		)
		_, runtimeErr := parseLawDocumentResponse(
			context.Background(),
			[]byte(missingLawInfo),
		)
		assertLawDocumentSourceError(
			t,
			runtimeErr,
			model.SourceErrorCodeInvalidSourceResponse,
		)
	})
}

func TestEGovLawDataInputResponseIdentity(t *testing.T) {
	t.Parallel()
	t.Run("egov-law-data-input-response-identity", func(t *testing.T) {
		t.Parallel()

		request := newLawDocumentReadRequest(
			t,
			"322CO0000000016",
			"322CO0000000016_20240401_506CO0000000161",
			nil,
		)
		response, err := parseLawDocumentResponse(
			context.Background(),
			[]byte(minimumLawDataXML()),
		)
		if err != nil {
			t.Fatalf("正常な応答を解析できません: %v", err)
		}
		key := request.Resource().Key()
		if err := validateLawDocumentResponse(key, response); err != nil {
			t.Fatalf("一致する identity を拒否しました: %v", err)
		}

		mismatchedLaw := response
		mismatchedLaw.law.lawID = "別の法令"
		assertLawDocumentSourceError(
			t,
			validateLawDocumentResponse(key, mismatchedLaw),
			model.SourceErrorCodeInvalidSourceResponse,
		)
		mismatchedRevision := response
		mismatchedRevision.law.revisionID = "別のリビジョン"
		assertLawDocumentSourceError(
			t,
			validateLawDocumentResponse(key, mismatchedRevision),
			model.SourceErrorCodeInvalidSourceResponse,
		)
	})
}

func TestEGovLawDataXMLSafetyBoundary(t *testing.T) {
	t.Parallel()
	t.Run("egov-law-data-xml-safety-boundary", func(t *testing.T) {
		t.Parallel()

		body := []byte(`<!DOCTYPE law_data_response [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
			<law_data_response><law_info/><revision_info/><law_full_text>
			<Law>&xxe;</Law></law_full_text></law_data_response>`)
		_, documentErr := parseLawDocumentResponse(context.Background(), body)
		assertLawDocumentSourceError(
			t,
			documentErr,
			model.SourceErrorCodeUnsafeSourceContent,
		)
		_, articleErr := parseLawArticleResponse(
			context.Background(),
			body,
			mustLawArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
		)
		assertLawArticleSourceError(
			t,
			articleErr,
			model.SourceErrorCodeUnsafeSourceContent,
		)
	})
}

func minimumLawDataXML() string {
	return `<law_data_response>` +
		`<law_info><law_id>322CO0000000016</law_id></law_info>` +
		`<revision_info>` +
		`<law_revision_id>322CO0000000016_20240401_506CO0000000161</law_revision_id>` +
		`<law_title>地方自治法施行令</law_title>` +
		`</revision_info>` +
		`<law_full_text><Law><LawBody><MainProvision>` +
		`<Article Num="1"><Paragraph Num="1"/></Article>` +
		`</MainProvision></LawBody></Law></law_full_text>` +
		`</law_data_response>`
}

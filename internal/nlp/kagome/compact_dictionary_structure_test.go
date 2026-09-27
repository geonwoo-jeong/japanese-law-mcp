package kagome

import (
	"archive/zip"
	"testing"

	"github.com/ikawaha/kagome-dict/dict"
)

func TestCompactDictionaryRejectsMissingDuplicateAndUnknownParts(t *testing.T) {
	names := []string{dict.MorphDictFileName, dict.ContentMetaFileName, dict.IndexDictFileName, dict.ConnectionDictFileName, dict.CharDefDictFileName, dict.UnkDictFileName, dict.DictInfoFileName}
	parts := make([]*zip.File, 0, len(names))
	for _, name := range names {
		parts = append(parts, &zip.File{FileHeader: zip.FileHeader{Name: name}})
	}
	if err := validateCompactDictionaryParts(&zip.Reader{File: parts}); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]*zip.File{
		"不足":   parts[:len(parts)-1],
		"重複":   append(append([]*zip.File{}, parts...), parts[0]),
		"未定義":  append(append([]*zip.File{}, parts...), &zip.File{FileHeader: zip.FileHeader{Name: "unknown"}}),
		"従来品詞": append(append([]*zip.File{}, parts...), &zip.File{FileHeader: zip.FileHeader{Name: dict.POSDictFileName}}),
		"付加情報": append(append([]*zip.File{}, parts...), &zip.File{FileHeader: zip.FileHeader{Name: dict.ContentDictFileName}}),
	}
	for name, files := range cases {
		if err := validateCompactDictionaryParts(&zip.Reader{File: files}); err == nil {
			t.Fatalf("SOT-ENG-051: %s を含む辞書 archive を受理しました", name)
		}
	}
}

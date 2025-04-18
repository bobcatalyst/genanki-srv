package genanki_srv

import (
    "io/fs"
    "math/rand/v2"
    "path/filepath"
)

const BaseAnswerTemplate = "{{FrontSide}}\n<hr id=\"answer\">\n"

type AnkiModelField struct {
    Name   string  `json:"name"`
    Font   *string `json:"font"`
    RTL    bool    `json:"rtl"`
    Size   *int    `json:"size"`
    Sticky bool    `json:"sticky"`
}

const (
    DefaultModelFont = "Liberation Sans"
    DefaultModelSize = 20
)

func NewAnkiModelField(name string) *AnkiModelField {
    font := DefaultModelFont
    size := DefaultModelSize
    return &AnkiModelField{
        Name: name,
        Font: &font,
        Size: &size,
    }
}

func (f *AnkiModelField) SetFont(font string) {
    f.Font = &font
}

func (f *AnkiModelField) SetSize(size int) {
    f.Size = &size
}

type AnkiModelTemplate struct {
    Name  string `json:"name"`
    Qfmt  string `json:"qfmt"`
    Afmt  string `json:"afmt"`
    Bqfmt string `json:"bqfmt"`
    Bafmt string `json:"bafmt"`
    Bfont string `json:"bfont"`
    Bsize int    `json:"bsize"`
}

func NewAnkiModelTemplate(name, qfmt, afmt string) *AnkiModelTemplate {
    return &AnkiModelTemplate{
        Name: name,
        Qfmt: qfmt,
        Afmt: BaseAnswerTemplate + afmt,
    }
}

type ModelType int

const (
    ModelTypeBasic ModelType = 0
    ModelTypeCloze ModelType = 1
)

type AnkiModel struct {
    Id             int                  `json:"id"`
    Name           string               `json:"name"`
    Templates      []*AnkiModelTemplate `json:"templates"`
    Fields         []*AnkiModelField    `json:"fields"`
    Css            *string              `json:"css"`
    ModelType      int                  `json:"model_type"`
    LatexPre       *string              `json:"latex_pre"`
    LatexPost      *string              `json:"latex_post"`
    SortFieldIndex int                  `json:"sort_field_index"`
}

const (
    DefaultModelCSS = `.card {
    font-family: arial;
    font-size: 20px;
    text-align: center;
    color: black;
    background-color: white;
}`
    DefaultModelLatexPost = "\\end{document}"
    DefaultModelLatexPre  = "\\documentclass[12pt]{article}\n\\special{papersize=3in,5in}\n\\usepackage[utf8]{inputenc}\n" +
        "\\usepackage{amssymb,amsmath}\n\\pagestyle{empty}\n\\setlength{\\parindent}{0in}\n" +
        "\\begin{document}\n"
)

func NewAnkiModel(name string, templates []*AnkiModelTemplate, fields []*AnkiModelField, id ...int) *AnkiModel {
    am := &AnkiModel{
        Id:        getID(id),
        Name:      name,
        Templates: nonNilSlice(templates),
        Fields:    nonNilSlice(fields),
    }
    am.SetCSS(DefaultModelCSS)
    am.SetLatexPre(DefaultModelLatexPre)
    am.SetLatexPost(DefaultModelLatexPost)
    return am
}

func (m *AnkiModel) SetCSS(css string) {
    m.Css = &css
}

func (m *AnkiModel) SetLatexPost(latexPost string) {
    m.LatexPost = &latexPost
}

func (m *AnkiModel) SetLatexPre(latexPre string) {
    m.LatexPre = &latexPre
}

type AnkiNote struct {
    Model     int      `json:"model"`
    Fields    []string `json:"fields"`
    SortField *string  `json:"sort_field"`
    Tags      []string `json:"tags"`
    Guid      *string  `json:"guid"`
}

func NewAnkiNote(model *AnkiModel, fields []string, tags ...string) *AnkiNote {
    return &AnkiNote{
        Model:  model.Id,
        Fields: nonNilSlice(fields),
        Tags:   nonNilSlice(tags),
    }
}

func (n *AnkiNote) SetGuid(guid string) {
    n.Guid = &guid
}

func (n *AnkiNote) SetSortField(sortField string) {
    n.SortField = &sortField
}

type AnkiDeck struct {
    Id          int         `json:"id"`
    Name        string      `json:"name"`
    Description string      `json:"description"`
    Notes       []*AnkiNote `json:"notes"`
}

func NewAnkiDeck(name string, notes []*AnkiNote, id ...int) *AnkiDeck {
    return &AnkiDeck{
        Id:    getID(id),
        Name:  name,
        Notes: notes,
    }
}

type GenerateRequest struct {
    Files  map[string][]byte `json:"files"`
    Decks  []*AnkiDeck       `json:"decks"`
    Models []*AnkiModel      `json:"models"`
}

func NewGenerateRequest(decksModelsFiles ...any) *GenerateRequest {
    ad := &GenerateRequest{
        Decks:  []*AnkiDeck{},
        Models: []*AnkiModel{},
        Files:  map[string][]byte{},
    }

    for i := 0; i < len(decksModelsFiles); i++ {
        switch e := decksModelsFiles[i].(type) {
        case *AnkiDeck:
            ad.Decks = append(ad.Decks, e)
        case *AnkiModel:
            ad.Models = append(ad.Models, e)
        case string:
            if i++; i >= len(decksModelsFiles) {
                panic("expected file after file name")
            }
            ad.AddFile(e, decksModelsFiles[i].([]byte))
        default:
            panic("unexpected type")
        }
    }

    return ad
}

func (r *GenerateRequest) AddFile(name string, data []byte) {
    r.Files[name] = data
}

func (r *GenerateRequest) AddFileFS(fsys fs.FS, name string) error {
    b, err := fs.ReadFile(fsys, name)
    if err != nil {
        return err
    }
    r.AddFile(filepath.Base(name), b)
    return nil
}

func getID(id []int) int {
    if len(id) == 0 {
        return int(rand.Int32())
    }
    return id[0]
}

func nonNilSlice[T any](s []T) []T {
    if s == nil {
        return []T{}
    }
    return s
}

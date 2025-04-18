package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/bobcatalyst/genanki-srv"
    "io"
    "net/http"
    "os"
)

const port = 8000

func main() {
    mod := genanki_srv.NewAnkiModel("Test Model", []*genanki_srv.AnkiModelTemplate{
        genanki_srv.NewAnkiModelTemplate("Test Card", "Front is: {{Front}}", "Back Is: {{Back}}"),
    }, []*genanki_srv.AnkiModelField{
        genanki_srv.NewAnkiModelField("Front"),
        genanki_srv.NewAnkiModelField("Back"),
    })

    deck := genanki_srv.NewAnkiDeck("Test", []*genanki_srv.AnkiNote{
        genanki_srv.NewAnkiNote(mod, []string{"Foo", "Bar"}),
    })

    gen := genanki_srv.NewGenerateRequest(deck, mod)
    b, err := json.Marshal(gen)
    if err != nil {
        panic(err)
    }

    resp, err := http.Post(fmt.Sprintf("http://localhost:%d", port), "application/json", bytes.NewReader(b))
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        _, _ = fmt.Fprintln(os.Stderr, resp.Status, http.StatusText(resp.StatusCode))
        if resp.Header.Get("Content-Type") == "application/json" {
            b, err := io.ReadAll(resp.Body)
            if err != nil {
                panic(err)
            }
            b, err = json.MarshalIndent(json.RawMessage(b), "", "    ")
            if err != nil {
                panic(err)
            }
            _, _ = os.Stderr.Write(b)
        } else {
            _, _ = io.Copy(os.Stderr, resp.Body)
        }
        _, _ = os.Stderr.Write([]byte("\n"))
        return
    }

    f, err := os.Create("test.apkg")
    if err != nil {
        panic(err)
    }
    defer f.Close()

    _, err = io.Copy(f, resp.Body)
    if err != nil {
        panic(err)
    }
}

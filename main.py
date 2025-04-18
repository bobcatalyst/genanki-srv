import base64
import os
import shutil
import tempfile
import time
from typing import Iterable

from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import genanki

app = FastAPI()

class AnkiModelField(BaseModel):
    name: str
    font: str = "Liberation Sans"
    rtl: bool = False
    size: int = 20
    sticky: bool = False

class AnkiModelTemplate(BaseModel):
    name: str
    qfmt: str
    afmt: str
    bqfmt: str = ""
    bafmt: str = ""
    bfont: str = ""
    bsize: int = 0

class AnkiModel(BaseModel):
    id: int
    name: str
    fields: list[AnkiModelField]
    templates: list[AnkiModelTemplate]
    css: str = """.card {
    font-family: arial;
    font-size: 20px;
    text-align: center;
    color: black;
    background-color: white;
}"""
    model_type: int = genanki.Model.FRONT_BACK
    latex_pre: str = genanki.Model.DEFAULT_LATEX_PRE
    latex_post: str = genanki.Model.DEFAULT_LATEX_POST
    sort_field_index: int = 0

class AnkiNote(BaseModel):
    model: int
    fields: list[str]
    sort_field: str|None = None
    tags: list[str] = []
    guid: str|None = None

class AnkiDeck(BaseModel):
    id: int
    name: str
    description: str = ""
    notes: list[AnkiNote]

class GenerateRequest(BaseModel):
    files: dict[str, str]
    decks: list[AnkiDeck]
    models: list[AnkiModel]

def _write_media_files(media_dir: str, files: dict[str, str], pkg: genanki.Package):
    for filename, b64 in files.items():
        filename = os.path.join(media_dir, filename)
        dec = base64.b64decode(b64)
        with open(filename, "wb") as f:
            f.write(dec)
        pkg.media_files.append(filename)

def _force_delete(filename: str):
    try:
        os.remove(filename)
    except FileNotFoundError:
        pass

def _force_delete_dir(dirname: str):
    try:
        shutil.rmtree(dirname)
    except FileNotFoundError:
        pass

def _generate_anki_package(decks: list[genanki.Deck], files: dict[str, str], timestamp: float) -> str:
    # Create dir for media
    media_dir = tempfile.mkdtemp()
    # Create file for package
    pkg_file, pkg_filename = tempfile.mkstemp()
    os.close(pkg_file)
    os.remove(pkg_filename)

    try:
        # Generate anki package
        pkg = genanki.Package(deck_or_decks=decks)
        _write_media_files(media_dir, files, pkg)
        pkg.write_to_file(pkg_filename, timestamp)
    except Exception as e:
        # on exception delete file
        _force_delete(pkg_filename)
        raise e
    finally:
        # anki package zip containing media created, delete directory
        _force_delete_dir(media_dir)
    return pkg_filename

def _iterfile(pkg_file: str) -> Iterable[bytes]:
    try:
        with open(pkg_file, mode="rb") as file_like:
            yield from file_like
    finally:
        _force_delete(pkg_file)

def _generate_decks(request: GenerateRequest) -> list[genanki.Deck]:
    models = {}
    for model in request.models:
        models[model.id] = genanki.Model(
            model_id=model.id,
            name=model.name,
            fields=[{
                "name": field.name,
                "font": field.font,
                "rtl": field.rtl,
                "size": field.size,
                "sticky": field.sticky,
            } for field in model.fields],
            templates=[{
                "name": template.name,
                "qfmt": template.qfmt,
                "afmt": template.afmt,
                "bqfmt": template.bqfmt,
                "bafmt": template.bafmt,
                "bfont": template.bfont,
                "bsize": template.bsize,
            } for template in model.templates],
            css=model.css,
            model_type=model.model_type,
            latex_pre=model.latex_pre,
            latex_post=model.latex_post,
            sort_field_index=model.sort_field_index,
        )

    decks = []
    for rdeck in request.decks:
        deck = genanki.Deck(name=rdeck.name, description=rdeck.description, deck_id=rdeck.id)
        for note in rdeck.notes:
            deck.add_note(genanki.Note(
                model=models[note.model],
                fields=note.fields,
                sort_field=note.sort_field,
                tags=note.tags,
                guid=note.guid if note.guid is not None else genanki.guid_for(note.fields),
            ))
        decks.append(deck)
    return decks

@app.post("/", response_class = StreamingResponse,     responses={
    200: {
        "content": {"application/octet-stream": {}},
        "description": "The generated anki package.",
    }
},)
async def root(request: GenerateRequest, timestamp: float|None = None):
    timestamp = timestamp if timestamp is not None else time.time()
    pkg_file = _generate_anki_package(_generate_decks(request), request.files, timestamp)
    return StreamingResponse(_iterfile(pkg_file), media_type="application/octet-stream")

# Translations

Dank Calendar uses the same translation pipeline as DankMaterialShell: strings
are wrapped in `I18n.tr(term, context)` in QML, extracted to a
POEditor-compatible `en.json`, and translated exports are dropped into
`poexports/` as `<locale>.json` key-value files.

## Adding strings

Wrap user-facing strings at the call site:

```qml
text: I18n.tr("Create event", "sidebar button to create a new event")
```

- The second argument is a short translator context describing where the
  string appears. Identical terms in different roles need distinct contexts.
- Placeholders use `%1`/`%2` with `.arg()`:
  `I18n.tr("Delete \"%1\"?", "...").arg(cal.name)`
- Don't wrap icon names, ids/keys, Qt date format strings, or log messages.
- Month and day names come from `SettingsData.monthName()` /
  `SettingsData.dayName()`, not translations.

## Extracting

```sh
make i18n-extract          # or: python3 extract_translations.py
```

Regenerates `en.json` (POEditor source upload) and `template.json` (empty
template for translators). Run it after adding or changing strings.

`make i18n-local` does the same but prints which terms were added/removed.

## POEditor sync

This repo has its own POEditor project, separate from DankMaterialShell. Set
your own credentials (distinct env var names so the two projects don't clash):

```sh
export DCAL_POEDITOR_API_TOKEN=...   # POEditor account API token
export DCAL_POEDITOR_PROJECT_ID=...  # this project's numeric id
```

Then:

```sh
make i18n-test    # extract + validate, no network
make i18n-sync    # upload en.json terms, download translations, stage changes
make i18n-check   # fail if local i18n is out of sync (for CI / pre-commit)
```

`i18n-sync` uploads new/changed source terms to POEditor, downloads each
configured locale into `poexports/`, and `git add`s the results. The language
list lives in `scripts/i18nsync.py` (`LANGUAGES`) and is empty until a locale
has translations — add `"<poeditor-code>": "<locale>.json"` entries as they
come online.

## Translated exports

`poexports/<locale>.json` files are nested `context -> term -> translation`
objects, e.g.:

```json
{
  "sidebar button to create a new event": {
    "Create event": "Termin erstellen"
  }
}
```

`Common/I18n.qml` picks the file matching the system locale (full tag first,
then language code) and falls back to the built-in English strings.

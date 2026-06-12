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
python3 extract_translations.py
```

Regenerates `en.json` (POEditor source upload) and `template.json` (empty
template for translators). Run it after adding or changing strings.

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
then language code) and falls back to the built-in English strings. There is
no POEditor project for this repo yet; once one exists, exports go straight
into `poexports/`.

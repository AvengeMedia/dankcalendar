pragma Singleton
pragma ComponentBehavior: Bound

import QtQuick
import Qt.labs.folderlistmodel
import Quickshell
import Quickshell.Io
import qs.Services

Singleton {
    id: root

    readonly property var log: Log.scoped("I18n")

    property string _resolvedLocale: "en"

    readonly property string _rawLocale: Qt.locale().name
    readonly property string _lang: _rawLocale.split(/[_-]/)[0]
    readonly property var _candidates: {
        const fullUnderscore = _rawLocale;
        const fullHyphen = _rawLocale.replace("_", "-");
        return [fullUnderscore, fullHyphen, _lang].filter(c => c && c !== "en");
    }

    readonly property var _rtlLanguages: ["ar", "he", "iw", "fa", "ur", "ps", "sd", "dv", "yi", "ku"]
    readonly property bool isRtl: _rtlLanguages.includes(_lang)

    readonly property url translationsFolder: Qt.resolvedUrl("../translations/poexports")
    readonly property url commonTranslationsFolder: Qt.resolvedUrl("../DankCommon/translations/poexports")

    property var presentLocales: ({
            "en": Qt.locale("en")
        })
    property var translations: ({})
    property bool translationsLoaded: false
    property var commonTranslations: ({})
    property bool commonTranslationsLoaded: false

    property url _selectedPath: ""
    property url _commonSelectedPath: ""

    FolderListModel {
        id: dir
        folder: root.translationsFolder
        nameFilters: ["*.json"]
        showDirs: false
        showDotAndDotDot: false

        onStatusChanged: if (status === FolderListModel.Ready) {
            root._loadPresentLocales();
            root._pickTranslation();
        }
    }

    FolderListModel {
        id: commonDir
        folder: root.commonTranslationsFolder
        nameFilters: ["*.json"]
        showDirs: false
        showDotAndDotDot: false

        onStatusChanged: if (status === FolderListModel.Ready) {
            root._pickCommonTranslation();
        }
    }

    FileView {
        id: translationLoader
        path: root._selectedPath

        onLoaded: {
            try {
                root.translations = JSON.parse(text());
                root.translationsLoaded = true;
                root.log.info(`Loaded translations for '${root._resolvedLocale}' (${Object.keys(root.translations).length} contexts)`);
            } catch (e) {
                root.log.warn(`Error parsing '${root._resolvedLocale}':`, e, "- falling back to English");
                root._fallbackToEnglish();
            }
        }

        onLoadFailed: error => {
            root.log.warn(`Failed to load '${root._resolvedLocale}' (${error}), falling back to English`);
            root._fallbackToEnglish();
        }
    }

    FileView {
        id: commonTranslationLoader
        path: root._commonSelectedPath
        printErrors: false

        onLoaded: {
            try {
                root.commonTranslations = JSON.parse(text());
                root.commonTranslationsLoaded = true;
                root.log.info(`Loaded DankCommon translations (${Object.keys(root.commonTranslations).length} contexts)`);
            } catch (e) {
                root.log.warn("Error parsing DankCommon translations:", e);
            }
        }
    }

    function _loadPresentLocales() {
        if (Object.keys(presentLocales).length > 1)
            return;
        for (let i = 0; i < dir.count; i++) {
            const name = dir.get(i, "fileName");
            if (name && name.endsWith(".json")) {
                const shortName = name.slice(0, -5);
                presentLocales[shortName] = Qt.locale(shortName);
            }
        }
    }

    function _pickTranslation() {
        for (let i = 0; i < _candidates.length; i++) {
            const cand = _candidates[i];
            if (presentLocales[cand] === undefined)
                continue;
            _resolvedLocale = cand;
            useLocale(cand, cand.startsWith("en") ? "" : translationsFolder + "/" + cand + ".json");
            return;
        }

        _resolvedLocale = "en";
        _fallbackToEnglish();
    }

    function useLocale(localeTag, fileUrl) {
        _resolvedLocale = localeTag || "en";
        _selectedPath = fileUrl;
        translationsLoaded = false;
        translations = ({});
        log.info(`Using locale '${localeTag}' from ${fileUrl}`);
    }

    function _fallbackToEnglish() {
        _selectedPath = "";
        translationsLoaded = false;
        translations = ({});
    }

    function _pickCommonTranslation() {
        const present = {};
        for (let i = 0; i < commonDir.count; i++) {
            const name = commonDir.get(i, "fileName");
            if (name && name.endsWith(".json"))
                present[name.slice(0, -5)] = true;
        }
        for (let i = 0; i < _candidates.length; i++) {
            if (!present[_candidates[i]])
                continue;
            _commonSelectedPath = commonTranslationsFolder + "/" + _candidates[i] + ".json";
            return;
        }
    }

    function _lookup(table, term, context) {
        if (!table)
            return "";
        const ctx = context || term;
        if (table[ctx] && table[ctx][term])
            return table[ctx][term];
        for (const c in table) {
            if (table[c] && table[c][term])
                return table[c][term];
        }
        return "";
    }

    function tr(term, context) {
        if (translationsLoaded) {
            const hit = _lookup(translations, term, context);
            if (hit)
                return hit;
        }
        if (commonTranslationsLoaded) {
            const hit = _lookup(commonTranslations, term, context);
            if (hit)
                return hit;
        }
        return term;
    }

    function trContext(context, term) {
        if (translationsLoaded && translations[context] && translations[context][term])
            return translations[context][term];
        if (commonTranslationsLoaded && commonTranslations[context] && commonTranslations[context][term])
            return commonTranslations[context][term];
        return term;
    }
}

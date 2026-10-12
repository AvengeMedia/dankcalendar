// Run with: node scripts/test-matugen.mjs
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

// Exercise the actual QML JavaScript helpers without requiring a desktop session.
const theme = readFileSync(new URL('../quickshell/Common/Theme.qml', import.meta.url), 'utf8');
const context = { customThemeRaw: null };
for (const name of ['validateMatugenTheme', 'customColorsForMode', 'buildCustomTheme']) {
    const source = theme.match(new RegExp(`    function ${name}\\([^]*?\\n    }`));
    assert.ok(source, `Missing theme helper: ${name}`);
    runInNewContext(source[0], context);
}

const fallback = { primary: '#123456', background: '#101010' };
const palette = { dark: { primary: '#abcdef' }, light: { primary: '#654321' } };
context.customThemeRaw = context.validateMatugenTheme(palette);
assert.equal(context.buildCustomTheme(fallback, false).primary, '#abcdef');
assert.equal(context.buildCustomTheme(fallback, true).primary, '#654321');
assert.equal(context.buildCustomTheme(fallback, false).background, '#101010');
assert.equal(fallback.primary, '#123456');

context.customThemeRaw = context.validateMatugenTheme({ primary: '#AaBbCc' });
assert.equal(context.buildCustomTheme(fallback, true).primary, '#AaBbCc');
context.customThemeRaw = context.validateMatugenTheme({ dark: { primary: '#abcdef' } });
assert.equal(context.buildCustomTheme(fallback, true).primary, '#abcdef');
context.customThemeRaw = null;
assert.equal(context.buildCustomTheme(fallback, false), fallback);

for (const invalid of [null, [], {}, true, 'red', 42, { primary: 42 },
    { primary: '#xyzxyz' }, { primary: '' }, { primary: '#12345678' },
    { dark: [] }, { dark: null }, { dark: {} }, { light: { primary: [] } }]) {
    assert.throws(() => context.validateMatugenTheme(invalid));
}

// The bundled template must produce valid app palettes for both modes.
const template = readFileSync(new URL('../assets/matugen/dankcalendar.json.template', import.meta.url), 'utf8');
const rendered = template.replace(/\{\{colors\.[a-z_]+\.(dark|light)\.hex\}\}/g,
    (_, mode) => mode === 'dark' ? '#112233' : '#ddeeff');
context.customThemeRaw = context.validateMatugenTheme(JSON.parse(rendered));
for (const light of [false, true]) {
    const colors = context.buildCustomTheme(fallback, light);
    for (const role of ['background', 'surface', 'surfaceText', 'primary', 'secondary', 'outline', 'error'])
        assert.equal(colors[role], light ? '#ddeeff' : '#112233');
}
console.log('Matugen palette validation, fallback, modes, and template checks passed.');

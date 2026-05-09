/* Settings page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    let draft = RS.loadSettings();

    function $(s) { return document.querySelector(s); }
    function $$(s) { return document.querySelectorAll(s); }

    function render() {
        $('#set-brand-name').value    = draft.brandName;
        $('#set-brand-tagline').value = draft.tagline;
        $('#set-page-title').value    = draft.pageTitle;
        $('#set-accent-color').value  = draft.accent;
        $('#set-accent-hex').value    = draft.accent.toUpperCase();
        updateLogoPreview();
        updateSwatches();
        updateThemeToggle();
    }
    function updateLogoPreview() {
        const el = $('#set-logo-preview');
        if (draft.logoData) {
            el.innerHTML = `<img src="${draft.logoData}" alt="logo">`;
        } else {
            el.textContent = (draft.brandName || 'R').trim().charAt(0).toUpperCase() || 'R';
        }
    }
    function updateSwatches() {
        $$('#set-accent-swatches .color-swatch').forEach((sw) => {
            sw.classList.toggle('active', sw.dataset.color.toLowerCase() === draft.accent.toLowerCase());
        });
    }
    function updateThemeToggle() {
        $$('#set-theme-toggle button').forEach((btn) => {
            btn.classList.toggle('active', btn.dataset.theme === draft.theme);
        });
    }

    function bindTextInput(id, key) {
        const el = $(id);
        el.addEventListener('input', () => {
            draft[key] = el.value;
            if (key === 'brandName') updateLogoPreview();
            RS.applySettings(draft);
        });
    }
    bindTextInput('#set-brand-name',    'brandName');
    bindTextInput('#set-brand-tagline', 'tagline');
    bindTextInput('#set-page-title',    'pageTitle');

    const colorEl = $('#set-accent-color');
    const hexEl   = $('#set-accent-hex');
    colorEl.addEventListener('input', () => {
        draft.accent = colorEl.value;
        hexEl.value = colorEl.value.toUpperCase();
        RS.applySettings(draft);
        updateSwatches();
    });
    hexEl.addEventListener('change', () => {
        const v = hexEl.value.trim();
        if (!/^#?[0-9a-fA-F]{6}$/.test(v)) {
            hexEl.value = draft.accent.toUpperCase();
            return;
        }
        const norm = v.startsWith('#') ? v : '#' + v;
        draft.accent = norm.toLowerCase();
        colorEl.value = draft.accent;
        hexEl.value = norm.toUpperCase();
        RS.applySettings(draft);
        updateSwatches();
    });

    $$('#set-accent-swatches .color-swatch').forEach((sw) => {
        sw.addEventListener('click', () => {
            draft.accent = sw.dataset.color;
            colorEl.value = sw.dataset.color;
            hexEl.value = sw.dataset.color.toUpperCase();
            RS.applySettings(draft);
            updateSwatches();
        });
    });
    $$('#set-theme-toggle button').forEach((btn) => {
        btn.addEventListener('click', () => {
            draft.theme = btn.dataset.theme;
            RS.applySettings(draft);
            updateThemeToggle();
        });
    });

    const fileEl = $('#set-logo-file');
    $('#btn-logo-upload').addEventListener('click', () => fileEl.click());
    fileEl.addEventListener('change', () => {
        const file = fileEl.files && fileEl.files[0];
        if (!file) return;
        if (file.size > 512 * 1024) {
            RS.showToast('Logo must be under 512 KB', 'error');
            return;
        }
        const reader = new FileReader();
        reader.onload = () => {
            draft.logoData = reader.result;
            updateLogoPreview();
            RS.applySettings(draft);
        };
        reader.readAsDataURL(file);
    });
    $('#btn-logo-clear').addEventListener('click', () => {
        draft.logoData = '';
        fileEl.value = '';
        updateLogoPreview();
        RS.applySettings(draft);
    });

    $('#btn-settings-save').addEventListener('click', () => {
        RS.saveSettings(draft);
        RS.applySettings(draft);
        RS.showToast('Settings saved', 'success');
    });
    $('#btn-settings-reset').addEventListener('click', () => {
        if (!confirm('Reset all branding and appearance settings to defaults?')) return;
        draft = { ...RS.DEFAULT_SETTINGS };
        RS.saveSettings(draft);
        RS.applySettings(draft);
        render();
        RS.showToast('Settings reset to defaults', 'success');
    });

    render();
})();

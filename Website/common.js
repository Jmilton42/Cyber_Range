/* ============================================================
   Range Studio — Common shell
   Loaded on every authenticated page. Provides:
     - Branding/settings (applied before paint to avoid flash)
     - Sidebar injection + active-nav highlighting
     - API helper, toast, command palette, context menu
     - Page version + mock-mode banner
   Pages opt in by setting <body data-page="projects"> etc.
   ============================================================ */

(function () {
    'use strict';

    const API_BASE = window.location.origin;
    const SETTINGS_KEY = 'rangestudio.settings';

    const DEFAULT_SETTINGS = {
        brandName: 'Range Studio',
        tagline:   'InSPIRE Platform',
        pageTitle: 'Range Studio — InSPIRE Cyber Range',
        accent:    '#2563eb',
        theme:     'dark',
        logoData:  '',
    };

    // Pages keyed by data-page on <body>. The href is a real URL so the
    // browser handles navigation — auth gating becomes a per-page concern.
    const NAV_ITEMS = [
        { id: 'projects',  href: 'index.html',     label: 'Projects',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><rect x="1" y="1" width="7" height="7" rx="1.5" stroke="currentColor" stroke-width="1.5"/><rect x="10" y="1" width="7" height="7" rx="1.5" stroke="currentColor" stroke-width="1.5"/><rect x="1" y="10" width="7" height="7" rx="1.5" stroke="currentColor" stroke-width="1.5"/><rect x="10" y="10" width="7" height="7" rx="1.5" stroke="currentColor" stroke-width="1.5"/></svg>' },
        { id: 'images',    href: 'images.html',    label: 'Images',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><rect x="1" y="3" width="16" height="12" rx="2" stroke="currentColor" stroke-width="1.5"/><circle cx="6" cy="8" r="1.5" stroke="currentColor" stroke-width="1.2"/><path d="M1 13l4-3 3 2 4-4 5 5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>' },
        { id: 'profiles',  href: 'profiles.html',  label: 'Profiles',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><path d="M3 3h12v2H3zM3 8h12v2H3zM3 13h12v2H3z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>' },
        { id: 'cluster',   href: 'cluster.html',   label: 'Cluster',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><circle cx="9" cy="9" r="2" stroke="currentColor" stroke-width="1.5"/><circle cx="9" cy="3" r="1.5" stroke="currentColor" stroke-width="1.2"/><circle cx="15" cy="9" r="1.5" stroke="currentColor" stroke-width="1.2"/><circle cx="9" cy="15" r="1.5" stroke="currentColor" stroke-width="1.2"/><circle cx="3" cy="9" r="1.5" stroke="currentColor" stroke-width="1.2"/><line x1="9" y1="5" x2="9" y2="7" stroke="currentColor" stroke-width="1"/><line x1="13" y1="9" x2="11" y2="9" stroke="currentColor" stroke-width="1"/><line x1="9" y1="13" x2="9" y2="11" stroke="currentColor" stroke-width="1"/><line x1="5" y1="9" x2="7" y2="9" stroke="currentColor" stroke-width="1"/></svg>' },
        { id: 'templates', href: 'templates.html', label: 'Templates',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><path d="M4 2h7l5 5v9a2 2 0 01-2 2H4a2 2 0 01-2-2V4a2 2 0 012-2z" stroke="currentColor" stroke-width="1.5"/><path d="M11 2v5h5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>' },
        { id: 'settings',  href: 'settings.html',  label: 'Settings',
          icon: '<svg width="18" height="18" viewBox="0 0 18 18" fill="none"><circle cx="9" cy="9" r="2.2" stroke="currentColor" stroke-width="1.5"/><path d="M14.5 9a5.5 5.5 0 00-.08-.93l1.4-1.07-1.5-2.6-1.65.62a5.5 5.5 0 00-1.6-.93L10.5 2.5h-3l-.57 1.59a5.5 5.5 0 00-1.6.93l-1.65-.62-1.5 2.6 1.4 1.07A5.5 5.5 0 003.5 9c0 .32.03.63.08.93l-1.4 1.07 1.5 2.6 1.65-.62c.47.4 1.01.7 1.6.93L7.5 15.5h3l.57-1.59a5.5 5.5 0 001.6-.93l1.65.62 1.5-2.6-1.4-1.07c.05-.3.08-.61.08-.93z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"/></svg>' },
    ];

    // ---------- Settings ----------
    function loadSettings() {
        try {
            const raw = localStorage.getItem(SETTINGS_KEY);
            if (!raw) return { ...DEFAULT_SETTINGS };
            return { ...DEFAULT_SETTINGS, ...JSON.parse(raw) };
        } catch {
            return { ...DEFAULT_SETTINGS };
        }
    }
    function saveSettings(s) {
        localStorage.setItem(SETTINGS_KEY, JSON.stringify(s));
    }
    function hexToRgb(hex) {
        const h = hex.replace('#', '');
        const n = parseInt(h.length === 3 ? h.split('').map((c) => c + c).join('') : h, 16);
        return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
    }
    function readableFg(hex) {
        const [r, g, b] = hexToRgb(hex);
        const l = (0.299 * r + 0.587 * g + 0.114 * b) / 255;
        return l > 0.6 ? '#0b0d10' : '#ffffff';
    }
    function darken(hex, amount) {
        const [r, g, b] = hexToRgb(hex);
        const f = 1 - amount;
        const to = (v) => Math.max(0, Math.min(255, Math.round(v * f)));
        return '#' + [to(r), to(g), to(b)].map((v) => v.toString(16).padStart(2, '0')).join('');
    }
    function applySettings(s) {
        document.documentElement.setAttribute('data-theme', s.theme === 'light' ? 'light' : 'dark');
        const root = document.documentElement;
        const [r, g, b] = hexToRgb(s.accent);
        root.style.setProperty('--accent', s.accent);
        root.style.setProperty('--accent-hover', darken(s.accent, 0.12));
        root.style.setProperty('--accent-soft', `rgba(${r}, ${g}, ${b}, 0.10)`);
        root.style.setProperty('--accent-ring', `rgba(${r}, ${g}, ${b}, 0.22)`);
        root.style.setProperty('--accent-fg', readableFg(s.accent));
        if (s.pageTitle && !document.title) document.title = s.pageTitle;

        const titleEl = document.getElementById('brand-title');
        const tagEl   = document.getElementById('brand-tagline');
        if (titleEl) titleEl.textContent = s.brandName || DEFAULT_SETTINGS.brandName;
        if (tagEl)   tagEl.textContent   = s.tagline   || DEFAULT_SETTINGS.tagline;

        const logoEl = document.getElementById('brand-logo-icon');
        if (logoEl) {
            if (s.logoData) {
                logoEl.innerHTML = `<img src="${s.logoData}" alt="logo">`;
            } else {
                const ch = (s.brandName || 'R').trim().charAt(0).toUpperCase();
                logoEl.textContent = ch || 'R';
            }
        }
    }
    // Apply theme + accent immediately so first paint is correct.
    applySettings(loadSettings());

    // ---------- API ----------
    async function api(path, opts) {
        const res = await fetch(API_BASE + path, opts);
        if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
        return res.json();
    }

    // ---------- Helpers ----------
    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    function escapeAttr(str) {
        return (str || '').replace(/'/g, "\\'").replace(/"/g, '&quot;');
    }
    function formatDate(iso) {
        if (!iso) return '—';
        try {
            return new Date(iso).toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
        } catch { return iso; }
    }
    function formatBytes(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        const units = ['GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
    }
    function truncatePath(path) {
        if (!path) return '';
        if (path.length <= 45) return path;
        return '…' + path.slice(-42);
    }

    // ---------- Toast ----------
    let toastTimeout;
    function showToast(message, type) {
        const toast = document.getElementById('rs-toast');
        if (!toast) return;
        toast.className = 'toast ' + (type || '');
        toast.textContent = message;
        toast.classList.remove('hidden');
        clearTimeout(toastTimeout);
        toastTimeout = setTimeout(() => toast.classList.add('hidden'), 2500);
    }

    // ---------- Command Palette ----------
    function getCommands(projectName) {
        return [
            { label: 'Status',        command: 'forge status',        type: 'forge', desc: 'Show subnet allocation status' },
            { label: 'Plan',          command: 'forge plan',          type: 'forge', desc: 'Preview infrastructure changes' },
            { label: 'Apply',         command: 'forge apply',         type: 'forge', desc: 'Apply infrastructure changes' },
            { label: 'Destroy',       command: 'forge destroy',       type: 'forge', desc: 'Tear down infrastructure' },
            { label: 'Doctor',        command: 'forge doctor',        type: 'forge', desc: 'Run preflight checks' },
            { label: 'Validate',      command: 'tofu validate',       type: 'tofu',  desc: 'Validate HCL configuration' },
            { label: 'Init',          command: 'tofu init',           type: 'tofu',  desc: 'Initialize working directory' },
            { label: 'Subnets List',  command: 'forge subnets list',  type: 'forge', desc: 'List all subnet allocations' },
        ];
    }
    function openCommandPalette(projectName) {
        const palette = document.getElementById('rs-palette');
        if (!palette) return;
        document.getElementById('rs-palette-project').textContent = projectName || '';
        const cmds = getCommands(projectName);
        const forge = cmds.filter((c) => c.type === 'forge');
        const tofu  = cmds.filter((c) => c.type === 'tofu');
        const item = (c) => `
            <div class="command-item" data-cmd="${escapeAttr(c.command)}">
                <div class="command-icon ${c.type}">${c.type === 'forge' ? '⚒' : '🔧'}</div>
                <div class="command-text">
                    <div class="command-name">${escapeHtml(c.label)}</div>
                    <div class="command-desc">${escapeHtml(c.desc)}</div>
                </div>
                <button class="command-copy" data-cmd="${escapeAttr(c.command)}">Copy</button>
            </div>`;
        document.getElementById('rs-palette-body').innerHTML = `
            <div class="command-group-label">Forge</div>
            ${forge.map(item).join('')}
            <div class="command-group-label" style="margin-top: 8px;">OpenTofu</div>
            ${tofu.map(item).join('')}`;
        palette.classList.remove('hidden');
    }
    function closeCommandPalette() {
        const palette = document.getElementById('rs-palette');
        if (palette) palette.classList.add('hidden');
    }

    // ---------- Context Menu (project cards) ----------
    function showContextMenu(x, y, projectName) {
        const menu = document.getElementById('rs-ctx');
        if (!menu) return;
        const cmds = getCommands(projectName);
        const items = document.getElementById('rs-ctx-items');
        items.innerHTML = `
            <div class="context-item" data-action="open" data-name="${escapeAttr(projectName)}">
                <span class="context-item-icon">📋</span><span>View Details</span>
            </div>
            <div class="context-separator"></div>
            ${cmds.map((c) => `
                <div class="context-item" data-action="copy" data-cmd="${escapeAttr(c.command)}">
                    <span class="context-item-icon">${c.type === 'forge' ? '⚒' : '🔧'}</span>
                    <span>${escapeHtml(c.label)}</span>
                </div>
            `).join('')}
            <div class="context-separator"></div>
            <div class="context-item" data-action="palette" data-name="${escapeAttr(projectName)}">
                <span class="context-item-icon">⌘</span><span>Command Palette</span>
            </div>`;
        menu.style.left = Math.min(x, window.innerWidth - 240) + 'px';
        menu.style.top  = Math.min(y, window.innerHeight - 300) + 'px';
        menu.classList.remove('hidden');
    }
    function hideContextMenu() {
        const menu = document.getElementById('rs-ctx');
        if (menu) menu.classList.add('hidden');
    }

    // ---------- Clipboard ----------
    async function copyCommand(command) {
        try {
            await navigator.clipboard.writeText(command);
        } catch {
            const ta = document.createElement('textarea');
            ta.value = command;
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            document.body.removeChild(ta);
        }
        showToast(`Copied: ${command}`, 'success');
        hideContextMenu();
        closeCommandPalette();
    }

    // ---------- Sidebar + floating UI injection ----------
    function buildSidebar(activeId) {
        return `
            <div class="sidebar-header">
                <a class="logo" href="index.html" style="text-decoration:none;color:inherit;">
                    <div class="logo-icon" id="brand-logo-icon">R</div>
                    <div class="logo-text">
                        <span class="logo-title" id="brand-title">Range Studio</span>
                        <span class="logo-sub" id="brand-tagline">InSPIRE Platform</span>
                    </div>
                </a>
            </div>
            <nav class="sidebar-nav">
                ${NAV_ITEMS.map((n) => `
                    <a class="nav-item ${n.id === activeId ? 'active' : ''}" href="${n.href}">
                        ${n.icon}<span>${n.label}</span>
                    </a>
                `).join('')}
            </nav>
            <div class="sidebar-footer">
                <div class="sidebar-version" id="sidebar-version">v…</div>
            </div>`;
    }

    function buildFloatingUI() {
        const wrap = document.createElement('div');
        wrap.innerHTML = `
            <div id="rs-toast" class="toast hidden"></div>
            <div id="rs-palette" class="command-palette hidden">
                <div class="command-palette-backdrop" id="rs-palette-backdrop"></div>
                <div class="command-palette-dialog">
                    <div class="command-palette-header">
                        <svg width="18" height="18" viewBox="0 0 18 18" fill="none"><path d="M5 3l6 5-6 5" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                        <span>Command Palette</span>
                        <span class="command-palette-project" id="rs-palette-project"></span>
                        <button class="command-palette-close" id="rs-palette-close">
                            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                        </button>
                    </div>
                    <div class="command-palette-body" id="rs-palette-body"></div>
                </div>
            </div>
            <div id="rs-ctx" class="context-menu hidden">
                <div class="context-menu-items" id="rs-ctx-items"></div>
            </div>`;
        document.body.appendChild(wrap);
    }

    function bindFloatingUI() {
        document.getElementById('rs-palette-close').addEventListener('click', closeCommandPalette);
        document.getElementById('rs-palette-backdrop').addEventListener('click', closeCommandPalette);

        document.getElementById('rs-palette-body').addEventListener('click', (e) => {
            const target = e.target.closest('[data-cmd]');
            if (target) copyCommand(target.dataset.cmd);
        });
        document.getElementById('rs-ctx-items').addEventListener('click', (e) => {
            const target = e.target.closest('.context-item');
            if (!target) return;
            const action = target.dataset.action;
            if (action === 'copy')    copyCommand(target.dataset.cmd);
            if (action === 'palette') openCommandPalette(target.dataset.name);
            if (action === 'open')    window.location.href = 'project.html?name=' + encodeURIComponent(target.dataset.name);
        });
        document.addEventListener('click', (e) => {
            if (!e.target.closest('#rs-ctx')) hideContextMenu();
        });
        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                openCommandPalette(window.RangeStudio._currentProject || '');
            }
            if (e.key === 'Escape') {
                closeCommandPalette();
                hideContextMenu();
            }
        });
    }

    async function initShell() {
        const activeId = document.body.dataset.page || '';
        const sidebar = document.getElementById('sidebar');
        if (sidebar) sidebar.innerHTML = buildSidebar(activeId);

        // Re-apply settings now that the sidebar DOM exists
        applySettings(loadSettings());

        buildFloatingUI();
        bindFloatingUI();

        // Mock banner + version
        try {
            const info = await api('/api/info');
            const versionEl = document.getElementById('sidebar-version');
            if (versionEl) versionEl.textContent = 'v' + info.version;
            if (info.mock) {
                const banner = document.getElementById('rs-mock-banner');
                if (banner) banner.classList.remove('hidden');
                document.body.classList.add('mock-active');
            }
        } catch (err) {
            console.warn('Failed to fetch /api/info:', err);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initShell);
    } else {
        initShell();
    }

    // ---------- Public API ----------
    window.RangeStudio = {
        api,
        showToast,
        escapeHtml,
        escapeAttr,
        formatDate,
        formatBytes,
        truncatePath,
        loadSettings,
        saveSettings,
        applySettings,
        DEFAULT_SETTINGS,
        getCommands,
        openCommandPalette,
        closeCommandPalette,
        showContextMenu,
        hideContextMenu,
        copyCommand,
        _currentProject: '',
    };
})();

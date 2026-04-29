/* ============================================================
   Range Studio — Application Logic
   Vanilla JS SPA for the InSPIRE Cyber Range platform.
   ============================================================ */

(function () {
    'use strict';

    // ---------- Configuration ----------
    const API_BASE = window.location.origin;

    // ---------- State ----------
    const state = {
        mock: false,
        projects: [],
        images: [],
        profiles: [],
        cluster: [],
        templates: [],
        currentView: 'projects',
        selectedProject: null,
    };

    // ---------- DOM Helpers ----------
    const $ = (sel) => document.querySelector(sel);
    const $$ = (sel) => document.querySelectorAll(sel);

    // ---------- API ----------
    async function api(path) {
        const res = await fetch(API_BASE + path);
        if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
        return res.json();
    }

    // ---------- Init ----------
    async function init() {
        try {
            const info = await api('/api/info');
            state.mock = info.mock;
            $('#sidebar-version').textContent = 'v' + info.version;

            if (state.mock) {
                $('#mock-banner').classList.remove('hidden');
                document.body.classList.add('mock-active');
            }

            await loadProjects();
            showView('projects');
            $('#loading').classList.add('hidden');
        } catch (err) {
            console.error('Init failed:', err);
            $('#loading').innerHTML = `
                <div style="color: var(--red); text-align: center;">
                    <p style="font-size: 1.1rem; font-weight: 600; margin-bottom: 8px;">Connection Failed</p>
                    <p style="color: var(--text-muted); font-size: 0.88rem;">Could not reach the Range Studio API.</p>
                    <p style="color: var(--text-dim); font-size: 0.8rem; margin-top: 4px; font-family: var(--font-mono);">${err.message}</p>
                </div>`;
        }
    }

    // ---------- Data Loaders ----------
    async function loadProjects() {
        state.projects = await api('/api/projects');
    }

    async function loadImages() {
        if (state.images.length === 0) {
            state.images = await api('/api/lxd/images');
        }
    }

    async function loadProfiles() {
        if (state.profiles.length === 0) {
            state.profiles = await api('/api/lxd/profiles');
        }
    }

    async function loadCluster() {
        if (state.cluster.length === 0) {
            state.cluster = await api('/api/lxd/cluster');
        }
    }

    async function loadTemplates() {
        if (state.templates.length === 0) {
            state.templates = await api('/api/templates');
        }
    }

    // ---------- View Router ----------
    async function showView(view) {
        state.currentView = view;

        // Hide all views
        $$('.view').forEach((v) => v.classList.add('hidden'));

        // Update nav
        $$('.nav-item').forEach((n) => n.classList.remove('active'));
        const navBtn = $(`[data-view="${view}"]`);
        if (navBtn) navBtn.classList.add('active');

        switch (view) {
            case 'projects':
                renderProjects();
                $('#view-projects').classList.remove('hidden');
                break;
            case 'project-detail':
                renderProjectDetail();
                $('#view-project-detail').classList.remove('hidden');
                break;
            case 'images':
                await loadImages();
                renderImages();
                $('#view-images').classList.remove('hidden');
                break;
            case 'profiles':
                await loadProfiles();
                renderProfiles();
                $('#view-profiles').classList.remove('hidden');
                break;
            case 'cluster':
                await loadCluster();
                renderCluster();
                $('#view-cluster').classList.remove('hidden');
                break;
            case 'templates':
                await loadTemplates();
                renderTemplates();
                $('#view-templates').classList.remove('hidden');
                break;
        }
    }

    // ---------- Render: Projects ----------
    function renderProjects() {
        const grid = $('#projects-grid');
        if (state.projects.length === 0) {
            grid.innerHTML = `
                <div style="grid-column: 1 / -1; text-align: center; padding: 60px 20px; color: var(--text-muted);">
                    <p style="font-size: 1.1rem; margin-bottom: 8px;">No projects found</p>
                    <p style="font-size: 0.84rem;">Allocations will appear here once subnets.json is populated.</p>
                </div>`;
            return;
        }

        grid.innerHTML = state.projects
            .map(
                (p) => `
            <div class="project-card" data-project="${escapeHtml(p.name)}" id="project-card-${escapeHtml(p.name)}">
                <div class="project-card-header">
                    <span class="project-name">${escapeHtml(p.name)}</span>
                    <span class="project-status status-${p.status}">
                        <span class="status-dot"></span>
                        ${p.status === 'missing_dir' ? 'No Dir' : p.status}
                    </span>
                </div>
                <div class="project-meta">
                    <div class="project-meta-row">
                        <span class="meta-label">Subnet</span>
                        <span class="meta-value subnet">${escapeHtml(p.subnet)}</span>
                    </div>
                    <div class="project-meta-row">
                        <span class="meta-label">Gateway</span>
                        <span class="meta-value">${escapeHtml(p.gateway)}</span>
                    </div>
                    <div class="project-meta-row">
                        <span class="meta-label">Octet</span>
                        <span class="meta-value">${p.subnet_octet}</span>
                    </div>
                    <div class="project-meta-row">
                        <span class="meta-label">Allocated</span>
                        <span class="meta-value">${formatDate(p.allocated_at)}</span>
                    </div>
                    ${p.work_dir ? `
                    <div class="project-meta-row">
                        <span class="meta-label">Path</span>
                        <span class="meta-value path" title="${escapeHtml(p.work_dir)}">${truncatePath(p.work_dir)}</span>
                    </div>` : ''}
                    ${p.has_main_tf ? '<div class="project-meta-row"><span class="meta-label"></span><span class="badge-main-tf">main.tf ✓</span></div>' : ''}
                </div>
            </div>`
            )
            .join('');

        // Click handlers
        grid.querySelectorAll('.project-card').forEach((card) => {
            card.addEventListener('click', () => {
                const name = card.dataset.project;
                state.selectedProject = state.projects.find((p) => p.name === name);
                showView('project-detail');
            });
            card.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                const name = card.dataset.project;
                state.selectedProject = state.projects.find((p) => p.name === name);
                showContextMenu(e.clientX, e.clientY, name);
            });
        });
    }

    // ---------- Render: Project Detail ----------
    async function renderProjectDetail() {
        const p = state.selectedProject;
        if (!p) return;

        $('#detail-project-name').textContent = p.name;
        $('#detail-subtitle').textContent = `${p.subnet} • Octet ${p.subnet_octet}`;

        let detail;
        try {
            detail = await api(`/api/projects/${encodeURIComponent(p.name)}`);
        } catch (err) {
            detail = { ...p, files: [], has_state: false };
        }

        const content = $('#detail-content');
        content.innerHTML = `
            <!-- Info Card -->
            <div class="detail-card">
                <div class="detail-card-title">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M8 5v3M8 10v1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                    Project Info
                </div>
                <div class="detail-info-grid">
                    <span class="detail-info-label">Name</span>
                    <span class="detail-info-value">${escapeHtml(detail.name)}</span>
                    <span class="detail-info-label">Subnet</span>
                    <span class="detail-info-value" style="color: var(--cyan);">${escapeHtml(detail.subnet)}</span>
                    <span class="detail-info-label">Gateway</span>
                    <span class="detail-info-value">${escapeHtml(detail.gateway)}</span>
                    <span class="detail-info-label">Octet</span>
                    <span class="detail-info-value">${detail.subnet_octet}</span>
                    <span class="detail-info-label">Allocated</span>
                    <span class="detail-info-value">${formatDate(detail.allocated_at)}</span>
                    <span class="detail-info-label">Status</span>
                    <span class="detail-info-value"><span class="project-status status-${detail.status}" style="font-size: 0.72rem;">${detail.status}</span></span>
                    <span class="detail-info-label">Work Dir</span>
                    <span class="detail-info-value">${detail.work_dir ? escapeHtml(detail.work_dir) : '<span style="color:var(--text-dim)">not resolved</span>'}</span>
                    <span class="detail-info-label">main.tf</span>
                    <span class="detail-info-value">${detail.has_main_tf ? '<span style="color:var(--green)">✓ present</span>' : '<span style="color:var(--text-dim)">—</span>'}</span>
                    <span class="detail-info-label">TF State</span>
                    <span class="detail-info-value">${detail.has_state ? '<span style="color:var(--green)">✓ exists</span>' : '<span style="color:var(--text-dim)">none</span>'}</span>
                </div>
            </div>

            <!-- Files Card -->
            <div class="detail-card">
                <div class="detail-card-title">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M2 4a2 2 0 012-2h3l2 2h3a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V4z" stroke="currentColor" stroke-width="1.5"/></svg>
                    Files
                </div>
                ${detail.files && detail.files.length > 0 ? `
                <div class="file-list">
                    ${detail.files.map((f) => `
                        <div class="file-item">
                            <span class="file-icon ${f.is_dir ? 'dir' : ''}">${f.is_dir ? '📁' : '📄'}</span>
                            <span>${escapeHtml(f.name)}</span>
                            ${!f.is_dir ? `<span class="file-size">${formatBytes(f.size)}</span>` : ''}
                        </div>
                    `).join('')}
                </div>` : `
                <p style="color: var(--text-muted); font-size: 0.84rem;">No directory resolved for this project.</p>`}
            </div>

            <!-- Commands Card -->
            <div class="detail-card full-width">
                <div class="detail-card-title">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M5 3l6 5-6 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                    Quick Commands
                </div>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 8px;">
                    ${getCommands(detail.name).map((cmd) => `
                        <div class="command-item" onclick="window.RangeStudio.copyCommand('${escapeHtml(cmd.command)}')">
                            <div class="command-icon ${cmd.type}">${cmd.type === 'forge' ? '⚒' : '🔧'}</div>
                            <div class="command-text">
                                <div class="command-name">${escapeHtml(cmd.label)}</div>
                                <div class="command-desc" style="font-family: var(--font-mono);">${escapeHtml(cmd.command)}</div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>`;
    }

    // ---------- Render: Images ----------
    function renderImages() {
        const list = $('#images-list');
        list.innerHTML = state.images
            .map(
                (img) => `
            <div class="resource-card">
                <div class="resource-card-header">
                    <span class="resource-name">${escapeHtml(img.aliases[0] || img.fingerprint.slice(0, 12))}</span>
                    <span class="resource-badge ${img.type === 'virtual-machine' ? 'badge-vm' : 'badge-container'}">${img.type === 'virtual-machine' ? 'VM' : 'Container'}</span>
                </div>
                <div class="resource-desc">${escapeHtml(img.description)}</div>
                <div class="resource-meta">
                    <div class="resource-meta-item">Size: <span>${formatBytes(img.size)}</span></div>
                    <div class="resource-meta-item">Arch: <span>${escapeHtml(img.architecture)}</span></div>
                    <div class="resource-meta-item">Uploaded: <span>${formatDate(img.uploaded_at)}</span></div>
                    <div class="resource-meta-item">Fingerprint: <span>${escapeHtml(img.fingerprint.slice(0, 12))}…</span></div>
                </div>
            </div>`
            )
            .join('');
    }

    // ---------- Render: Profiles ----------
    function renderProfiles() {
        const list = $('#profiles-list');
        list.innerHTML = state.profiles
            .map(
                (prof) => `
            <div class="resource-card">
                <div class="resource-card-header">
                    <span class="resource-name">${escapeHtml(prof.name)}</span>
                </div>
                <div class="resource-desc">${escapeHtml(prof.description)}</div>
                <div class="resource-meta">
                    ${prof.config['limits.cpu'] ? `<div class="resource-meta-item">CPU: <span>${escapeHtml(prof.config['limits.cpu'])}</span></div>` : ''}
                    ${prof.config['limits.memory'] ? `<div class="resource-meta-item">Memory: <span>${escapeHtml(prof.config['limits.memory'])}</span></div>` : ''}
                    ${prof.devices.root ? `<div class="resource-meta-item">Disk: <span>${escapeHtml(prof.devices.root.size || 'default')}</span></div>` : ''}
                    ${prof.devices.root ? `<div class="resource-meta-item">Pool: <span>${escapeHtml(prof.devices.root.pool || 'default')}</span></div>` : ''}
                </div>
            </div>`
            )
            .join('');
    }

    // ---------- Render: Cluster ----------
    function renderCluster() {
        const list = $('#cluster-list');
        list.innerHTML = state.cluster
            .map(
                (m) => `
            <div class="resource-card">
                <div class="resource-card-header">
                    <span class="resource-name">${escapeHtml(m.server_name)}</span>
                    <div style="display: flex; gap: 6px;">
                        ${m.gpu ? '<span class="resource-badge badge-gpu">🎮 GPU</span>' : ''}
                        <span class="resource-badge badge-online">${escapeHtml(m.status)}</span>
                    </div>
                </div>
                <div class="resource-meta">
                    <div class="resource-meta-item">URL: <span>${escapeHtml(m.url)}</span></div>
                    <div class="resource-meta-item">Arch: <span>${escapeHtml(m.architecture)}</span></div>
                    ${m.roles.length > 0 ? `<div class="resource-meta-item">Roles: <span>${escapeHtml(m.roles.join(', '))}</span></div>` : ''}
                </div>
            </div>`
            )
            .join('');
    }

    // ---------- Render: Templates ----------
    function renderTemplates() {
        const list = $('#templates-list');
        if (state.templates.length === 0) {
            list.innerHTML = '<p style="color: var(--text-muted); padding: 20px;">No templates discovered.</p>';
            return;
        }
        list.innerHTML = state.templates
            .map(
                (t) => `
            <div class="resource-card">
                <div class="resource-card-header">
                    <span class="resource-name">${escapeHtml(t.id)}</span>
                    ${t.id === 'blank' ? '<span class="resource-badge badge-container">Synthetic</span>' : ''}
                </div>
                <div class="resource-desc">${t.description ? escapeHtml(t.description) : t.path ? `Source: ${escapeHtml(t.path)}` : 'Built-in template'}</div>
            </div>`
            )
            .join('');
    }

    // ---------- Commands ----------
    function getCommands(projectName) {
        const name = projectName || 'PROJECT';
        return [
            { label: 'Status', command: `forge status`, type: 'forge', desc: 'Show subnet allocation status' },
            { label: 'Plan', command: `forge plan`, type: 'forge', desc: 'Preview infrastructure changes' },
            { label: 'Apply', command: `forge apply`, type: 'forge', desc: 'Apply infrastructure changes' },
            { label: 'Destroy', command: `forge destroy`, type: 'forge', desc: 'Tear down infrastructure' },
            { label: 'Doctor', command: `forge doctor`, type: 'forge', desc: 'Run preflight checks' },
            { label: 'Validate', command: `tofu validate`, type: 'tofu', desc: 'Validate HCL configuration' },
            { label: 'Init', command: `tofu init`, type: 'tofu', desc: 'Initialize working directory' },
            { label: 'Subnets List', command: `forge subnets list`, type: 'forge', desc: 'List all subnet allocations' },
        ];
    }

    // ---------- Command Palette ----------
    function openCommandPalette(projectName) {
        const palette = $('#command-palette');
        const body = $('#palette-commands');
        const projectLabel = $('#palette-project-name');

        projectLabel.textContent = projectName || '';

        const cmds = getCommands(projectName);
        const forgeCommands = cmds.filter((c) => c.type === 'forge');
        const tofuCommands = cmds.filter((c) => c.type === 'tofu');

        body.innerHTML = `
            <div class="command-group-label">Forge</div>
            ${forgeCommands.map((cmd) => renderCommandItem(cmd)).join('')}
            <div class="command-group-label" style="margin-top: 8px;">OpenTofu</div>
            ${tofuCommands.map((cmd) => renderCommandItem(cmd)).join('')}
        `;

        palette.classList.remove('hidden');
    }

    function renderCommandItem(cmd) {
        return `
            <div class="command-item" onclick="window.RangeStudio.copyCommand('${escapeAttr(cmd.command)}')">
                <div class="command-icon ${cmd.type}">${cmd.type === 'forge' ? '⚒' : '🔧'}</div>
                <div class="command-text">
                    <div class="command-name">${escapeHtml(cmd.label)}</div>
                    <div class="command-desc">${escapeHtml(cmd.desc)}</div>
                </div>
                <button class="command-copy" onclick="event.stopPropagation(); window.RangeStudio.copyCommand('${escapeAttr(cmd.command)}')">
                    Copy
                </button>
            </div>`;
    }

    function closeCommandPalette() {
        $('#command-palette').classList.add('hidden');
    }

    // ---------- Context Menu ----------
    function showContextMenu(x, y, projectName) {
        const menu = $('#context-menu');
        const items = $('#context-menu-items');
        const cmds = getCommands(projectName);

        items.innerHTML = `
            <div class="context-item" onclick="window.RangeStudio.openDetail('${escapeAttr(projectName)}')">
                <span class="context-item-icon">📋</span>
                <span>View Details</span>
            </div>
            <div class="context-separator"></div>
            ${cmds.map((cmd) => `
                <div class="context-item" onclick="window.RangeStudio.copyCommand('${escapeAttr(cmd.command)}')">
                    <span class="context-item-icon">${cmd.type === 'forge' ? '⚒' : '🔧'}</span>
                    <span>${escapeHtml(cmd.label)}</span>
                </div>
            `).join('')}
            <div class="context-separator"></div>
            <div class="context-item" onclick="window.RangeStudio.openPalette('${escapeAttr(projectName)}')">
                <span class="context-item-icon">⌘</span>
                <span>Command Palette</span>
            </div>
        `;

        // Position the menu
        menu.style.left = Math.min(x, window.innerWidth - 240) + 'px';
        menu.style.top = Math.min(y, window.innerHeight - 300) + 'px';
        menu.classList.remove('hidden');
    }

    function hideContextMenu() {
        $('#context-menu').classList.add('hidden');
    }

    // ---------- Clipboard / Toast ----------
    async function copyCommand(command) {
        try {
            await navigator.clipboard.writeText(command);
            showToast(`Copied: ${command}`, 'success');
        } catch {
            // Fallback
            const ta = document.createElement('textarea');
            ta.value = command;
            document.body.appendChild(ta);
            ta.select();
            document.execCommand('copy');
            document.body.removeChild(ta);
            showToast(`Copied: ${command}`, 'success');
        }
        hideContextMenu();
        closeCommandPalette();
    }

    function showToast(message, type) {
        const toast = $('#toast');
        toast.className = 'toast ' + (type || '');
        toast.textContent = message;
        toast.classList.remove('hidden');
        clearTimeout(toast._timeout);
        toast._timeout = setTimeout(() => toast.classList.add('hidden'), 2500);
    }

    // ---------- Utilities ----------
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
            const d = new Date(iso);
            return d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
        } catch {
            return iso;
        }
    }

    function formatBytes(bytes) {
        if (!bytes || bytes === 0) return '0 B';
        const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
    }

    function truncatePath(path) {
        if (!path) return '';
        if (path.length <= 45) return path;
        return '…' + path.slice(-42);
    }

    // ---------- Event Listeners ----------
    // Sidebar navigation
    $$('.nav-item').forEach((btn) => {
        btn.addEventListener('click', () => showView(btn.dataset.view));
    });

    // Back button
    $('#btn-back').addEventListener('click', () => showView('projects'));

    // Refresh
    $('#btn-refresh').addEventListener('click', async () => {
        await loadProjects();
        renderProjects();
        showToast('Projects refreshed', 'success');
    });

    // Command palette buttons
    $('#btn-command-palette').addEventListener('click', () => {
        const name = state.selectedProject ? state.selectedProject.name : '';
        openCommandPalette(name);
    });
    $('#btn-detail-commands').addEventListener('click', () => {
        const name = state.selectedProject ? state.selectedProject.name : '';
        openCommandPalette(name);
    });

    // Palette close
    $('#btn-palette-close').addEventListener('click', closeCommandPalette);
    $('.command-palette-backdrop').addEventListener('click', closeCommandPalette);

    // Hide context menu on click elsewhere
    document.addEventListener('click', hideContextMenu);

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        // Ctrl+K → command palette
        if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
            e.preventDefault();
            const name = state.selectedProject ? state.selectedProject.name : '';
            openCommandPalette(name);
        }
        // Escape → close overlays
        if (e.key === 'Escape') {
            closeCommandPalette();
            hideContextMenu();
        }
    });

    // ---------- Public API (for inline onclick handlers) ----------
    window.RangeStudio = {
        copyCommand,
        openDetail(projectName) {
            hideContextMenu();
            state.selectedProject = state.projects.find((p) => p.name === projectName);
            showView('project-detail');
        },
        openPalette(projectName) {
            hideContextMenu();
            openCommandPalette(projectName);
        },
    };

    // ---------- Boot ----------
    init();
})();

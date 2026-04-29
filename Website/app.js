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

    // Wizard state - mirrors CreateProjectRequest on the server.
    // topology fields use snake_case to match the Go JSON tags.
    const wizState = {
        name: '',
        group: 'Education',
        subGroup: 'DCIG',
        teamCount: 1,
        needsGPU: false,
        placementMode: 'default',
        pinnedTarget: 'micro-01',
        template: '',
        topology: {
            network_mode: 'multi',
            networks: [],
            project_firewall: { image: 'openwrt-project', profile: 'pfsense', wan_network: 'CLASS_WAN', interfaces: [] },
            team_firewall:    { image: 'openwrt-team-new', profile: 'pfsense', wan_network: '', interfaces: [] },
            vms: [],
            placement_mode: 'default',
            pinned_target: '',
            include_salt: true,
            include_guac: true,
            include_project_fw: true,
        },
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
            case 'wizard':
                $('#view-wizard').classList.remove('hidden');
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

    // ---------- Wizard ----------
    function openWizard() {
        showView('wizard');
        showWizStep(1);
    }
    function closeWizard() {
        showView('projects');
    }

    // wizardLabel returns the right step subtitle. Custom builds get a
    // 4-step flow; non-custom flows skip the topology step entirely.
    function wizardLabel(step, isCustom) {
        const steps = isCustom
            ? ['Info', 'Template', 'Topology', 'Review']
            : ['Info', 'Template', '', 'Review'];
        return steps[step - 1] || '';
    }

    function showWizStep(step) {
        const isCustom = wizState.template === 'custom';
        // Skip step 3 entirely for non-custom templates.
        if (step === 3 && !isCustom) step = 4;

        $$('.wizard-step').forEach((s) => s.classList.add('hidden'));
        $(`#wizard-step-${step}`).classList.remove('hidden');
        $('#wizard-subtitle').textContent =
            `Step ${step} of 4: ${wizardLabel(step, isCustom)}`;
        $('#btn-wizard-create').classList.toggle('hidden', step !== 4);

        if (step === 2) renderWizTemplates();
        if (step === 3) renderWizTopology();
        if (step === 4) loadWizPreview();
    }

    // ---------- Step 1 ----------
    $('#wiz-group').addEventListener('change', (e) => {
        const isCIG = e.target.value === 'CIG';
        const isResearch = e.target.value === 'Research';
        $('#wiz-subgroup-group').classList.toggle('hidden', !isCIG);
        $('#wiz-research-block').classList.toggle('hidden', !isResearch);
    });
    $('#wiz-placement').addEventListener('change', (e) => {
        $('#wiz-pinned-target-group').classList.toggle('hidden', e.target.value !== 'pinned');
    });

    $('#btn-wiz-next-1').addEventListener('click', () => {
        const name = $('#wiz-name').value.trim();
        if (!name) return showToast('Project name is required', 'error');
        if (!/^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$/.test(name)) {
            return showToast('Name must be 2-41 chars: letters, digits, dash, underscore', 'error');
        }
        wizState.name = name;
        wizState.group = $('#wiz-group').value;
        wizState.subGroup = $('#wiz-subgroup').value;
        wizState.teamCount = Math.max(1, parseInt($('#wiz-teams').value, 10) || 1);

        if (wizState.group === 'Research') {
            const gpu = document.querySelector('input[name="wiz-gpu"]:checked');
            wizState.needsGPU = gpu && gpu.value === 'true';
            wizState.placementMode = $('#wiz-placement').value;
            wizState.pinnedTarget = $('#wiz-pinned-target').value;
        } else {
            wizState.needsGPU = false;
            wizState.placementMode = 'default';
            wizState.pinnedTarget = '';
        }
        wizState.topology.placement_mode = wizState.placementMode;
        wizState.topology.pinned_target = wizState.pinnedTarget;
        showWizStep(2);
    });

    // ---------- Step 2 ----------
    async function renderWizTemplates() {
        await loadTemplates();
        const grid = $('#wiz-template-grid');
        const customSel = wizState.template === 'custom' ? 'selected' : '';
        grid.innerHTML = `
            <div class="template-card custom-build ${customSel}" data-id="custom">
                <div class="template-title">Build Custom Topology</div>
                <div class="template-desc">Design networks, firewalls, and VMs via the wizard. Generates a fresh main.tf.</div>
            </div>
        ` + state.templates.map((t) => {
            const sel = wizState.template === t.id ? 'selected' : '';
            const desc = t.description || (t.path ? `Source: ${t.path}` : 'Built-in template');
            return `
            <div class="template-card ${sel}" data-id="${escapeAttr(t.id)}">
                <div class="template-title">${escapeHtml(t.id)}</div>
                <div class="template-desc">${escapeHtml(desc)}</div>
            </div>`;
        }).join('');

        grid.querySelectorAll('.template-card').forEach((card) => {
            card.addEventListener('click', () => {
                grid.querySelectorAll('.template-card').forEach((c) => c.classList.remove('selected'));
                card.classList.add('selected');
                wizState.template = card.dataset.id;
                $('#btn-wiz-next-2').disabled = false;
            });
        });

        $('#btn-wiz-next-2').disabled = !wizState.template;
    }

    $('#btn-wiz-prev-2').addEventListener('click', () => showWizStep(1));
    $('#btn-wiz-next-2').addEventListener('click', () => {
        if (wizState.template === 'custom') showWizStep(3);
        else showWizStep(4);
    });

    // ---------- Step 3 (custom topology) ----------
    $('#btn-wiz-prev-3').addEventListener('click', () => showWizStep(2));
    $('#btn-wiz-next-3').addEventListener('click', () => {
        if (wizState.topology.networks.length === 0) {
            return showToast('Add at least one network before continuing', 'error');
        }
        if (wizState.topology.networks.some((n) => !n.name)) {
            return showToast('Every network needs a name', 'error');
        }
        showWizStep(4);
    });

    function renderWizTopology() {
        document.querySelectorAll('input[name="wiz-net-mode"]').forEach((r) => {
            r.checked = r.value === wizState.topology.network_mode;
        });
        renderWizNetworks();
        renderWizFirewall('team_firewall', '#topo-fw-interfaces');
        renderWizVMs();
    }

    function renderWizNetworks() {
        const list = $('#topo-networks');
        list.innerHTML = wizState.topology.networks.map((n, i) => `
            <div class="topo-item">
                <input type="text" placeholder="Network Name (e.g. lan, dmz)" value="${escapeAttr(n.name)}"
                       onchange="window.RangeStudio.updateNet(${i}, this.value)">
                <button class="btn btn-sm btn-danger"
                        onclick="window.RangeStudio.delNet(${i})">Remove</button>
            </div>
        `).join('');
    }

    function renderWizFirewall(key, sel) {
        const list = $(sel);
        const fw = wizState.topology[key];
        list.innerHTML = fw.interfaces.map((intf, i) => `
            <div class="topo-item">
                <span class="eth-label">eth${i + (key === 'project_firewall' ? 3 : 1)}</span>
                <select onchange="window.RangeStudio.updateFwIntf('${key}', ${i}, 'network', this.value)">
                    <option value="">Select Network...</option>
                    ${wizState.topology.networks.map((n) => `
                        <option value="${escapeAttr(n.name)}" ${intf.network === n.name ? 'selected' : ''}>${escapeHtml(n.name)}</option>
                    `).join('')}
                </select>
                <input type="text" placeholder="IP (e.g. 192.168.1.1/24)"
                       value="${escapeAttr(intf.ip)}"
                       onchange="window.RangeStudio.updateFwIntf('${key}', ${i}, 'ip', this.value)">
                <button class="btn btn-sm btn-danger"
                        onclick="window.RangeStudio.delFwIntf('${key}', ${i})">Remove</button>
            </div>
        `).join('');
    }

    function renderWizVMs() {
        const list = $('#topo-vms');
        list.innerHTML = wizState.topology.vms.map((vm, i) => {
            const showGPU = wizState.group === 'Research';
            const targets = ['default', 'round_robin', 'micro-01', 'micro-02', 'micro-03', 'micro-04', 'micro-05', 'micro-06', 'micro-08', 'micro-09'];
            const ci = vm.cloud_init || { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
            const netIps = vm.net_ips || [];
            return `
            <div class="vm-card">
                <div class="vm-card-header">
                    <input type="text" class="vm-name" placeholder="VM Name (e.g. Jumpbox)"
                           value="${escapeAttr(vm.name)}"
                           onchange="window.RangeStudio.updateVm(${i}, 'name', this.value)">
                    <button class="btn btn-sm btn-danger"
                            onclick="window.RangeStudio.delVm(${i})">Remove VM</button>
                </div>
                <div class="vm-grid">
                    <div>
                        <label>Image</label>
                        <select onchange="window.RangeStudio.updateVm(${i}, 'image', this.value)">
                            <option value="">Select Image...</option>
                            ${state.images.map((img) => {
                                const alias = img.aliases[0] || img.fingerprint;
                                return `<option value="${escapeAttr(alias)}" ${vm.image === alias ? 'selected' : ''}>${escapeHtml(alias)} (${img.type})</option>`;
                            }).join('')}
                        </select>
                    </div>
                    <div>
                        <label>Profile</label>
                        <select onchange="window.RangeStudio.updateVm(${i}, 'profile', this.value)">
                            <option value="">Select Profile...</option>
                            ${state.profiles.map((p) => `
                                <option value="${escapeAttr(p.name)}" ${vm.profile === p.name ? 'selected' : ''}>${escapeHtml(p.name)}</option>
                            `).join('')}
                        </select>
                    </div>
                    <div>
                        <label>Type</label>
                        <select onchange="window.RangeStudio.updateVm(${i}, 'type', this.value)">
                            <option value="virtual-machine" ${vm.type !== 'container' ? 'selected' : ''}>virtual-machine</option>
                            <option value="container" ${vm.type === 'container' ? 'selected' : ''}>container</option>
                        </select>
                    </div>
                    <div>
                        <label>CPU (cores)</label>
                        <input type="text" placeholder="2" value="${escapeAttr(vm.cpu)}"
                               onchange="window.RangeStudio.updateVm(${i}, 'cpu', this.value)">
                    </div>
                    <div>
                        <label>Memory</label>
                        <input type="text" placeholder="4GiB" value="${escapeAttr(vm.memory)}"
                               onchange="window.RangeStudio.updateVm(${i}, 'memory', this.value)">
                    </div>
                    <div>
                        <label>Disk</label>
                        <input type="text" placeholder="50GiB" value="${escapeAttr(vm.disk_size)}"
                               onchange="window.RangeStudio.updateVm(${i}, 'disk_size', this.value)">
                    </div>
                    <div class="vm-grid-wide">
                        <label>Networks &amp; IPs</label>
                        <div class="vm-network-ips">
                            <div class="vm-net-ip-row">
                                <label class="check-pill">
                                    <input type="checkbox" ${vm.networks.includes('guac_wan') ? 'checked' : ''}
                                           onchange="window.RangeStudio.toggleVmNet(${i}, 'guac_wan', this.checked)">
                                    <span>guac_wan</span>
                                </label>
                                <span class="text-muted text-sm" style="flex:1;">Remote access (DHCP - no IP needed)</span>
                            </div>
                            ${wizState.topology.networks.map((n, ni) => {
                                const netIp = netIps.find(nip => nip.network === n.name) || { network: n.name, ip: '' };
                                const isConnected = vm.networks.includes(n.name);
                                return `
                                <div class="vm-net-ip-row">
                                    <label class="check-pill">
                                        <input type="checkbox" ${isConnected ? 'checked' : ''}
                                               onchange="window.RangeStudio.toggleVmNet(${i}, '${escapeAttr(n.name)}', this.checked)">
                                        <span>${escapeHtml(n.name)}</span>
                                    </label>
                                    <input type="text" placeholder="IP (e.g. 10.10.0.50/24)" 
                                           value="${escapeAttr(netIp.ip)}"
                                           ${!isConnected ? 'disabled' : ''}
                                           onchange="window.RangeStudio.updateVmNetIp(${i}, '${escapeAttr(n.name)}', this.value)">
                                </div>`;
                            }).join('')}
                        </div>
                    </div>
                    <div>
                        <label>Placement</label>
                        <select onchange="window.RangeStudio.updateVm(${i}, 'target', this.value)">
                            ${targets.map((t) => `
                                <option value="${t === 'default' ? '' : t}" ${(vm.target || '') === (t === 'default' ? '' : t) ? 'selected' : ''}>${t === 'default' ? 'Cluster default' : t === 'round_robin' ? 'Round-robin (excl. 07)' : t}</option>
                            `).join('')}
                        </select>
                    </div>
                    ${showGPU ? `
                    <div>
                        <label>GPU</label>
                        <label class="check-pill" style="margin-top: 4px;">
                            <input type="checkbox" ${vm.needs_gpu ? 'checked' : ''}
                                   onchange="window.RangeStudio.updateVm(${i}, 'needs_gpu', this.checked)">
                            <span>Pin to micro-07-gpu</span>
                        </label>
                    </div>` : ''}
                    <div>
                        <label>Start on create</label>
                        <label class="check-pill" style="margin-top: 4px;">
                            <input type="checkbox" ${vm.running !== false ? 'checked' : ''}
                                   onchange="window.RangeStudio.updateVm(${i}, 'running', this.checked)">
                            <span>Power on (uncheck for Windows)</span>
                        </label>
                    </div>
                </div>
                <details class="vm-advanced">
                    <summary>Security &amp; Cloud-Init Settings</summary>
                    <div class="vm-grid" style="margin-top: 12px;">
                        <div>
                            <label>Secure Boot</label>
                            <label class="check-pill" style="margin-top: 4px;">
                                <input type="checkbox" ${vm.secure_boot ? 'checked' : ''}
                                       onchange="window.RangeStudio.updateVm(${i}, 'secure_boot', this.checked)">
                                <span>Enable (usually off)</span>
                            </label>
                        </div>
                        <div>
                            <label>CSM / Legacy Boot</label>
                            <label class="check-pill" style="margin-top: 4px;">
                                <input type="checkbox" ${vm.csm ? 'checked' : ''}
                                       onchange="window.RangeStudio.updateVm(${i}, 'csm', this.checked)">
                                <span>Enable CSM</span>
                            </label>
                        </div>
                        <div>
                            <label>Swap Size</label>
                            <input type="text" placeholder="e.g. 4G" value="${escapeAttr(ci.swap_size || '')}"
                                   onchange="window.RangeStudio.updateVmCloudInit(${i}, 'swap_size', this.value)">
                        </div>
                        <div>
                            <label>SSH Password Auth</label>
                            <label class="check-pill" style="margin-top: 4px;">
                                <input type="checkbox" ${ci.ssh_pwauth ? 'checked' : ''}
                                       onchange="window.RangeStudio.updateVmCloudInit(${i}, 'ssh_pwauth', this.checked)">
                                <span>Enable ssh_pwauth</span>
                            </label>
                        </div>
                        <div>
                            <label>Package Update</label>
                            <label class="check-pill" style="margin-top: 4px;">
                                <input type="checkbox" ${ci.package_update ? 'checked' : ''}
                                       onchange="window.RangeStudio.updateVmCloudInit(${i}, 'package_update', this.checked)">
                                <span>Update on boot</span>
                            </label>
                        </div>
                        <div class="vm-grid-wide">
                            <label>Run Commands (one per line)</label>
                            <textarea rows="3" placeholder="echo hello&#10;ufw --force reset"
                                      onchange="window.RangeStudio.updateVmCloudInit(${i}, 'runcmds', this.value)">${escapeHtml((ci.runcmds || []).join('\n'))}</textarea>
                        </div>
                    </div>
                </details>
            </div>`;
        }).join('');
    }

    // Step 3 buttons
    document.querySelectorAll('input[name="wiz-net-mode"]').forEach((r) => {
        r.addEventListener('change', (e) => {
            wizState.topology.network_mode = e.target.value;
        });
    });

    $('#btn-add-network').addEventListener('click', () => {
        wizState.topology.networks.push({ name: '' });
        renderWizTopology();
    });
    $('#btn-add-fw-interface').addEventListener('click', () => {
        const fw = wizState.topology.team_firewall;
        const idx = fw.interfaces.length;
        fw.interfaces.push({ eth_name: `eth${idx + 1}`, network: '', ip: '' });
        renderWizFirewall('team_firewall', '#topo-fw-interfaces');
    });
    $('#btn-add-vm').addEventListener('click', async () => {
        await loadImages();
        await loadProfiles();
        wizState.topology.vms.push({
            name: '', image: '', profile: 'default', type: 'virtual-machine',
            networks: [], net_ips: [], cpu: '', memory: '', disk_size: '',
            needs_gpu: false, target: '', running: true,
            secure_boot: false, csm: false,
            cloud_init: { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] },
        });
        renderWizTopology();
    });

    // ---------- Step 4 ----------
    $('#btn-wiz-prev-4').addEventListener('click', () => {
        if (wizState.template === 'custom') showWizStep(3);
        else showWizStep(2);
    });

    function buildPayload() {
        return {
            name: wizState.name,
            group: wizState.group,
            sub_group: wizState.subGroup,
            team_count: wizState.teamCount,
            template: wizState.template,
            needs_gpu: wizState.needsGPU,
            topology: wizState.template === 'custom' ? wizState.topology : null,
        };
    }

    async function loadWizPreview() {
        const grid = $('#wiz-review-summary');
        const summaryRows = [
            ['Name', wizState.name],
            ['Group', wizState.group + (wizState.group === 'CIG' ? ` (${wizState.subGroup})` : '')],
            ['Team Count', wizState.teamCount],
            ['Template', wizState.template],
        ];
        if (wizState.group === 'Research') {
            summaryRows.push(['GPU', wizState.needsGPU ? 'Yes (micro-07-gpu)' : 'No']);
            summaryRows.push(['Placement', wizState.placementMode + (wizState.placementMode === 'pinned' ? ` (${wizState.pinnedTarget})` : '')]);
        }
        if (wizState.template === 'custom') {
            summaryRows.push(['Networks', wizState.topology.networks.map((n) => n.name).join(', ') || '—']);
            summaryRows.push(['VMs', wizState.topology.vms.length]);
            summaryRows.push(['Mode', wizState.topology.network_mode]);
        }
        grid.innerHTML = summaryRows.map(([k, v]) =>
            `<div class="review-key">${escapeHtml(k)}</div><div class="review-val">${escapeHtml(String(v))}</div>`
        ).join('');

        const code = $('#wiz-preview-hcl');
        code.textContent = 'Generating preview...';

        try {
            const res = await fetch(API_BASE + '/api/projects/preview', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(buildPayload()),
            });
            const data = await res.json();
            code.textContent = data.error ? 'Error: ' + data.error : data.hcl;
        } catch (e) {
            code.textContent = 'Failed to load preview: ' + e.message;
        }
    }

    $('#btn-wizard-create').addEventListener('click', async () => {
        const btn = $('#btn-wizard-create');
        const original = btn.innerHTML;
        btn.disabled = true;
        btn.textContent = 'Creating...';
        try {
            const res = await fetch(API_BASE + '/api/projects/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(buildPayload()),
            });
            const data = await res.json();
            if (data.error) throw new Error(data.error);
            showToast(`Created ${wizState.name} at ${data.output_dir}`, 'success');
            await loadProjects();
            closeWizard();
            showView('projects');
        } catch (e) {
            showToast(e.message, 'error');
        } finally {
            btn.disabled = false;
            btn.innerHTML = original;
        }
    });

    $('#btn-new-project').addEventListener('click', openWizard);
    $('#btn-wizard-cancel').addEventListener('click', closeWizard);

    // ---------- Public API (inline onclick handlers) ----------
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
        // Networks
        updateNet(i, val) {
            wizState.topology.networks[i].name = val.trim();
        },
        delNet(i) {
            wizState.topology.networks.splice(i, 1);
            renderWizTopology();
        },
        // Firewall interfaces
        updateFwIntf(key, i, field, val) {
            const fw = wizState.topology[key];
            fw.interfaces[i][field] = val;
            // Keep eth_name in sync with index. Project FW reserves
            // eth0..eth2 for upstream/team_wan/salt_lan; user-added
            // interfaces start at eth3.
            const offset = key === 'project_firewall' ? 3 : 1;
            fw.interfaces[i].eth_name = `eth${i + offset}`;
        },
        delFwIntf(key, i) {
            wizState.topology[key].interfaces.splice(i, 1);
            renderWizFirewall(key, key === 'project_firewall' ? '#topo-projfw-interfaces' : '#topo-fw-interfaces');
        },
        // VMs
        updateVm(i, field, val) {
            wizState.topology.vms[i][field] = val;
            // Auto-uncheck "running" when a Windows image is selected
            if (field === 'image' && /win|windows/i.test(val)) {
                wizState.topology.vms[i].running = false;
                renderWizVMs(); // Re-render to reflect the change
            }
        },
        delVm(i) {
            wizState.topology.vms.splice(i, 1);
            renderWizVMs();
        },
        toggleVmNet(i, net, checked) {
            const vm = wizState.topology.vms[i];
            if (checked && !vm.networks.includes(net)) vm.networks.push(net);
            if (!checked) vm.networks = vm.networks.filter((n) => n !== net);
            // Also remove net_ips entry if unchecked
            if (!checked && vm.net_ips) {
                vm.net_ips = vm.net_ips.filter(nip => nip.network !== net);
            }
            renderWizVMs(); // Re-render to enable/disable IP input
        },
        updateVmNetIp(i, net, ip) {
            const vm = wizState.topology.vms[i];
            if (!vm.net_ips) vm.net_ips = [];
            const existing = vm.net_ips.find(nip => nip.network === net);
            if (existing) {
                existing.ip = ip;
            } else {
                vm.net_ips.push({ network: net, ip: ip });
            }
        },
        updateVmCloudInit(i, field, val) {
            const vm = wizState.topology.vms[i];
            if (!vm.cloud_init) vm.cloud_init = { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
            if (field === 'runcmds') {
                vm.cloud_init.runcmds = val.split('\n').filter(s => s.trim() !== '');
            } else {
                vm.cloud_init[field] = val;
            }
        },
    };

    // ---------- Boot ----------
    init();
})();

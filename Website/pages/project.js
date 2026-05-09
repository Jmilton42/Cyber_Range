/* Project detail page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    const params = new URLSearchParams(window.location.search);
    const name = params.get('name');
    if (!name) {
        window.location.href = 'index.html';
        return;
    }
    RS._currentProject = name;

    document.getElementById('detail-project-name').textContent = name;
    document.getElementById('btn-detail-commands').addEventListener('click', () => RS.openCommandPalette(name));

    async function load() {
        let detail;
        try {
            detail = await RS.api(`/api/projects/${encodeURIComponent(name)}`);
        } catch (err) {
            document.getElementById('detail-content').innerHTML =
                `<div class="detail-card full-width" style="color: var(--danger);">
                    <p style="font-weight:600;margin-bottom:6px;">Project not found</p>
                    <p style="font-size:0.85rem;color:var(--fg-muted);font-family:var(--font-mono);">${RS.escapeHtml(err.message)}</p>
                </div>`;
            return;
        }

        document.getElementById('detail-subtitle').textContent =
            `${detail.subnet} • Octet ${detail.subnet_octet}`;

        const cmds = RS.getCommands(detail.name);
        const content = document.getElementById('detail-content');
        content.innerHTML = `
            <div class="detail-card">
                <div class="detail-card-title">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/><path d="M8 5v3M8 10v1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>
                    Project Info
                </div>
                <div class="detail-info-grid">
                    <span class="detail-info-label">Name</span>
                    <span class="detail-info-value">${RS.escapeHtml(detail.name)}</span>
                    <span class="detail-info-label">Subnet</span>
                    <span class="detail-info-value" style="color: var(--accent);">${RS.escapeHtml(detail.subnet)}</span>
                    <span class="detail-info-label">Gateway</span>
                    <span class="detail-info-value">${RS.escapeHtml(detail.gateway)}</span>
                    <span class="detail-info-label">Octet</span>
                    <span class="detail-info-value">${detail.subnet_octet}</span>
                    <span class="detail-info-label">Allocated</span>
                    <span class="detail-info-value">${RS.formatDate(detail.allocated_at)}</span>
                    <span class="detail-info-label">Status</span>
                    <span class="detail-info-value"><span class="project-status status-${detail.status}" style="font-size: 0.7rem;">${detail.status}</span></span>
                    <span class="detail-info-label">Work Dir</span>
                    <span class="detail-info-value">${detail.work_dir ? RS.escapeHtml(detail.work_dir) : '<span style="color:var(--fg-faint)">not resolved</span>'}</span>
                    <span class="detail-info-label">main.tf</span>
                    <span class="detail-info-value">${detail.has_main_tf ? '<span style="color:var(--success)">✓ present</span>' : '<span style="color:var(--fg-faint)">—</span>'}</span>
                    <span class="detail-info-label">TF State</span>
                    <span class="detail-info-value">${detail.has_state ? '<span style="color:var(--success)">✓ exists</span>' : '<span style="color:var(--fg-faint)">none</span>'}</span>
                </div>
            </div>

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
                            <span>${RS.escapeHtml(f.name)}</span>
                            ${!f.is_dir ? `<span class="file-size">${RS.formatBytes(f.size)}</span>` : ''}
                        </div>`).join('')}
                </div>` : `<p style="color: var(--fg-muted); font-size: 0.84rem;">No directory resolved for this project.</p>`}
            </div>

            <div class="detail-card full-width">
                <div class="detail-card-title">
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M5 3l6 5-6 5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
                    Quick Commands
                </div>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 8px;">
                    ${cmds.map((c) => `
                        <div class="command-item" data-cmd="${RS.escapeAttr(c.command)}">
                            <div class="command-icon ${c.type}">${c.type === 'forge' ? '⚒' : '🔧'}</div>
                            <div class="command-text">
                                <div class="command-name">${RS.escapeHtml(c.label)}</div>
                                <div class="command-desc" style="font-family: var(--font-mono);">${RS.escapeHtml(c.command)}</div>
                            </div>
                        </div>`).join('')}
                </div>
            </div>`;

        content.querySelectorAll('[data-cmd]').forEach((el) => {
            el.addEventListener('click', () => RS.copyCommand(el.dataset.cmd));
        });
    }

    load();
})();

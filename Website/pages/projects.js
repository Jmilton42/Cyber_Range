/* Projects list page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    let projects = [];

    async function load() {
        try {
            projects = await RS.api('/api/projects');
            render();
        } catch (err) {
            document.getElementById('projects-grid').innerHTML =
                `<div style="grid-column: 1 / -1; padding: 40px; text-align: center; color: var(--danger);">
                    <p style="font-weight:600;margin-bottom:6px;">Could not load projects</p>
                    <p style="font-size:0.85rem;color:var(--fg-muted);font-family:var(--font-mono);">${RS.escapeHtml(err.message)}</p>
                </div>`;
        }
    }

    function render() {
        const grid = document.getElementById('projects-grid');
        if (projects.length === 0) {
            grid.innerHTML = `
                <div class="empty-state-card">
                    <div class="empty-icon">
                        <svg width="26" height="26" viewBox="0 0 26 26" fill="none">
                            <rect x="3" y="3" width="9" height="9" rx="2" stroke="currentColor" stroke-width="1.6"/>
                            <rect x="14" y="3" width="9" height="9" rx="2" stroke="currentColor" stroke-width="1.6"/>
                            <rect x="3" y="14" width="9" height="9" rx="2" stroke="currentColor" stroke-width="1.6"/>
                            <path d="M18.5 14.5v8M14.5 18.5h8" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
                        </svg>
                    </div>
                    <h3>No projects yet</h3>
                    <p>Create your first range project to allocate a subnet and define a topology.</p>
                    <a class="btn btn-primary" href="new-project.html">
                        <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M7 2v10M2 7h10" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
                        Create your first project
                    </a>
                </div>`;
            return;
        }

        grid.innerHTML = projects.map((p) => `
            <a class="project-card" href="project.html?name=${encodeURIComponent(p.name)}" data-project="${RS.escapeAttr(p.name)}">
                <div class="project-card-header">
                    <span class="project-name">${RS.escapeHtml(p.name)}</span>
                    <span class="project-status status-${p.status}">
                        <span class="status-dot"></span>
                        ${p.status === 'missing_dir' ? 'No Dir' : p.status}
                    </span>
                </div>
                <div class="project-meta">
                    <div class="project-meta-row"><span class="meta-label">Subnet</span><span class="meta-value subnet">${RS.escapeHtml(p.subnet)}</span></div>
                    <div class="project-meta-row"><span class="meta-label">Gateway</span><span class="meta-value">${RS.escapeHtml(p.gateway)}</span></div>
                    <div class="project-meta-row"><span class="meta-label">Octet</span><span class="meta-value">${p.subnet_octet}</span></div>
                    <div class="project-meta-row"><span class="meta-label">Allocated</span><span class="meta-value">${RS.formatDate(p.allocated_at)}</span></div>
                    ${p.work_dir ? `<div class="project-meta-row"><span class="meta-label">Path</span><span class="meta-value path" title="${RS.escapeHtml(p.work_dir)}">${RS.truncatePath(p.work_dir)}</span></div>` : ''}
                    ${p.has_main_tf ? '<div class="project-meta-row"><span class="meta-label"></span><span class="badge-main-tf">main.tf ✓</span></div>' : ''}
                </div>
            </a>`).join('');

        grid.querySelectorAll('.project-card').forEach((card) => {
            card.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                RS.showContextMenu(e.clientX, e.clientY, card.dataset.project);
            });
        });
    }

    document.getElementById('btn-refresh').addEventListener('click', async () => {
        await load();
        RS.showToast('Projects refreshed', 'success');
    });
    document.getElementById('btn-command-palette').addEventListener('click', () => {
        RS.openCommandPalette('');
    });

    load();
})();

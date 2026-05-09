/* Cluster page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    async function load() {
        const list = document.getElementById('resource-list');
        try {
            const cluster = await RS.api('/api/lxd/cluster');
            list.innerHTML = cluster.map((m) => `
                <div class="resource-card">
                    <div class="resource-card-header">
                        <span class="resource-name">${RS.escapeHtml(m.server_name)}</span>
                        <div style="display: flex; gap: 6px;">
                            ${m.gpu ? '<span class="resource-badge badge-gpu">GPU</span>' : ''}
                            <span class="resource-badge badge-online">${RS.escapeHtml(m.status)}</span>
                        </div>
                    </div>
                    <div class="resource-meta">
                        <div class="resource-meta-item">URL: <span>${RS.escapeHtml(m.url)}</span></div>
                        <div class="resource-meta-item">Arch: <span>${RS.escapeHtml(m.architecture)}</span></div>
                        ${m.roles.length > 0 ? `<div class="resource-meta-item">Roles: <span>${RS.escapeHtml(m.roles.join(', '))}</span></div>` : ''}
                    </div>
                </div>`).join('') || '<p style="color:var(--fg-muted);padding:20px;">No cluster members found.</p>';
        } catch (err) {
            list.innerHTML = `<p style="color: var(--danger); padding: 20px;">Failed to load: ${RS.escapeHtml(err.message)}</p>`;
        }
    }
    load();
})();

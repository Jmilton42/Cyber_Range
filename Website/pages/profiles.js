/* Profiles page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    async function load() {
        const list = document.getElementById('resource-list');
        try {
            const profiles = await RS.api('/api/lxd/profiles');
            list.innerHTML = profiles.map((prof) => `
                <div class="resource-card">
                    <div class="resource-card-header">
                        <span class="resource-name">${RS.escapeHtml(prof.name)}</span>
                    </div>
                    <div class="resource-desc">${RS.escapeHtml(prof.description)}</div>
                    <div class="resource-meta">
                        ${prof.config['limits.cpu'] ? `<div class="resource-meta-item">CPU: <span>${RS.escapeHtml(prof.config['limits.cpu'])}</span></div>` : ''}
                        ${prof.config['limits.memory'] ? `<div class="resource-meta-item">Memory: <span>${RS.escapeHtml(prof.config['limits.memory'])}</span></div>` : ''}
                        ${prof.devices.root ? `<div class="resource-meta-item">Disk: <span>${RS.escapeHtml(prof.devices.root.size || 'default')}</span></div>` : ''}
                        ${prof.devices.root ? `<div class="resource-meta-item">Pool: <span>${RS.escapeHtml(prof.devices.root.pool || 'default')}</span></div>` : ''}
                    </div>
                </div>`).join('') || '<p style="color:var(--fg-muted);padding:20px;">No profiles found.</p>';
        } catch (err) {
            list.innerHTML = `<p style="color: var(--danger); padding: 20px;">Failed to load: ${RS.escapeHtml(err.message)}</p>`;
        }
    }
    load();
})();

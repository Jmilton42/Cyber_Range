/* Templates page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    async function load() {
        const list = document.getElementById('resource-list');
        try {
            const templates = await RS.api('/api/templates');
            if (templates.length === 0) {
                list.innerHTML = '<p style="color:var(--fg-muted);padding:20px;">No templates discovered.</p>';
                return;
            }
            list.innerHTML = templates.map((t) => `
                <div class="resource-card">
                    <div class="resource-card-header">
                        <span class="resource-name">${RS.escapeHtml(t.id)}</span>
                        ${t.id === 'blank' ? '<span class="resource-badge badge-container">Synthetic</span>' : ''}
                    </div>
                    <div class="resource-desc">${t.description ? RS.escapeHtml(t.description) : t.path ? `Source: ${RS.escapeHtml(t.path)}` : 'Built-in template'}</div>
                </div>`).join('');
        } catch (err) {
            list.innerHTML = `<p style="color: var(--danger); padding: 20px;">Failed to load: ${RS.escapeHtml(err.message)}</p>`;
        }
    }
    load();
})();

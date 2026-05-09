/* Images page */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    async function load() {
        const list = document.getElementById('resource-list');
        try {
            const images = await RS.api('/api/lxd/images');
            list.innerHTML = images.map((img) => `
                <div class="resource-card">
                    <div class="resource-card-header">
                        <span class="resource-name">${RS.escapeHtml(img.aliases[0] || img.fingerprint.slice(0, 12))}</span>
                        <span class="resource-badge ${img.type === 'virtual-machine' ? 'badge-vm' : 'badge-container'}">${img.type === 'virtual-machine' ? 'VM' : 'Container'}</span>
                    </div>
                    <div class="resource-desc">${RS.escapeHtml(img.description)}</div>
                    <div class="resource-meta">
                        <div class="resource-meta-item">Size: <span>${RS.formatBytes(img.size)}</span></div>
                        <div class="resource-meta-item">Arch: <span>${RS.escapeHtml(img.architecture)}</span></div>
                        <div class="resource-meta-item">Uploaded: <span>${RS.formatDate(img.uploaded_at)}</span></div>
                        <div class="resource-meta-item">Fingerprint: <span>${RS.escapeHtml(img.fingerprint.slice(0, 12))}…</span></div>
                    </div>
                </div>`).join('') || '<p style="color:var(--fg-muted);padding:20px;">No images found.</p>';
        } catch (err) {
            list.innerHTML = `<p style="color: var(--danger); padding: 20px;">Failed to load: ${RS.escapeHtml(err.message)}</p>`;
        }
    }
    load();
})();

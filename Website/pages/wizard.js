/* New Project Wizard */
(function () {
    'use strict';
    const RS = window.RangeStudio;

    function $(s) { return document.querySelector(s); }
    function $$(s) { return document.querySelectorAll(s); }

    // Wizard state - mirrors CreateProjectRequest on the server.
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

    // Cached external data
    const cache = { templates: null, images: null, profiles: null };
    async function loadTemplates() { if (!cache.templates) cache.templates = await RS.api('/api/templates'); return cache.templates; }
    async function loadImages()    { if (!cache.images)    cache.images    = await RS.api('/api/lxd/images'); return cache.images; }
    async function loadProfiles()  { if (!cache.profiles)  cache.profiles  = await RS.api('/api/lxd/profiles'); return cache.profiles; }

    // ---------- Step navigation ----------
    function wizardLabel(step, isCustom) {
        const steps = isCustom
            ? ['Info', 'Template', 'Topology', 'Review']
            : ['Info', 'Template', '', 'Review'];
        return steps[step - 1] || '';
    }

    function showWizStep(step) {
        const isCustom = wizState.template === 'custom';
        if (step === 3 && !isCustom) step = 4;

        $$('.wizard-step').forEach((s) => s.classList.add('hidden'));
        $(`#wizard-step-${step}`).classList.remove('hidden');
        $('#wizard-subtitle').textContent = `Step ${step} of 4 — ${wizardLabel(step, isCustom)}`;
        $('#btn-wizard-create').classList.toggle('hidden', step !== 4);

        updateStepperUI(step, isCustom);

        if (step === 2) renderWizTemplates();
        if (step === 3) renderWizTopology();
        if (step === 4) loadWizPreview();
    }

    function updateStepperUI(currentStep, isCustom) {
        $$('#wizard-stepper .step').forEach((el) => {
            const n = parseInt(el.dataset.step, 10);
            el.classList.remove('active', 'complete', 'skipped');
            if (!isCustom && n === 3) { el.classList.add('skipped'); return; }
            if (n < currentStep) el.classList.add('complete');
            else if (n === currentStep) el.classList.add('active');
        });
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
        if (!name) return RS.showToast('Project name is required', 'error');
        if (!/^[A-Za-z0-9][A-Za-z0-9_-]{1,40}$/.test(name)) {
            return RS.showToast('Name must be 2–41 chars: letters, digits, dash, underscore', 'error');
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

    // ---------- Step 2: Template selection ----------
    async function renderWizTemplates() {
        const templates = await loadTemplates();
        const grid = $('#wiz-template-grid');
        const customSel = wizState.template === 'custom' ? 'selected' : '';
        const customIcon = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M8 2v12M2 8h12" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`;
        const tmplIcon = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none"><path d="M3 2h6l4 4v8a1 1 0 01-1 1H3a1 1 0 01-1-1V3a1 1 0 011-1z" stroke="currentColor" stroke-width="1.4"/><path d="M9 2v4h4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>`;
        const checkSvg = `<svg width="10" height="10" viewBox="0 0 10 10" fill="none"><path d="M2 5l2 2 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>`;

        grid.innerHTML = `
            <div class="template-card custom-build ${customSel}" data-id="custom">
                <div class="template-card-icon">${customIcon}</div>
                <div class="template-title">Build Custom Topology</div>
                <div class="template-desc">Design networks, firewalls, and VMs via the wizard. Generates a fresh <code>main.tf</code>.</div>
                <div class="template-check">${checkSvg}</div>
            </div>
        ` + templates.map((t) => {
            const sel = wizState.template === t.id ? 'selected' : '';
            const desc = t.description || (t.path ? `Source: ${t.path}` : 'Built-in template');
            return `
            <div class="template-card ${sel}" data-id="${RS.escapeAttr(t.id)}">
                <div class="template-card-icon">${tmplIcon}</div>
                <div class="template-title">${RS.escapeHtml(t.id)}</div>
                <div class="template-desc">${RS.escapeHtml(desc)}</div>
                <div class="template-check">${checkSvg}</div>
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

    // ---------- Step 3: Topology builder ----------
    $('#btn-wiz-prev-3').addEventListener('click', () => showWizStep(2));
    $('#btn-wiz-next-3').addEventListener('click', () => {
        if (wizState.topology.networks.length === 0) {
            return RS.showToast('Add at least one network before continuing', 'error');
        }
        if (wizState.topology.networks.some((n) => !n.name)) {
            return RS.showToast('Every network needs a name', 'error');
        }
        showWizStep(4);
    });

    function renderWizTopology() {
        $$('input[name="wiz-net-mode"]').forEach((r) => {
            r.checked = r.value === wizState.topology.network_mode;
        });
        renderWizNetworks();
        renderWizFirewall('team_firewall', '#topo-fw-interfaces');
        renderWizVMs();
        renderTopoSummary();
    }

    function renderTopoSummary() {
        const bar = $('#topo-summary');
        if (!bar) return;
        const teams = wizState.topology.network_mode === 'single' ? 1 : Math.max(1, wizState.teamCount);
        const vmsPerTeam = wizState.topology.vms.length;
        const totalVMs = vmsPerTeam * teams;
        const nets = wizState.topology.networks.length;
        bar.innerHTML = `
            <div class="topo-summary-stat">
                <span class="topo-summary-label">Teams</span>
                <span class="topo-summary-value">${teams}</span>
            </div>
            <div class="topo-summary-stat">
                <span class="topo-summary-label">Networks</span>
                <span class="topo-summary-value">${nets}<span class="stat-unit">per team</span></span>
            </div>
            <div class="topo-summary-stat">
                <span class="topo-summary-label">VMs</span>
                <span class="topo-summary-value">${vmsPerTeam}<span class="stat-unit">per team</span></span>
            </div>
            <div class="topo-summary-stat">
                <span class="topo-summary-label">Total VMs</span>
                <span class="topo-summary-value accent">${totalVMs}</span>
            </div>`;
    }

    function renderWizNetworks() {
        const list = $('#topo-networks');
        if (wizState.topology.networks.length === 0) {
            list.innerHTML = `
                <div class="topo-empty">
                    <div class="topo-empty-title">No networks yet</div>
                    <div class="topo-empty-desc">Click <strong>+ Add Network</strong> to create the first OVN segment.</div>
                </div>`;
            return;
        }
        list.innerHTML = wizState.topology.networks.map((n, i) => `
            <div class="topo-item">
                <span class="net-color-dot"></span>
                <input type="text" placeholder="Network Name (e.g. lan, dmz)" value="${RS.escapeAttr(n.name)}"
                       data-net-i="${i}" data-action="net-name">
                <button class="btn btn-sm btn-danger" data-net-i="${i}" data-action="net-del" title="Remove network">Remove</button>
            </div>
        `).join('');
        list.querySelectorAll('[data-action="net-name"]').forEach((el) => {
            el.addEventListener('change', () => {
                wizState.topology.networks[+el.dataset.netI].name = el.value.trim();
                renderTopoSummary();
            });
        });
        list.querySelectorAll('[data-action="net-del"]').forEach((el) => {
            el.addEventListener('click', () => {
                wizState.topology.networks.splice(+el.dataset.netI, 1);
                renderWizTopology();
            });
        });
    }

    function renderWizFirewall(key, sel) {
        const list = $(sel);
        const fw = wizState.topology[key];
        if (fw.interfaces.length === 0) {
            list.innerHTML = `
                <div class="topo-empty">
                    <div class="topo-empty-title">No interfaces beyond <code>eth0</code></div>
                    <div class="topo-empty-desc">Add an interface to wire the team firewall into your networks.</div>
                </div>`;
            return;
        }
        list.innerHTML = fw.interfaces.map((intf, i) => `
            <div class="topo-item">
                <span class="eth-label">eth${i + (key === 'project_firewall' ? 3 : 1)}</span>
                <select data-fw-i="${i}" data-action="fw-net">
                    <option value="">Select Network...</option>
                    ${wizState.topology.networks.map((n) => `
                        <option value="${RS.escapeAttr(n.name)}" ${intf.network === n.name ? 'selected' : ''}>${RS.escapeHtml(n.name)}</option>
                    `).join('')}
                </select>
                <input type="text" placeholder="IP (e.g. 192.168.1.1/24)" value="${RS.escapeAttr(intf.ip)}"
                       data-fw-i="${i}" data-action="fw-ip">
                <button class="btn btn-sm btn-danger" data-fw-i="${i}" data-action="fw-del">Remove</button>
            </div>
        `).join('');

        const offset = key === 'project_firewall' ? 3 : 1;
        list.querySelectorAll('[data-action="fw-net"]').forEach((el) => {
            el.addEventListener('change', () => {
                const i = +el.dataset.fwI;
                fw.interfaces[i].network = el.value;
                fw.interfaces[i].eth_name = `eth${i + offset}`;
            });
        });
        list.querySelectorAll('[data-action="fw-ip"]').forEach((el) => {
            el.addEventListener('change', () => {
                const i = +el.dataset.fwI;
                fw.interfaces[i].ip = el.value;
                fw.interfaces[i].eth_name = `eth${i + offset}`;
            });
        });
        list.querySelectorAll('[data-action="fw-del"]').forEach((el) => {
            el.addEventListener('click', () => {
                fw.interfaces.splice(+el.dataset.fwI, 1);
                renderWizFirewall(key, sel);
            });
        });
    }

    // Parse "4GiB" → { amount: '4', unit: 'GiB' }. Tolerates "4G", "4GB", "4gib".
    function parseSize(str) {
        if (!str) return { amount: '', unit: 'GiB' };
        const m = String(str).trim().match(/^([\d.]+)\s*([A-Za-z]+)?$/);
        if (!m) return { amount: String(str), unit: 'GiB' };
        const raw = (m[2] || 'GiB').trim();
        const first = raw.charAt(0).toUpperCase();
        const unit = first === 'T' ? 'TiB' : first === 'M' ? 'MiB' : 'GiB';
        return { amount: m[1], unit };
    }
    function joinSize(amount, unit) {
        if (!amount) return '';
        return amount + (unit || 'GiB');
    }
    const SIZE_UNITS = ['MiB', 'GiB', 'TiB'];

    function sizeInputHTML(field, value, placeholder) {
        const { amount, unit } = parseSize(value);
        const u = SIZE_UNITS.includes(unit) ? unit : 'GiB';
        return `
            <div class="size-input">
                <input type="text" inputmode="decimal" placeholder="${placeholder}"
                       value="${RS.escapeAttr(amount)}" data-vm-size-amount="${field}">
                <select data-vm-size-unit="${field}">
                    ${SIZE_UNITS.map((opt) => `<option value="${opt}" ${opt === u ? 'selected' : ''}>${opt}</option>`).join('')}
                </select>
            </div>`;
    }

    // Section header SVGs
    const ICON_COMPUTE   = `<svg width="12" height="12" viewBox="0 0 12 12" fill="none"><rect x="1.5" y="1.5" width="9" height="9" rx="1" stroke="currentColor" stroke-width="1.3"/><path d="M4 4h4v4H4z" stroke="currentColor" stroke-width="1.3"/></svg>`;
    const ICON_NET       = `<svg width="12" height="12" viewBox="0 0 12 12" fill="none"><circle cx="6" cy="6" r="4.5" stroke="currentColor" stroke-width="1.3"/><path d="M1.5 6h9M6 1.5a7 7 0 010 9M6 1.5a7 7 0 000 9" stroke="currentColor" stroke-width="1.1"/></svg>`;
    const ICON_PLACEMENT = `<svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M6 1.5l4 2v3c0 2.5-1.8 4-4 4.5C3.8 10.5 2 9 2 6.5v-3l4-2z" stroke="currentColor" stroke-width="1.3"/></svg>`;
    const ICON_BOOT      = `<svg width="12" height="12" viewBox="0 0 12 12" fill="none"><path d="M6 1v3M3 5l-1 6h8l-1-6" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>`;

    function renderWizVMs() {
        const list = $('#topo-vms');
        const images = cache.images || [];
        const profiles = cache.profiles || [];

        if (wizState.topology.vms.length === 0) {
            list.innerHTML = `
                <div class="topo-empty">
                    <div class="topo-empty-title">No VMs yet</div>
                    <div class="topo-empty-desc">Click <strong>+ Add VM</strong> to define a virtual machine. Each VM will be replicated per team.</div>
                </div>`;
            return;
        }

        const showGPU = wizState.group === 'Research';
        const targets = ['default', 'round_robin', 'micro-01', 'micro-02', 'micro-03', 'micro-04', 'micro-05', 'micro-06', 'micro-08', 'micro-09'];

        list.innerHTML = wizState.topology.vms.map((vm, i) => {
            const ci = vm.cloud_init || { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
            const netIps = vm.net_ips || [];

            return `
            <div class="vm-card" data-vm-i="${i}">
                <div class="vm-card-header">
                    <span class="vm-card-index">${i + 1}</span>
                    <input type="text" class="vm-name" placeholder="VM Name (e.g. Jumpbox)" value="${RS.escapeAttr(vm.name)}" data-vm-field="name">
                    <div class="vm-card-actions">
                        <button class="btn btn-secondary btn-icon" data-vm-action="dup" title="Duplicate VM">
                            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><rect x="3.5" y="3.5" width="7.5" height="7.5" rx="1.5" stroke="currentColor" stroke-width="1.3"/><path d="M3 8.5V3a1 1 0 011-1h5.5" stroke="currentColor" stroke-width="1.3"/></svg>
                        </button>
                        <button class="btn btn-danger btn-icon" data-vm-action="del" title="Remove VM">
                            <svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 4h8M5.5 4V2.5h3V4M4 4l.5 8h5L10 4" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
                        </button>
                    </div>
                </div>

                <div class="vm-section">
                    <div class="vm-section-title">${ICON_COMPUTE} Compute</div>
                    <div class="vm-grid">
                        <div>
                            <label>Image</label>
                            <select data-vm-field="image">
                                <option value="">Select Image...</option>
                                ${images.map((img) => {
                                    const alias = img.aliases[0] || img.fingerprint;
                                    return `<option value="${RS.escapeAttr(alias)}" ${vm.image === alias ? 'selected' : ''}>${RS.escapeHtml(alias)} (${img.type})</option>`;
                                }).join('')}
                            </select>
                        </div>
                        <div>
                            <label>Profile</label>
                            <select data-vm-field="profile">
                                <option value="">Select Profile...</option>
                                ${profiles.map((p) => `
                                    <option value="${RS.escapeAttr(p.name)}" ${vm.profile === p.name ? 'selected' : ''}>${RS.escapeHtml(p.name)}</option>
                                `).join('')}
                            </select>
                        </div>
                        <div>
                            <label>Type</label>
                            <select data-vm-field="type">
                                <option value="virtual-machine" ${vm.type !== 'container' ? 'selected' : ''}>virtual-machine</option>
                                <option value="container" ${vm.type === 'container' ? 'selected' : ''}>container</option>
                            </select>
                        </div>
                        <div>
                            <label>CPU (cores)</label>
                            <input type="text" placeholder="2" value="${RS.escapeAttr(vm.cpu)}" data-vm-field="cpu">
                        </div>
                        <div>
                            <label>Memory</label>
                            ${sizeInputHTML('memory', vm.memory, '4')}
                        </div>
                        <div>
                            <label>Disk</label>
                            ${sizeInputHTML('disk_size', vm.disk_size, '50')}
                        </div>
                    </div>
                </div>

                <div class="vm-section">
                    <div class="vm-section-title">${ICON_NET} Networks &amp; IPs</div>
                    <div class="vm-network-ips">
                        <div class="vm-net-ip-row">
                            <label class="check-pill">
                                <input type="checkbox" ${vm.networks.includes('guac_wan') ? 'checked' : ''} data-vm-net="guac_wan">
                                <span>guac_wan</span>
                            </label>
                            <span class="text-muted text-sm" style="flex:1;">Remote access (DHCP — no IP needed)</span>
                        </div>
                        ${wizState.topology.networks.length === 0 ? `
                            <div class="topo-empty" style="padding:14px;">
                                <div class="topo-empty-desc">Add networks above to attach this VM to them.</div>
                            </div>` : ''}
                        ${wizState.topology.networks.map((n) => {
                            const netIp = netIps.find((nip) => nip.network === n.name) || { network: n.name, ip: '' };
                            const isConnected = vm.networks.includes(n.name);
                            return `
                            <div class="vm-net-ip-row">
                                <label class="check-pill">
                                    <input type="checkbox" ${isConnected ? 'checked' : ''} data-vm-net="${RS.escapeAttr(n.name)}">
                                    <span>${RS.escapeHtml(n.name || '(unnamed)')}</span>
                                </label>
                                <input type="text" placeholder="IP (e.g. 10.10.0.50/24)" value="${RS.escapeAttr(netIp.ip)}"
                                       ${!isConnected ? 'disabled' : ''} data-vm-netip="${RS.escapeAttr(n.name)}">
                            </div>`;
                        }).join('')}
                    </div>
                </div>

                <div class="vm-section">
                    <div class="vm-section-title">${ICON_PLACEMENT} Placement &amp; Power</div>
                    <div class="vm-grid">
                        <div>
                            <label>Cluster placement</label>
                            <select data-vm-field="target">
                                ${targets.map((t) => `
                                    <option value="${t === 'default' ? '' : t}" ${(vm.target || '') === (t === 'default' ? '' : t) ? 'selected' : ''}>${t === 'default' ? 'Cluster default' : t === 'round_robin' ? 'Round-robin (excl. 07)' : t}</option>
                                `).join('')}
                            </select>
                        </div>
                    </div>
                    <div class="vm-toggles">
                        <label class="switch">
                            <input type="checkbox" ${vm.running !== false ? 'checked' : ''} data-vm-bool="running">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">Start on create<small>Uncheck for Windows VMs</small></span>
                        </label>
                        ${showGPU ? `
                        <label class="switch">
                            <input type="checkbox" ${vm.needs_gpu ? 'checked' : ''} data-vm-bool="needs_gpu">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">Pin to GPU node<small>micro-07-gpu</small></span>
                        </label>` : ''}
                    </div>
                </div>

                <details class="vm-section vm-advanced">
                    <summary class="vm-section-title" style="cursor:pointer; margin-bottom: 0;">${ICON_BOOT} Security &amp; Cloud-Init <span style="margin-left:6px;color:var(--fg-faint);font-weight:500;text-transform:none;letter-spacing:0;">(advanced)</span></summary>
                    <div class="vm-toggles" style="margin-top: 14px;">
                        <label class="switch">
                            <input type="checkbox" ${vm.secure_boot ? 'checked' : ''} data-vm-bool="secure_boot">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">Secure Boot<small>Usually off</small></span>
                        </label>
                        <label class="switch">
                            <input type="checkbox" ${vm.csm ? 'checked' : ''} data-vm-bool="csm">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">CSM / Legacy boot</span>
                        </label>
                        <label class="switch">
                            <input type="checkbox" ${ci.ssh_pwauth ? 'checked' : ''} data-vm-ci-bool="ssh_pwauth">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">SSH password auth<small>cloud-init <code>ssh_pwauth</code></small></span>
                        </label>
                        <label class="switch">
                            <input type="checkbox" ${ci.package_update ? 'checked' : ''} data-vm-ci-bool="package_update">
                            <span class="switch-track"><span class="switch-thumb"></span></span>
                            <span class="switch-text">Package update on boot</span>
                        </label>
                    </div>
                    <div class="vm-grid" style="margin-top: 14px;">
                        <div>
                            <label>Swap size</label>
                            <input type="text" placeholder="e.g. 4G" value="${RS.escapeAttr(ci.swap_size || '')}" data-vm-ci="swap_size">
                        </div>
                        <div class="vm-grid-wide">
                            <label>Run commands (one per line)</label>
                            <textarea rows="3" placeholder="echo hello&#10;ufw --force reset" data-vm-ci-runcmds>${RS.escapeHtml((ci.runcmds || []).join('\n'))}</textarea>
                        </div>
                    </div>
                </details>
            </div>`;
        }).join('');

        // Wire VM card events
        list.querySelectorAll('.vm-card').forEach((card) => {
            const i = +card.dataset.vmI;
            const vm = wizState.topology.vms[i];

            card.querySelector('[data-vm-action="del"]').addEventListener('click', () => {
                wizState.topology.vms.splice(i, 1);
                renderWizVMs();
                renderTopoSummary();
            });
            card.querySelector('[data-vm-action="dup"]').addEventListener('click', () => {
                const copy = JSON.parse(JSON.stringify(vm));
                copy.name = (vm.name || `vm${i + 1}`) + '-copy';
                wizState.topology.vms.splice(i + 1, 0, copy);
                renderWizVMs();
                renderTopoSummary();
            });

            // Size inputs (memory / disk_size) — combine amount + unit on either change.
            const syncSize = (field) => {
                const amt  = card.querySelector(`[data-vm-size-amount="${field}"]`);
                const unit = card.querySelector(`[data-vm-size-unit="${field}"]`);
                vm[field] = joinSize((amt.value || '').trim(), unit.value);
            };
            card.querySelectorAll('[data-vm-size-amount]').forEach((el) => {
                el.addEventListener('input',  () => syncSize(el.dataset.vmSizeAmount));
                el.addEventListener('change', () => syncSize(el.dataset.vmSizeAmount));
            });
            card.querySelectorAll('[data-vm-size-unit]').forEach((el) => {
                el.addEventListener('change', () => syncSize(el.dataset.vmSizeUnit));
            });

            card.querySelectorAll('[data-vm-field]').forEach((el) => {
                el.addEventListener('change', () => {
                    vm[el.dataset.vmField] = el.value;
                    if (el.dataset.vmField === 'image' && /win|windows/i.test(el.value)) {
                        vm.running = false;
                        renderWizVMs();
                    }
                });
            });
            card.querySelectorAll('[data-vm-bool]').forEach((el) => {
                el.addEventListener('change', () => { vm[el.dataset.vmBool] = el.checked; });
            });
            card.querySelectorAll('[data-vm-net]').forEach((el) => {
                el.addEventListener('change', () => {
                    const net = el.dataset.vmNet;
                    if (el.checked && !vm.networks.includes(net)) vm.networks.push(net);
                    if (!el.checked) {
                        vm.networks = vm.networks.filter((n) => n !== net);
                        if (vm.net_ips) vm.net_ips = vm.net_ips.filter((nip) => nip.network !== net);
                    }
                    renderWizVMs();
                });
            });
            card.querySelectorAll('[data-vm-netip]').forEach((el) => {
                el.addEventListener('change', () => {
                    const net = el.dataset.vmNetip;
                    if (!vm.net_ips) vm.net_ips = [];
                    const existing = vm.net_ips.find((nip) => nip.network === net);
                    if (existing) existing.ip = el.value;
                    else vm.net_ips.push({ network: net, ip: el.value });
                });
            });
            card.querySelectorAll('[data-vm-ci]').forEach((el) => {
                el.addEventListener('change', () => {
                    if (!vm.cloud_init) vm.cloud_init = { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
                    vm.cloud_init[el.dataset.vmCi] = el.value;
                });
            });
            card.querySelectorAll('[data-vm-ci-bool]').forEach((el) => {
                el.addEventListener('change', () => {
                    if (!vm.cloud_init) vm.cloud_init = { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
                    vm.cloud_init[el.dataset.vmCiBool] = el.checked;
                });
            });
            const runcmds = card.querySelector('[data-vm-ci-runcmds]');
            if (runcmds) runcmds.addEventListener('change', () => {
                if (!vm.cloud_init) vm.cloud_init = { swap_size: '', ssh_pwauth: false, package_update: false, runcmds: [] };
                vm.cloud_init.runcmds = runcmds.value.split('\n').filter((s) => s.trim() !== '');
            });
        });
    }

    $$('input[name="wiz-net-mode"]').forEach((r) => {
        r.addEventListener('change', (e) => {
            wizState.topology.network_mode = e.target.value;
            renderTopoSummary();
        });
    });

    $('#btn-add-network').addEventListener('click', () => {
        wizState.topology.networks.push({ name: '' });
        renderWizTopology();
        const inputs = document.querySelectorAll('#topo-networks .topo-item input[type="text"]');
        const last = inputs[inputs.length - 1];
        if (last) last.focus();
    });
    $('#btn-add-fw-interface').addEventListener('click', () => {
        const fw = wizState.topology.team_firewall;
        const idx = fw.interfaces.length;
        fw.interfaces.push({ eth_name: `eth${idx + 1}`, network: '', ip: '' });
        renderWizFirewall('team_firewall', '#topo-fw-interfaces');
        const rows = document.querySelectorAll('#topo-fw-interfaces .topo-item');
        const last = rows[rows.length - 1];
        if (last) {
            last.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
            const sel = last.querySelector('select');
            if (sel) sel.focus();
        }
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
        // Scroll the new card into view and focus its name input.
        const cards = document.querySelectorAll('#topo-vms .vm-card');
        const last = cards[cards.length - 1];
        if (last) {
            last.scrollIntoView({ behavior: 'smooth', block: 'center' });
            const nameInput = last.querySelector('.vm-name');
            if (nameInput) nameInput.focus();
        }
    });

    // ---------- Step 4: Review ----------
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
        const rows = [
            ['Name', wizState.name],
            ['Group', wizState.group + (wizState.group === 'CIG' ? ` (${wizState.subGroup})` : '')],
            ['Team Count', wizState.teamCount],
            ['Template', wizState.template],
        ];
        if (wizState.group === 'Research') {
            rows.push(['GPU', wizState.needsGPU ? 'Yes (micro-07-gpu)' : 'No']);
            rows.push(['Placement', wizState.placementMode + (wizState.placementMode === 'pinned' ? ` (${wizState.pinnedTarget})` : '')]);
        }
        if (wizState.template === 'custom') {
            rows.push(['Networks', wizState.topology.networks.map((n) => n.name).join(', ') || '—']);
            rows.push(['VMs', wizState.topology.vms.length]);
            rows.push(['Mode', wizState.topology.network_mode]);
        }
        grid.innerHTML = rows.map(([k, v]) =>
            `<div class="review-key">${RS.escapeHtml(k)}</div><div class="review-val">${RS.escapeHtml(String(v))}</div>`
        ).join('');

        const code = $('#wiz-preview-hcl');
        code.textContent = 'Generating preview...';
        try {
            const res = await fetch(window.location.origin + '/api/projects/preview', {
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
            const res = await fetch(window.location.origin + '/api/projects/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(buildPayload()),
            });
            const data = await res.json();
            if (data.error) throw new Error(data.error);
            RS.showToast(`Created ${wizState.name} at ${data.output_dir}`, 'success');
            // Brief delay so the toast is visible before navigation.
            setTimeout(() => { window.location.href = 'index.html'; }, 600);
        } catch (e) {
            RS.showToast(e.message, 'error');
            btn.disabled = false;
            btn.innerHTML = original;
        }
    });

    // Boot
    showWizStep(1);
})();

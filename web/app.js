(function () {
  const statusUrl = '/v1/status';
  const incidentsUrl = '/v1/incidents?state=open';
  const statusIntervalMs = 10000;
  const incidentsIntervalMs = 20000;

  function el(id) {
    return document.getElementById(id);
  }

  function renderRpc(container, data) {
    if (!data || !data.rpc_providers) return;
    container.innerHTML = data.rpc_providers.map(function (p) {
      const cls = p.healthy ? 'healthy' : 'unhealthy';
      const lat = p.latency_ms != null ? p.latency_ms + ' ms' : '—';
      return (
        '<div class="card ' + cls + '">' +
        '<div class="name">' + escapeHtml(p.name) + '</div>' +
        '<div class="latency">' + lat + '</div>' +
        (p.url ? '<div class="url">' + escapeHtml(p.url) + '</div>' : '') +
        (p.last_error ? '<div class="error">' + escapeHtml(p.last_error) + '</div>' : '') +
        '</div>'
      );
    }).join('');
  }

  function renderDapps(container, data) {
    if (!data || !data.dapps) return;
    container.innerHTML = data.dapps.map(function (d) {
      const cls = d.healthy ? 'healthy' : 'unhealthy';
      const lat = d.latency_ms != null ? d.latency_ms + ' ms' : '—';
      return (
        '<div class="card ' + cls + '">' +
        '<div class="name">' + escapeHtml(d.name) + '</div>' +
        '<div class="latency">' + lat + '</div>' +
        (d.url ? '<div class="url">' + escapeHtml(d.url) + '</div>' : '') +
        '</div>'
      );
    }).join('');
  }

  function renderIncidents(listEl, data) {
    if (!listEl) return;
    if (!data || !Array.isArray(data) || data.length === 0) {
      listEl.innerHTML = '<li class="empty">No open incidents.</li>';
      return;
    }
    listEl.innerHTML = data.map(function (i) {
      const cls = i.severity === 'CRIT' ? 'crit' : '';
      return (
        '<li class="' + cls + '">' +
        '<strong>' + escapeHtml(i.entity_name) + '</strong> (' + escapeHtml(i.severity) + ') ' +
        escapeHtml(i.summary) + ' <span class="muted">' + escapeHtml(i.started_at) + '</span>' +
        '</li>'
      );
    }).join('');
  }

  function escapeHtml(s) {
    if (s == null) return '';
    var div = document.createElement('div');
    div.textContent = s;
    return div.innerHTML;
  }

  function fetchStatus() {
    fetch(statusUrl)
      .then(function (r) { return r.ok ? r.json() : Promise.reject(r.status); })
      .then(function (data) {
        el('recommended-rpc').textContent = data.recommended_provider || '—';
        renderRpc(el('rpc-cards'), data);
        renderDapps(el('dapp-cards'), data);
        renderIncidents(el('incidents-list'), data.open_incidents || []);
      })
      .catch(function () {
        el('recommended-rpc').textContent = '—';
        el('rpc-cards').innerHTML = '<p class="muted">Failed to load status.</p>';
        el('dapp-cards').innerHTML = '';
      });
  }

  function fetchIncidents() {
    fetch(incidentsUrl)
      .then(function (r) { return r.ok ? r.json() : Promise.reject(r.status); })
      .then(function (data) {
        renderIncidents(el('incidents-list'), data);
      })
      .catch(function () {});
  }

  fetchStatus();
  fetchIncidents();
  setInterval(fetchStatus, statusIntervalMs);
  setInterval(fetchIncidents, incidentsIntervalMs);
})();

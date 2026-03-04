var API = '/_admin/api/redirects';
var rules = [];

function flash(msg, ok) {
  var el = document.getElementById('status');
  el.textContent = msg;
  el.className = ok ? 'status-ok' : 'status-err';
  el.style.display = 'block';
  setTimeout(function() { el.style.display = 'none'; }, 2000);
}

function load() {
  fetch(API).then(function(res) { return res.json(); }).then(function(data) {
    rules = data;
    render();
  }).catch(function(e) { flash('Failed to load: ' + e.message, false); });
}

function esc(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; }

function render() {
  var wrap = document.getElementById('table-wrap');
  if (!rules || !rules.length) { wrap.innerHTML = '<div class="empty">No redirect rules yet. Add one above.</div>'; return; }
  var html = '<table><thead><tr><th>Host</th><th>Path</th><th>Destination</th><th>Type</th><th>Hits</th><th>Actions</th></tr></thead><tbody>';
  for (var i = 0; i < rules.length; i++) {
    var r = rules[i];
    html += '<tr>';
    html += '<td class="mono">' + esc(r.source_host || '*') + '</td>';
    html += '<td class="mono">' + esc(r.source_path) + (r.is_regex ? ' <span class="badge badge-regex">regex</span>' : '') + '</td>';
    html += '<td class="mono truncate" title="' + esc(r.destination) + '">' + esc(r.destination) + '</td>';
    html += '<td><span class="badge badge-' + r.redirect_type + '">' + r.redirect_type + '</span></td>';
    html += '<td class="hits">' + r.hit_count.toLocaleString() + '</td>';
    html += '<td class="actions">';
    html += '<button class="edit" onclick="editRule(' + r.id + ')">Edit</button>';
    html += '<button class="danger" onclick="deleteRule(' + r.id + ')">Delete</button>';
    html += '</td></tr>';
  }
  html += '</tbody></table>';
  wrap.innerHTML = html;
}

document.getElementById('addForm').addEventListener('submit', function(e) {
  e.preventDefault();
  var body = {
    source_host: document.getElementById('f-host').value.trim(),
    source_path: document.getElementById('f-path').value.trim() || '/',
    destination: document.getElementById('f-dest').value.trim(),
    redirect_type: parseInt(document.getElementById('f-type').value),
    is_regex: document.getElementById('f-regex').checked ? 1 : 0
  };
  if (!body.destination) { flash('Destination required', false); return; }
  fetch(API, { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(body) })
    .then(function(res) { return res.json().then(function(data) { return {ok: res.ok, data: data}; }); })
    .then(function(result) {
      if (!result.ok) { flash(result.data.error || 'Error', false); return; }
      flash('Rule added', true);
      document.getElementById('f-host').value = '';
      document.getElementById('f-path').value = '/';
      document.getElementById('f-dest').value = '';
      document.getElementById('f-regex').checked = false;
      load();
    }).catch(function(e) { flash(e.message, false); });
});

function deleteRule(id) {
  if (!confirm('Delete this rule?')) return;
  fetch(API + '/' + id, { method: 'DELETE' })
    .then(function() { flash('Deleted', true); load(); })
    .catch(function(e) { flash(e.message, false); });
}

function editRule(id) {
  var r = null;
  for (var i = 0; i < rules.length; i++) { if (rules[i].id === id) { r = rules[i]; break; } }
  if (!r) return;
  var host = prompt('Source Host:', r.source_host);
  if (host === null) return;
  var path = prompt('Source Path:', r.source_path);
  if (path === null) return;
  var dest = prompt('Destination:', r.destination);
  if (dest === null) return;
  var type_ = prompt('Redirect Type (301/302):', r.redirect_type);
  if (type_ === null) return;
  var regex = confirm('Is regex pattern?');

  fetch(API + '/' + id, {
    method: 'PUT',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({
      source_host: host,
      source_path: path || '/',
      destination: dest,
      redirect_type: parseInt(type_) || 301,
      is_regex: regex ? 1 : 0
    })
  }).then(function(res) { return res.json(); }).then(function(data) {
    if (data.error) { flash(data.error, false); return; }
    flash('Updated', true);
    load();
  }).catch(function(e) { flash(e.message, false); });
}

load();

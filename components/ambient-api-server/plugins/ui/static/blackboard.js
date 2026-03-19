// Blackboard — project-scoped agent fleet dashboard
// Driven by:
//   GET /api/ambient/v1/agents?search=project_id='...'
//   GET /api/ambient/v1/session_check_ins?search=agent_id='...'  (per agent, batched)
//   GET /api/ambient/v1/projects/{id}/blackboard  (SSE, when endpoint lands)
//   POST /api/ambient/v1/agents/{id}/ignite
//   POST /api/ambient/v1/agents/{id}/inbox
//   GET /api/ambient/v1/project_documents?search=project_id='...'&slug='protocol'

var Blackboard = (function () {
  var API = '/api/ambient/v1';
  var projectId = null;
  var agents = [];           // flat list
  var checkIns = {};         // agent_id → latest SessionCheckIn
  var collapsed = {};        // agent_id → bool
  var sseSource = null;
  var logEntries = [];

  // ── Public init ──────────────────────────────────────────────────────────

  function init(pid, container) {
    projectId = pid;
    render(container, '<div class="bb-loading">Loading agents...</div>');
    loadAll(container);
  }

  function destroy() {
    if (sseSource) { sseSource.close(); sseSource = null; }
  }

  // ── Data loading ─────────────────────────────────────────────────────────

  function loadAll(container) {
    fetch(API + '/agents?search=project_id%3D%27' + encodeURIComponent(projectId) + '%27&size=100')
      .then(function (r) { return r.json(); })
      .then(function (data) {
        agents = (data.items || []).sort(function (a, b) {
          return (a.display_name || a.name || '').localeCompare(b.display_name || b.name || '');
        });
        return loadCheckIns();
      })
      .then(function () {
        renderBoard(container);
        subscribeSSE(container);
      })
      .catch(function (e) {
        render(container, '<div class="bb-error">Failed to load agents: ' + esc(e.message) + '</div>');
      });
  }

  function loadCheckIns() {
    if (agents.length === 0) return Promise.resolve();
    var ids = agents.map(function (a) { return a.id; });
    var search = 'agent_id in (' + ids.map(function (id) { return "'" + id + "'"; }).join(',') + ')';
    return fetch(API + '/session_check_ins?search=' + encodeURIComponent(search) + '&size=200&orderBy=created_at desc')
      .then(function (r) { return r.json(); })
      .then(function (data) {
        checkIns = {};
        (data.items || []).forEach(function (ci) {
          if (!checkIns[ci.agent_id]) checkIns[ci.agent_id] = ci;
        });
      })
      .catch(function () {});
  }

  // ── SSE ──────────────────────────────────────────────────────────────────

  function subscribeSSE(container) {
    // Wire up when /projects/{id}/blackboard SSE endpoint exists.
    // For now, poll check-ins every 30s as a fallback.
    var interval = setInterval(function () {
      loadCheckIns().then(function () { renderBoard(container); });
    }, 30000);

    // Store cleanup handle
    if (sseSource) clearInterval(sseSource);
    sseSource = interval;
  }

  // ── Tree building ─────────────────────────────────────────────────────────

  function buildTree() {
    var byId = {};
    agents.forEach(function (a) { byId[a.id] = a; });

    var roots = [];
    var children = {};
    agents.forEach(function (a) {
      var pid = a.parent_agent_id;
      if (pid && byId[pid]) {
        if (!children[pid]) children[pid] = [];
        children[pid].push(a);
      } else {
        roots.push(a);
      }
    });

    var rows = [];
    function walk(agent, depth) {
      rows.push({ agent: agent, depth: depth });
      if (!collapsed[agent.id] && children[agent.id]) {
        children[agent.id].forEach(function (child) { walk(child, depth + 1); });
      }
    }
    roots.forEach(function (r) { walk(r, 0); });
    return rows;
  }

  // ── Rendering ─────────────────────────────────────────────────────────────

  function renderBoard(container) {
    var rows = buildTree();
    var hasChildren = {};
    agents.forEach(function (a) {
      if (a.parent_agent_id) hasChildren[a.parent_agent_id] = true;
    });

    var o = '<div class="bb-wrap">';

    // Toolbar
    o += '<div class="bb-toolbar">';
    o += '<button class="btn btn-sm" onclick="Blackboard.newAgent()">⊕ New Agent</button>';
    o += '<button class="btn btn-sm" onclick="Blackboard.showProtocol()">📄 Protocol</button>';
    o += '<button class="btn btn-sm" onclick="Blackboard.refresh()">↻ Refresh</button>';
    o += '</div>';

    if (rows.length === 0) {
      o += '<div class="bb-empty">No agents in this project. Create one with ⊕ New Agent.</div>';
    } else {
      // Table
      o += '<div class="bb-table-wrap"><table class="bb-table">';
      o += '<thead><tr>';
      o += '<th style="width:220px">Agent</th>';
      o += '<th style="width:90px">Status</th>';
      o += '<th style="width:130px">Branch</th>';
      o += '<th style="width:70px">PR</th>';
      o += '<th style="width:60px">Tests</th>';
      o += '<th>Summary</th>';
      o += '<th style="width:130px">Actions</th>';
      o += '</tr></thead><tbody>';

      rows.forEach(function (row) {
        var a = row.agent;
        var ci = checkIns[a.id];
        var indent = row.depth * 20;
        var isParent = hasChildren[a.id];
        var isCollapsed = collapsed[a.id];

        var phase = ci ? ci.phase : (a.current_session_id ? 'active' : 'idle');
        var phaseClass = phaseToClass(phase);

        o += '<tr class="bb-row" data-agent-id="' + esc(a.id) + '">';

        // Agent name + indent + collapse
        o += '<td style="padding-left:' + (12 + indent) + 'px">';
        if (isParent) {
          o += '<span class="bb-collapse" onclick="Blackboard.toggleCollapse(\'' + esc(a.id) + '\')">';
          o += isCollapsed ? '▶ ' : '▼ ';
          o += '</span>';
        } else {
          o += '<span style="display:inline-block;width:14px"></span>';
        }
        o += '<span class="bb-agent-name">' + esc(a.display_name || a.name) + '</span>';
        o += '</td>';

        // Status
        o += '<td><span class="bb-phase ' + phaseClass + '">' + esc(phase) + '</span></td>';

        // Branch
        o += '<td class="bb-mono bb-truncate">' + esc((ci && ci.branch) || '—') + '</td>';

        // PR
        if (ci && ci.pr) {
          o += '<td><a href="' + esc(ci.pr) + '" target="_blank" class="bb-link">PR</a></td>';
        } else {
          o += '<td class="bb-dim">—</td>';
        }

        // Tests
        o += '<td class="bb-center">' + (ci && ci.test_count ? esc(String(ci.test_count)) : '<span class="bb-dim">—</span>') + '</td>';

        // Summary — questions amber, blockers red
        var summary = (ci && ci.summary) || '';
        var questions = (ci && ci.questions) || '';
        var blockers = (ci && ci.blockers) || '';
        var summaryHtml = esc(summary.length > 80 ? summary.substring(0, 77) + '…' : summary);
        if (blockers) summaryHtml += ' <span class="bb-blocker" title="' + esc(blockers) + '">!</span>';
        if (questions) summaryHtml += ' <span class="bb-question" title="' + esc(questions) + '">?</span>';
        o += '<td class="bb-summary">' + (summaryHtml || '<span class="bb-dim">—</span>') + '</td>';

        // Actions
        o += '<td class="bb-actions">';
        o += '<button class="btn btn-sm bb-btn-ignite" onclick="Blackboard.ignite(\'' + esc(a.id) + '\',\'' + esc(a.display_name || a.name) + '\')">▶ Ignite</button> ';
        o += '<button class="btn btn-sm" onclick="Blackboard.openInbox(\'' + esc(a.id) + '\',\'' + esc(a.display_name || a.name) + '\')">✉</button>';
        o += '</td>';

        o += '</tr>';
      });

      o += '</tbody></table></div>';
    }

    // Log strip
    if (logEntries.length > 0) {
      o += '<div class="bb-log">';
      logEntries.slice(-10).forEach(function (e) {
        o += '<span class="' + e.cls + '">[' + esc(e.time) + '] ' + esc(e.msg) + '</span>  ';
      });
      o += '</div>';
    }

    o += '</div>';
    render(container, o);
  }

  // ── Actions ───────────────────────────────────────────────────────────────

  function ignite(agentId, agentName) {
    log('info', 'Igniting ' + agentName + '…');
    fetch(API + '/agents/' + encodeURIComponent(agentId) + '/ignite', { method: 'POST' })
      .then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); })
      .then(function (data) {
        log('ok', agentName + ' ignited → session ' + ((data.session && data.session.id) || '?'));
        if (data.ignition_prompt) showIgnitionPrompt(agentName, data.ignition_prompt);
      })
      .catch(function (e) { log('err', 'Ignite failed: ' + e.message); });
  }

  function openInbox(agentId, agentName) {
    var msg = prompt('Message to ' + agentName + ':');
    if (!msg) return;
    log('info', 'Sending message to ' + agentName + '…');
    fetch(API + '/agents/' + encodeURIComponent(agentId) + '/inbox', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ body: msg, sender_name: 'human' })
    })
      .then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); })
      .then(function () { log('ok', 'Message sent to ' + agentName); })
      .catch(function (e) { log('err', 'Send failed: ' + e.message); });
  }

  function showProtocol() {
    fetch(API + '/project_documents?search=' + encodeURIComponent("project_id = '" + projectId + "' and slug = 'protocol'"))
      .then(function (r) { return r.json(); })
      .then(function (data) {
        var doc = data.items && data.items[0];
        if (!doc) { alert('No protocol document found.\n\nCreate one via:\nPUT /projects/' + projectId + '/documents/protocol'); return; }
        showModal(doc.title || 'Protocol', '<pre style="white-space:pre-wrap;font-size:12px;color:var(--text2)">' + esc(doc.content) + '</pre>');
      })
      .catch(function (e) { log('err', 'Protocol fetch failed: ' + e.message); });
  }

  function newAgent() {
    showModal('New Agent', [
      '<div style="display:grid;gap:10px">',
      '<input id="bb-na-name" class="bb-input" placeholder="name (e.g. api-agent)" />',
      '<input id="bb-na-display" class="bb-input" placeholder="display name" />',
      '<textarea id="bb-na-prompt" class="bb-input" rows="4" placeholder="system prompt…"></textarea>',
      '<input id="bb-na-repo" class="bb-input" placeholder="repo_url (optional)" />',
      '<button class="btn btn-primary" onclick="Blackboard._createAgent()">Create</button>',
      '</div>'
    ].join(''));
  }

  function _createAgent() {
    var name = (document.getElementById('bb-na-name') || {}).value || '';
    var display = (document.getElementById('bb-na-display') || {}).value || '';
    var prompt = (document.getElementById('bb-na-prompt') || {}).value || '';
    var repo = (document.getElementById('bb-na-repo') || {}).value || '';
    if (!name) { alert('Name is required'); return; }
    closeModal();
    log('info', 'Creating agent ' + name + '…');
    fetch(API + '/agents', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: name,
        display_name: display || name,
        prompt: prompt,
        repo_url: repo,
        project_id: projectId,
        owner_user_id: 'human'
      })
    })
      .then(function (r) { if (!r.ok) return r.text().then(function (t) { throw new Error(t); }); return r.json(); })
      .then(function (a) {
        log('ok', 'Agent ' + (a.display_name || a.name) + ' created');
        agents.push(a);
        refresh();
      })
      .catch(function (e) { log('err', 'Create failed: ' + e.message); });
  }

  function toggleCollapse(agentId) {
    collapsed[agentId] = !collapsed[agentId];
    var container = document.getElementById('bb-container');
    if (container) renderBoard(container);
  }

  function refresh() {
    var container = document.getElementById('bb-container');
    if (!container) return;
    loadAll(container);
  }

  // ── Modal ─────────────────────────────────────────────────────────────────

  function showIgnitionPrompt(agentName, promptText) {
    showModal('Ignition Prompt — ' + agentName,
      '<pre style="white-space:pre-wrap;font-size:11px;color:var(--text2);max-height:60vh;overflow-y:auto">' + esc(promptText) + '</pre>'
    );
  }

  function showModal(title, body) {
    closeModal();
    var el = document.createElement('div');
    el.id = 'bb-modal-overlay';
    el.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,.7);z-index:500;display:flex;align-items:center;justify-content:center';
    el.innerHTML = [
      '<div style="background:var(--bg2);border:1px solid var(--border2);border-radius:var(--radius);width:600px;max-width:90vw;max-height:80vh;display:flex;flex-direction:column">',
      '<div style="padding:12px 16px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;align-items:center">',
      '<span style="font-weight:700;font-size:14px">' + esc(title) + '</span>',
      '<button onclick="Blackboard.closeModal()" style="background:none;border:none;color:var(--text2);cursor:pointer;font-size:18px;line-height:1">×</button>',
      '</div>',
      '<div style="padding:16px;overflow-y:auto">' + body + '</div>',
      '</div>'
    ].join('');
    el.addEventListener('click', function (e) { if (e.target === el) closeModal(); });
    document.body.appendChild(el);
  }

  function closeModal() {
    var el = document.getElementById('bb-modal-overlay');
    if (el) el.remove();
  }

  // ── Helpers ───────────────────────────────────────────────────────────────

  function phaseToClass(phase) {
    switch ((phase || '').toLowerCase()) {
      case 'running': case 'active': return 'bb-phase-active';
      case 'idle': case 'completed': return 'bb-phase-idle';
      case 'failed': return 'bb-phase-failed';
      case 'pending': return 'bb-phase-pending';
      default: return 'bb-phase-idle';
    }
  }

  function log(cls, msg) {
    logEntries.push({ cls: 'bb-log-' + cls, time: new Date().toLocaleTimeString(), msg: msg });
    var container = document.getElementById('bb-container');
    if (container) renderBoard(container);
  }

  function render(container, html) {
    if (typeof container === 'string') container = document.getElementById(container);
    if (container) container.innerHTML = html;
  }

  function esc(s) {
    if (!s) return '';
    var d = document.createElement('div');
    d.textContent = String(s);
    return d.innerHTML;
  }

  return {
    init: init,
    destroy: destroy,
    refresh: refresh,
    toggleCollapse: toggleCollapse,
    ignite: ignite,
    openInbox: openInbox,
    showProtocol: showProtocol,
    newAgent: newAgent,
    _createAgent: _createAgent,
    closeModal: closeModal
  };
})();

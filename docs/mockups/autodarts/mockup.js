const order = [20, 1, 18, 4, 13, 6, 10, 15, 2, 17, 3, 19, 7, 16, 8, 11, 14, 9, 12, 5];
const $ = (selector) => document.querySelector(selector);
$('.table-wrap').insertAdjacentHTML('beforebegin', '<p class="table-scroll-hint">Scroll the table horizontally for all statistics.</p>');
let selectedPlayer = 'winner';
let capture;

function point(radius, angle) {
  return [250 + radius * Math.sin(angle), 250 - radius * Math.cos(angle)];
}

function wedge(inner, outer, angle) {
  const start = angle - Math.PI / 20;
  const end = angle + Math.PI / 20;
  return `M${point(inner, start)} L${point(outer, start)} A${outer},${outer} 0 0 1 ${point(outer, end)} L${point(inner, end)} A${inner},${inner} 0 0 0 ${point(inner, start)}Z`;
}

function dartsFor(player, leg) {
  const id = player === 'winner' ? capture.winnerId : capture.players.find(p => p.id !== capture.winnerId).id;
  return capture.darts.filter(dart => dart.playerId === id && (leg === 'all' || dart.leg === Number(leg)));
}

function counts(player, leg) {
  const groups = new Map();
  dartsFor(player, leg).forEach(dart => {
    const group = groups.get(dart.segment) ?? { segment: dart.segment, count: 0, x: 0, y: 0 };
    group.count++;
    group.x += dart.x;
    group.y += dart.y;
    groups.set(dart.segment, group);
  });
  return [...groups.values()].map(group => ({ ...group, x: group.x / group.count, y: group.y / group.count }));
}

function drawBoard(svg, player, leg, prefix) {
  if (dartsFor(player, leg).some(dart => dart.x === null || dart.y === null)) {
    svg.innerHTML = `<title id="${prefix}-title">Coordinates unavailable</title><desc id="${prefix}-description">No-coordinate fixture. Segment counts remain available below; no positions are invented.</desc><text x="250" y="245" text-anchor="middle" fill="#f6f1eb" font-size="22">Coordinates unavailable</text><text x="250" y="280" text-anchor="middle" fill="#a29d97" font-size="16">See textual segment totals</text>`;
    return;
  }
  const data = counts(player, leg);
  const total = data.reduce((sum, item) => sum + item.count, 0);
  const desc = `${total} synthetic darts: ${data.map(item => `${item.count} at ${item.segment}`).join(', ')}. Randomized coordinates from the supplied fixture, not measured physical accuracy.`;
  let markup = `<title id="${prefix}-title">Dart placement heatmap</title><desc id="${prefix}-description">${desc}</desc><defs><radialGradient id="${prefix}-heat"><stop stop-color="#fff0b0" stop-opacity=".96"/><stop offset=".25" stop-color="#ffc66d" stop-opacity=".9"/><stop offset=".6" stop-color="#ff5b5b" stop-opacity=".65"/><stop offset="1" stop-color="#ff5b5b" stop-opacity="0"/></radialGradient></defs><circle cx="250" cy="250" r="244" fill="#090909" stroke="#343436" stroke-width="2"/><circle cx="250" cy="250" r="228" fill="#121314"/>`;
  order.forEach((number, index) => {
    const angle = index * Math.PI / 10;
    [[8, 111], [111, 121], [121, 181], [181, 191]].forEach(([inner, outer], ring) => {
      const fill = ring % 2 ? (index % 2 ? '#355e51' : '#9b4347') : (index % 2 ? '#c8c1b6' : '#1a1b1e');
      markup += `<path d="${wedge(inner, outer, angle)}" fill="${fill}" stroke="#78736d" stroke-width=".65"/>`;
    });
    const [x, y] = point(212, angle);
    markup += `<text x="${x}" y="${y}" dy=".35em" text-anchor="middle" fill="#c8c1b6" font-family="Barlow,sans-serif" font-size="16">${number}</text>`;
  });
  markup += '<circle cx="250" cy="250" r="19" fill="#355e51" stroke="#78736d"/><circle cx="250" cy="250" r="8" fill="#9b4347" stroke="#78736d"/>';
  const darts = dartsFor(player, leg);
  darts.forEach(({ x, y }) => {
    markup += `<circle cx="${250 + x * 191}" cy="${250 - y * 191}" r="19" fill="url(#${prefix}-heat)" opacity=".3"/>`;
  });
  darts.forEach(({ x, y, segment, visit, dart }) => {
    markup += `<circle data-dart cx="${250 + x * 191}" cy="${250 - y * 191}" r="2.2" fill="#fff0b0" stroke="#090909" stroke-width=".5"><title>Visit ${visit + 1}, dart ${dart + 1}: ${segment}</title></circle>`;
  });
  data.forEach(({ segment, count, x, y }) => {
    const cx = 250 + x * 191;
    const cy = 250 - y * 191;
    markup += `<g><title>${segment}: ${count} darts</title><rect x="${cx + 20}" y="${cy - 9}" width="29" height="20" rx="10" fill="#090909" stroke="#ffc66d"/><text x="${cx + 34.5}" y="${cy + 5}" text-anchor="middle" fill="#fff0b0" font-size="12" font-family="Barlow,sans-serif" font-weight="600">${count}</text></g>`;
  });
  svg.innerHTML = markup;
  if (prefix === 'board') svg.setAttribute('aria-labelledby', 'board-title board-description');
}

function showView(view) {
  $('#review-view').hidden = view !== 'review';
  $('#analysis-view').hidden = view !== 'analysis';
  document.querySelectorAll('[data-view]').forEach(button => button.setAttribute('aria-pressed', String(button.dataset.view === view)));
}

function updateAnalysis() {
  const leg = $('#leg-filter').value;
  const data = counts(selectedPlayer, leg);
  const player = capture.players.find(player => (player.id === capture.winnerId) === (selectedPlayer === 'winner'));
  const stats = (leg === 'all' ? capture.matchStats : capture.legStats.find(item => item.leg === Number(leg)).stats).find(item => item.playerId === player.id);
  drawBoard($('#full-board'), selectedPlayer, leg, 'board');
  $('#scope-title').textContent = `${player.name.toUpperCase()} / ${leg === 'all' ? 'WHOLE MATCH' : `LEG ${leg}`}`;
  $('#dart-count').innerHTML = `${data.reduce((sum, item) => sum + item.count, 0)} <small>darts</small>`;
  $('#selected-average').textContent = stats.average.toFixed(1);
  $('#selected-first9').textContent = stats.first9Average.toFixed(1);
  $('#selected-checkout').textContent = `${stats.checkoutsHit} / ${stats.checkouts}`;
  $('#selected-180').textContent = String(stats.total180);
  const max = Math.max(...data.map(item => item.count));
  $('#distribution').innerHTML = data.map(item => `<div class="segment-row"><span>${item.segment}</span><div class="track"><div class="fill" style="width:${item.count / max * 100}%"></div></div><b>${item.count}</b></div>`).join('');
}

function updateMapping() {
  const valid = $('#division').value && $('#player-one').value && $('#player-two').value && $('#player-one').value !== $('#player-two').value;
  $('#confirm').disabled = !valid;
  $('#player-one').disabled = !$('#division').value;
  $('#player-two').disabled = !$('#division').value;
  $('#mapping-badge').textContent = valid ? 'Demo mapping ready' : 'Player mapping needed';
  $('#mapping-badge').className = `badge ${valid ? 'good' : 'warning'}`;
  $('#mapping-note').textContent = valid ? 'Sample mapping only. This test opponent is not a verified league player.' : 'Choose a league and two different players. The score cannot be confirmed until both sides are mapped.';
}

document.querySelectorAll('[data-view]').forEach(button => button.addEventListener('click', () => showView(button.dataset.view)));
$('#open-analysis').addEventListener('click', () => { showView('analysis'); $('[data-view="analysis"]').focus(); });
document.querySelectorAll('[data-player]').forEach(button => button.addEventListener('click', () => {
  selectedPlayer = button.dataset.player;
  document.querySelectorAll('[data-player]').forEach(item => item.setAttribute('aria-pressed', String(item === button)));
  updateAnalysis();
}));
$('#leg-filter').addEventListener('change', updateAnalysis);
document.querySelectorAll('.mapping-grid select').forEach(select => select.addEventListener('change', updateMapping));
$('#mapping-demo').addEventListener('click', () => {
  const unmatched = $('#player-two').value !== '';
  $('#player-two').value = unmatched ? '' : 'casey';
  $('#mapping-demo').textContent = unmatched ? 'Show mapped state' : 'Show unmatched state';
  showView('review');
  updateMapping();
});
$('#confirm').addEventListener('click', () => { $('#action-status').textContent = 'Preview: the 3-0 result and imported statistics would be recorded with an admin audit entry. Nothing was saved.'; $('#review-status').textContent = 'Confirmed preview'; $('#review-status').className = 'badge good'; });
$('#reject').addEventListener('click', () => { $('#action-status').textContent = 'Preview: this import would be rejected without changing league standings. Nothing was saved.'; $('#review-status').textContent = 'Rejected preview'; $('#review-status').className = 'badge warning'; });
const noCoordinates = new URLSearchParams(location.search).get('coordinates') === 'missing';
fetch(`../../autodarts-fixtures/${noCoordinates ? 'missing-coordinate' : 'randomized'}.json`).then(response => {
  if (!response.ok) throw new Error(`Capture load failed: ${response.status}`);
  return response.json();
}).then(data => {
  capture = data;
  if (noCoordinates) {
    document.querySelectorAll('.coordinate-note, .mini-map-footer > span').forEach(node => node.textContent = 'Coordinates unavailable. Segment totals retained; no positions invented.');
    $('[data-board="preview"]').setAttribute('aria-label', 'Coordinates unavailable; winner segment totals: T20 21, T19 3, D12 3');
    $('.legend').hidden = true;
  }
  drawBoard($('[data-board="preview"]'), 'winner', 'all', 'preview');
  updateAnalysis();
  updateMapping();
  showView(new URLSearchParams(location.search).get('view') === 'analysis' ? 'analysis' : 'review');
  document.body.dataset.captureLoaded = 'true';
}).catch(error => {
  $('#action-status').textContent = `Could not load synthetic match data: ${error.message}`;
  $('#confirm').disabled = true;
});

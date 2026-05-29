/**
 * Ambient UI Preview Bridge
 *
 * Include this script in pages rendered inside the Ambient UI preview iframe
 * to enable cross-origin element capture and hover highlighting.
 *
 * Usage: <script src="/preview-bridge.js"></script>
 *
 * The bridge listens for postMessage requests from the parent frame and
 * responds with element information at the requested coordinates.
 */
(function () {
  'use strict';

  var currentHighlight = null;

  // Element capture: parent asks for the element at (x, y)
  window.addEventListener('message', function (e) {
    if (!e.data || e.data.type !== 'ambient-capture') return;

    var x = e.data.x;
    var y = e.data.y;
    var el = document.elementFromPoint(x, y);

    if (!el) {
      e.source.postMessage({ type: 'ambient-captured', html: null, rect: null }, '*');
      return;
    }

    var rect = el.getBoundingClientRect();
    e.source.postMessage({
      type: 'ambient-captured',
      html: el.outerHTML.slice(0, 500),
      tagName: el.tagName.toLowerCase(),
      id: el.id || null,
      className: el.className || null,
      textContent: (el.textContent || '').slice(0, 100),
      rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
    }, '*');
  });

  // Hover highlight: parent sends cursor position, bridge outlines the element
  window.addEventListener('message', function (e) {
    if (!e.data || e.data.type !== 'ambient-hover') return;

    var el = document.elementFromPoint(e.data.x, e.data.y);

    // Remove previous highlight
    if (currentHighlight) {
      currentHighlight.style.outline = currentHighlight._ambientSavedOutline || '';
      delete currentHighlight._ambientSavedOutline;
      currentHighlight = null;
    }

    if (el && el !== document.documentElement && el !== document.body) {
      el._ambientSavedOutline = el.style.outline;
      el.style.outline = '2px solid #4394e5';
      currentHighlight = el;
    }

    if (el) {
      var rect = el.getBoundingClientRect();
      e.source.postMessage({
        type: 'ambient-hovered',
        rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
      }, '*');
    }
  });

  // Clear hover: remove any active highlight
  window.addEventListener('message', function (e) {
    if (!e.data || e.data.type !== 'ambient-hover-clear') return;
    if (currentHighlight) {
      currentHighlight.style.outline = currentHighlight._ambientSavedOutline || '';
      delete currentHighlight._ambientSavedOutline;
      currentHighlight = null;
    }
  });
})();

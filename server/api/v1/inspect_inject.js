(function () {
  if (window.__dqInspectInstalled) return;
  window.__dqInspectInstalled = true;

  var overlay = null;
  var active = false;

  function ensureOverlay() {
    if (overlay && overlay.isConnected) return overlay;
    overlay = document.createElement('div');
    overlay.id = '__dq_inspect_overlay';
    overlay.style.cssText =
      'position:fixed;pointer-events:none;z-index:2147483646;border:2px solid #3b82f6;background:rgba(59,130,246,0.12);transition:all 0.08s ease;display:none;box-sizing:border-box;';
    document.documentElement.appendChild(overlay);
    return overlay;
  }

  function isSkip(el) {
    return !el || el === document.body || el === document.documentElement || el.id === '__dq_inspect_overlay';
  }

  function isInteractive(el) {
    if (!el || el.nodeType !== 1) return false;
    var tag = el.tagName.toLowerCase();
    if (/^(button|a|input|select|textarea|summary|option|label)$/.test(tag)) return true;
    if (el.getAttribute('role')) return true;
    if (el.getAttribute('tabindex') != null) return true;
    if (el.onclick || typeof el.onclick === 'function') return true;
    return false;
  }

  function resolveTarget(el) {
    if (!el || el.nodeType !== 1) return el;
    var tag = el.tagName.toLowerCase();
    if (/^(svg|path|circle|rect|line|polyline|polygon|g|use|i|span)$/.test(tag) || el.closest('svg')) {
      var cur = el;
      while (cur && cur !== document.body) {
        if (isInteractive(cur)) return cur;
        cur = cur.parentElement;
      }
    }
    return el;
  }

  function cssEscape(s) {
    if (window.CSS && CSS.escape) return CSS.escape(s);
    return String(s).replace(/([ !"#$%&'()*+,./:;<=>?@[\\\]^`{|}~])/g, '\\$1');
  }

  function isUtilityClass(c) {
    return /^(hover:|focus:|active:|sm:|md:|lg:|xl:|2xl:|dark:|light:)/.test(c) ||
      /^(p|m|px|py|mx|my|pt|pb|pl|pr|mt|mb|ml|mr|w|h|min-w|min-h|max-w|max-h|gap|space|text|bg|border|rounded|flex|grid|col|row|justify|items|self|place|font|leading|tracking|opacity|z|top|left|right|bottom|inset|shadow|ring|transition|duration|ease|animate|overflow|cursor|pointer|select|whitespace|truncate|sr-only)-/.test(c) ||
      /^(flex|grid|block|inline|hidden|relative|absolute|fixed|sticky|static)$/.test(c);
  }

  function uniqueCss(sel) {
    try {
      return document.querySelectorAll(sel).length === 1;
    } catch (e) {
      return false;
    }
  }

  function buildSelectors(el) {
    var fallbacks = [];
    var css = '';
    var tag = el.tagName.toLowerCase();
    var testId =
      el.getAttribute('data-testid') ||
      el.getAttribute('data-test') ||
      el.getAttribute('data-qa') ||
      el.getAttribute('data-cy');

    if (testId) {
      var attr =
        el.hasAttribute('data-testid') ? 'data-testid' :
        el.hasAttribute('data-test') ? 'data-test' :
        el.hasAttribute('data-qa') ? 'data-qa' : 'data-cy';
      var s = tag + '[' + attr + '="' + cssEscape(testId) + '"]';
      if (uniqueCss(s)) css = s;
      else fallbacks.push(s);
    }

    if (!css && el.id && !/^[0-9a-f]{8}-[0-9a-f]{4}-/i.test(el.id) && el.id.length < 80) {
      var idSel = '#' + cssEscape(el.id);
      if (uniqueCss(idSel)) css = idSel;
      else fallbacks.push(idSel);
    }

    if (!css) {
      var role = el.getAttribute('role');
      var aria = el.getAttribute('aria-label');
      if (role && aria) {
        var ra = tag + '[role="' + cssEscape(role) + '"][aria-label="' + cssEscape(aria) + '"]';
        if (uniqueCss(ra)) css = ra;
        else fallbacks.push(ra);
      } else if (aria) {
        var aSel = tag + '[aria-label="' + cssEscape(aria) + '"]';
        if (uniqueCss(aSel)) css = aSel;
        else fallbacks.push(aSel);
      }
    }

    if (!css && el.name) {
      var nSel = tag + '[name="' + cssEscape(el.name) + '"]';
      if (uniqueCss(nSel)) css = nSel;
      else fallbacks.push(nSel);
    }

    if (!css) {
      var classes = Array.prototype.slice.call(el.classList || []).filter(function (c) {
        return c && !isUtilityClass(c) && !/^ng-|^_|data-v-|^css-|^svelte-/.test(c);
      }).slice(0, 3);
      if (classes.length) {
        var cSel = tag + '.' + classes.map(cssEscape).join('.');
        if (uniqueCss(cSel)) css = cSel;
        else fallbacks.push(cSel);
      }
    }

    if (!css) {
      var parts = [];
      var cur = el;
      var depth = 0;
      while (cur && cur.nodeType === 1 && cur !== document.body && depth < 5) {
        var t = cur.tagName.toLowerCase();
        var parent = cur.parentElement;
        if (parent) {
          var siblings = Array.prototype.filter.call(parent.children, function (c) {
            return c.tagName === cur.tagName;
          });
          if (siblings.length > 1) {
            var idx = siblings.indexOf(cur) + 1;
            t += ':nth-of-type(' + idx + ')';
          }
        }
        parts.unshift(t);
        cur = parent;
        depth++;
      }
      css = parts.join(' > ');
      fallbacks.push(css);
    }

    return { css: css, fallbacks: fallbacks.slice(0, 4) };
  }

  function buildXPath(el) {
    if (el.id) return '//*[@id="' + el.id.replace(/"/g, '\\"') + '"]';
    var parts = [];
    var cur = el;
    while (cur && cur.nodeType === 1 && cur !== document.documentElement) {
      var tag = cur.tagName.toLowerCase();
      var parent = cur.parentNode;
      if (parent) {
        var same = 0;
        var idx = 0;
        for (var i = 0; i < parent.childNodes.length; i++) {
          var n = parent.childNodes[i];
          if (n.nodeType === 1 && n.tagName === cur.tagName) {
            same++;
            if (n === cur) idx = same;
          }
        }
        parts.unshift(same > 1 ? tag + '[' + idx + ']' : tag);
      } else {
        parts.unshift(tag);
      }
      cur = parent;
    }
    return '/' + parts.join('/');
  }

  function collectAttributes(el) {
    var out = {};
    if (!el.attributes) return out;
    var skip = /^(style|class|id)$|^on|^data-v-|^_ng|^ng-reflect/;
    for (var i = 0; i < el.attributes.length && Object.keys(out).length < 20; i++) {
      var a = el.attributes[i];
      if (skip.test(a.name)) continue;
      if (a.value && a.value.length > 200) continue;
      out[a.name] = a.value;
    }
    return out;
  }

  function detectComponent(el) {
    try {
      var keys = Object.keys(el);
      for (var i = 0; i < keys.length; i++) {
        var k = keys[i];
        if (k.indexOf('__reactFiber') === 0 || k.indexOf('__reactInternalInstance') === 0) {
          var fiber = el[k];
          var depth = 0;
          while (fiber && depth < 40) {
            var type = fiber.type;
            if (type) {
              var name =
                (typeof type === 'string' ? type : null) ||
                type.displayName ||
                type.name ||
                (type.render && (type.render.displayName || type.render.name));
              if (name && name !== 'Anonymous' && !/^[a-z]/.test(name)) {
                var file =
                  (type._source && type._source.fileName) ||
                  (fiber._debugSource && fiber._debugSource.fileName) ||
                  null;
                return { name: name, file: file, framework: 'react' };
              }
            }
            fiber = fiber.return;
            depth++;
          }
        }
      }
      var vue = el.__vueParentComponent || (el.__vnode && el.__vnode.component);
      if (vue) {
        var vt = vue.type || {};
        var vname = vt.name || vt.__name || (vt.__file && vt.__file.split('/').pop()) || null;
        var vfile = vt.__file || null;
        if (vname || vfile) return { name: vname || 'Anonymous', file: vfile, framework: 'vue' };
      }
    } catch (e) { /* ignore */ }
    return null;
  }

  function extract(el) {
    el = resolveTarget(el);
    var r = el.getBoundingClientRect();
    var html = el.outerHTML || '';
    if (html.length > 1500) html = html.slice(0, 1500) + '...';
    var text = (el.innerText || el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 500);
    var selectors = buildSelectors(el);
    var classes = Array.prototype.slice.call(el.classList || []);
    var testId =
      el.getAttribute('data-testid') ||
      el.getAttribute('data-test') ||
      el.getAttribute('data-qa') ||
      el.getAttribute('data-cy') ||
      '';

    return {
      type: 'dq-inspect-selected',
      tag: el.tagName.toLowerCase(),
      text: text,
      outerHTML: html,
      id: el.id || '',
      classes: classes,
      role: el.getAttribute('role') || '',
      ariaLabel: el.getAttribute('aria-label') || '',
      name: el.getAttribute('name') || '',
      placeholder: el.getAttribute('placeholder') || '',
      testId: testId,
      selectors: selectors,
      xpath: buildXPath(el),
      boundingBox: {
        x: Math.round(r.x),
        y: Math.round(r.y),
        w: Math.round(r.width),
        h: Math.round(r.height),
      },
      viewport: { w: window.innerWidth, h: window.innerHeight },
      attributes: collectAttributes(el),
      component: detectComponent(el),
      page: {
        url: location.href,
        title: document.title || '',
      },
      // legacy fields for older listeners
      html: html,
    };
  }

  function onMouseOver(e) {
    var el = resolveTarget(e.target);
    if (isSkip(el)) return;
    var o = ensureOverlay();
    var r = el.getBoundingClientRect();
    o.style.display = 'block';
    o.style.top = r.top + 'px';
    o.style.left = r.left + 'px';
    o.style.width = r.width + 'px';
    o.style.height = r.height + 'px';
  }

  function stop() {
    active = false;
    document.removeEventListener('mouseover', onMouseOver, true);
    document.removeEventListener('click', onClick, true);
    document.removeEventListener('keydown', onKeyDown, true);
    document.body.style.cursor = '';
    if (overlay) {
      overlay.style.display = 'none';
      if (overlay.isConnected) overlay.remove();
      overlay = null;
    }
  }

  function onClick(e) {
    e.preventDefault();
    e.stopPropagation();
    var el = resolveTarget(e.target);
    if (isSkip(el)) return;
    var payload = extract(el);
    stop();
    window.parent.postMessage(payload, '*');
  }

  function onKeyDown(e) {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      stop();
      window.parent.postMessage({ type: 'dq-inspect-cancel' }, '*');
    }
  }

  function start() {
    if (active) return;
    active = true;
    ensureOverlay();
    document.body.style.cursor = 'crosshair';
    document.addEventListener('mouseover', onMouseOver, true);
    document.addEventListener('click', onClick, true);
    document.addEventListener('keydown', onKeyDown, true);
  }

  window.addEventListener('message', function (e) {
    if (!e.data) return;
    if (e.data.type === 'dq-inspect-start') start();
    if (e.data.type === 'dq-inspect-stop') stop();
  });
})();

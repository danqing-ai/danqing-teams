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

  var STYLE_PROPS = [
    'display', 'position', 'box-sizing',
    'width', 'height', 'min-width', 'min-height', 'max-width', 'max-height',
    'margin', 'padding',
    'color', 'background-color', 'background-image', 'background-size', 'background-position',
    'font-family', 'font-size', 'font-weight', 'font-style', 'line-height', 'letter-spacing',
    'text-align', 'text-decoration', 'text-transform', 'white-space',
    'border', 'border-radius', 'outline',
    'opacity', 'overflow', 'visibility', 'z-index',
    'flex', 'flex-direction', 'flex-wrap', 'justify-content', 'align-items', 'align-self', 'gap',
    'grid-template-columns', 'grid-template-rows', 'grid-gap',
    'box-shadow', 'transform', 'cursor', 'object-fit',
  ];

  function collectComputedStyles(el) {
    var out = {};
    try {
      var cs = window.getComputedStyle(el);
      for (var i = 0; i < STYLE_PROPS.length; i++) {
        var prop = STYLE_PROPS[i];
        var val = cs.getPropertyValue(prop);
        if (!val || val === 'none' || val === 'normal' || val === 'auto' || val === 'static' ||
            val === 'visible' || val === 'rgba(0, 0, 0, 0)' || val === 'transparent' ||
            val === '0px' || val === 'none repeat scroll 0% 0% / auto padding-box border-box') {
          continue;
        }
        // Skip huge background-image data URLs
        if (prop === 'background-image' && val.length > 120) {
          out[prop] = val.slice(0, 80) + '…';
          continue;
        }
        out[prop] = val;
      }
    } catch (e) { /* ignore */ }
    return out;
  }

  function summarizeNode(el) {
    if (!el || el.nodeType !== 1) return '';
    var tag = el.tagName.toLowerCase();
    var bits = ['<' + tag];
    if (el.id) bits.push(' id="' + el.id + '"');
    var cls = Array.prototype.slice.call(el.classList || []).slice(0, 4);
    if (cls.length) bits.push(' class="' + cls.join(' ') + '"');
    var role = el.getAttribute('role');
    if (role) bits.push(' role="' + role + '"');
    var aria = el.getAttribute('aria-label');
    if (aria) bits.push(' aria-label="' + aria.slice(0, 40) + '"');
    var text = (el.innerText || '').trim().replace(/\s+/g, ' ').slice(0, 40);
    bits.push('>');
    if (text) bits.push(text);
    bits.push('</' + tag + '>');
    return bits.join('');
  }

  function openTagSummary(el) {
    var tag = el.tagName.toLowerCase();
    var bits = ['<' + tag];
    if (el.id) bits.push(' id="' + el.id + '"');
    var cls = Array.prototype.slice.call(el.classList || []).slice(0, 5);
    if (cls.length) bits.push(' class="' + cls.join(' ') + '"');
    bits.push('>');
    return bits.join('');
  }

  function collectNeighborhood(el) {
    var parent = el.parentElement;
    if (!parent || parent === document.documentElement) {
      var solo = el.outerHTML || '';
      return solo.length > 2200 ? solo.slice(0, 2200) + '...' : solo;
    }
    var lines = [];
    lines.push(openTagSummary(parent));
    var children = parent.children;
    var maxSiblings = 8;
    var shown = 0;
    var targetShown = false;
    for (var i = 0; i < children.length; i++) {
      var child = children[i];
      if (child === el) {
        var html = el.outerHTML || '';
        if (html.length > 1200) html = html.slice(0, 1200) + '...';
        lines.push('  <!-- selected -->');
        lines.push('  ' + html);
        targetShown = true;
        shown++;
      } else if (shown < maxSiblings || (!targetShown && i >= children.length - 2)) {
        lines.push('  <!-- sibling --> ' + summarizeNode(child));
        shown++;
      } else if (i === children.length - 1 && !targetShown) {
        lines.push('  <!-- … -->');
      }
    }
    lines.push('</' + parent.tagName.toLowerCase() + '>');
    var out = lines.join('\n');
    if (out.length > 2800) out = out.slice(0, 2800) + '\n...';
    return out;
  }

  var SCREENSHOT_STYLE_PROPS = STYLE_PROPS.concat([
    'top', 'left', 'right', 'bottom', 'inset',
    'vertical-align', 'list-style', 'table-layout',
  ]);

  function stripCaptureNoise(root) {
    if (!root || !root.querySelectorAll) return;
    var bad = root.querySelectorAll('script,style,link,iframe,video,audio,canvas,noscript');
    for (var i = bad.length - 1; i >= 0; i--) {
      if (bad[i].parentNode) bad[i].parentNode.removeChild(bad[i]);
    }
  }

  function copyVisualStyles(src, dst, depth) {
    if (!src || !dst || src.nodeType !== 1 || dst.nodeType !== 1 || depth > 5) return;
    try {
      var cs = window.getComputedStyle(src);
      var css = [];
      for (var i = 0; i < SCREENSHOT_STYLE_PROPS.length; i++) {
        var prop = SCREENSHOT_STYLE_PROPS[i];
        var val = cs.getPropertyValue(prop);
        if (val) css.push(prop + ':' + val);
      }
      dst.setAttribute('style', css.join(';'));
    } catch (e) { /* ignore */ }
    var sc = src.children;
    var dc = dst.children;
    var n = Math.min(sc.length, dc.length, 24);
    for (var j = 0; j < n; j++) {
      copyVisualStyles(sc[j], dc[j], depth + 1);
    }
  }

  function captureScreenshot(el) {
    return new Promise(function (resolve) {
      var settled = false;
      function done(v) {
        if (settled) return;
        settled = true;
        resolve(v || null);
      }
      try {
        var rect = el.getBoundingClientRect();
        var w = Math.max(1, Math.ceil(rect.width));
        var h = Math.max(1, Math.ceil(rect.height));
        if (w < 2 || h < 2) {
          done(null);
          return;
        }
        var maxSide = 720;
        var scale = Math.min(2, maxSide / Math.max(w, h));
        if (scale > 1) scale = Math.min(scale, window.devicePixelRatio || 1);

        var clone = el.cloneNode(true);
        copyVisualStyles(el, clone, 0);
        stripCaptureNoise(clone);

        var wrapper = document.createElement('div');
        wrapper.setAttribute('xmlns', 'http://www.w3.org/1999/xhtml');
        wrapper.style.cssText =
          'margin:0;padding:0;width:' + w + 'px;height:' + h + 'px;overflow:hidden;background:#fff;';
        wrapper.appendChild(clone);

        var serialized = new XMLSerializer().serializeToString(wrapper);
        var svg =
          '<svg xmlns="http://www.w3.org/2000/svg" width="' + w + '" height="' + h + '">' +
          '<foreignObject width="100%" height="100%">' +
          serialized +
          '</foreignObject></svg>';

        var img = new Image();
        var timer = setTimeout(function () { done(null); }, 1600);
        img.onload = function () {
          clearTimeout(timer);
          try {
            var canvas = document.createElement('canvas');
            canvas.width = Math.max(1, Math.round(w * scale));
            canvas.height = Math.max(1, Math.round(h * scale));
            var ctx = canvas.getContext('2d');
            if (!ctx) {
              done(null);
              return;
            }
            ctx.fillStyle = '#ffffff';
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
            done(canvas.toDataURL('image/jpeg', 0.84));
          } catch (e) {
            done(null);
          }
        };
        img.onerror = function () {
          clearTimeout(timer);
          done(null);
        };
        img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg);
      } catch (e) {
        done(null);
      }
    });
  }

  function extract(el) {
    el = resolveTarget(el);
    var r = el.getBoundingClientRect();
    var html = el.outerHTML || '';
    if (html.length > 1800) html = html.slice(0, 1800) + '...';
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
      neighborhoodHTML: collectNeighborhood(el),
      computedStyles: collectComputedStyles(el),
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
    // Hide overlay so it does not appear in the crop.
    if (overlay) overlay.style.display = 'none';
    var payload = extract(el);
    var target = el;
    captureScreenshot(target).then(function (dataUrl) {
      if (dataUrl) payload.screenshot = dataUrl;
      stop();
      window.parent.postMessage(payload, '*');
    });
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

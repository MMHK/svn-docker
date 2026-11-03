/**
 * iF.SVNAdmin — 前端行為（原生 JS，取代舊的 templates/script.js）
 *
 * 對外保留舊頁面 inline handler 使用的全域函式名稱：
 *   filterDataTable(), selectAll(), deletionPrompt(), highlightDatatables(),
 *   flashElement(), ChangeLocale()
 */
(function () {
  'use strict';

  /* ---------------------------------------------------------------- utils */

  function ready(fn) {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', fn);
    } else {
      fn();
    }
  }

  function each(list, fn) {
    Array.prototype.forEach.call(list, fn);
  }

  /* ------------------------------------------------------- 資料表篩選/選取 */

  /**
   * 依文字內容過濾表格列表。
   * @param {string} tableId
   * @param {number} colIndex 保留相容參數（現行實作比對整列文字）
   * @param {string} text
   */
  function filterDataTable(tableId, colIndex, text) {
    var table = document.getElementById(tableId);
    if (!table) {
      return;
    }
    var bodies = table.getElementsByTagName('tbody');
    if (bodies.length < 1) {
      return;
    }
    var rows = bodies[bodies.length - 1].getElementsByTagName('tr');
    var needle = String(text == null ? '' : text).toLowerCase();

    each(rows, function (row) {
      var haystack = (row.textContent || '').toLowerCase();
      row.style.display = haystack.indexOf(needle) === -1 ? 'none' : '';
    });
  }

  /**
   * 勾選/取消「全選」核取方塊。
   * 2 參數簽名為舊頁面相容；未給 name 時自動選取同一張表格內的項目。
   * @param {HTMLInputElement} srcObj
   * @param {string=} targetCheckBoxName
   */
  function selectAll(srcObj, targetCheckBoxName) {
    var scope = srcObj && srcObj.closest ? srcObj.closest('table') : document;
    var selector = 'input[type="checkbox"]';
    if (targetCheckBoxName) {
      selector = 'input[name="' + targetCheckBoxName + '"][type="checkbox"]';
      scope = document;
    }
    each(scope.querySelectorAll(selector), function (box) {
      if (box === srcObj) {
        return;
      }
      var row = box.closest ? box.closest('tr') : null;
      if (row && row.style.display === 'none') {
        return; // 只選取過濾後仍可見的列
      }
      box.checked = srcObj.checked;
    });
  }

  /** 刪除確認（取代舊 deletionPrompt 的 confirm 包裝）。 */
  function deletionPrompt(question) {
    return window.confirm(question);
  }

  /** 交替列底色（保留舊頁面的 .trbase / .tralt 掛點）。 */
  function highlightDatatables() {
    each(document.querySelectorAll('table.datatable'), function (table) {
      var bodies = table.getElementsByTagName('tbody');
      if (bodies.length < 1) {
        return;
      }
      each(bodies[bodies.length - 1].getElementsByTagName('tr'), function (row, i) {
        row.classList.remove('trbase', 'tralt');
        row.classList.add(i % 2 === 0 ? 'trbase' : 'tralt');
      });
    });
  }

  /** 閃爍提示元素（取代舊版 fadeOut/fadeIn 迴圈）。 */
  function flashElement(elem) {
    if (!elem || !elem.classList) {
      return;
    }
    elem.classList.remove('flash-highlight');
    void elem.offsetWidth; // 重設動畫
    elem.classList.add('flash-highlight');
  }

  /** 切換語系（保留 window.location 的 query string）。 */
  function ChangeLocale(localeCode) {
    var url = window.location.href;
    var separator = url.indexOf('?') === -1 ? '?' : '&';
    window.location.href = url + separator + 'locale=' + encodeURIComponent(localeCode);
  }

  /* ------------------------------------------------------------ 深色模式 */

  var COLOR_MODES = ['auto', 'light', 'dark'];

  function currentColorMode() {
    var mode = document.documentElement.getAttribute('data-color-mode') || 'auto';
    return COLOR_MODES.indexOf(mode) === -1 ? 'auto' : mode;
  }

  function writeColorModeCookie(mode) {
    document.cookie =
      'color_mode=' + encodeURIComponent(mode) +
      '; path=/; max-age=' + (60 * 60 * 24 * 365) + '; SameSite=Lax';
  }

  function applyColorMode(mode) {
    document.documentElement.setAttribute('data-color-mode', mode);
    each(document.querySelectorAll('[data-color-mode-toggle]'), function (btn) {
      btn.setAttribute('aria-label', 'Color mode: ' + mode);
      btn.setAttribute('title', 'Color mode: ' + mode);
      var icon = btn.querySelector('[data-color-mode-icon]');
      if (icon) {
        icon.textContent = mode === 'dark' ? '\u263e' : (mode === 'light' ? '\u2600' : '\u25d1');
      }
    });
  }

  function cycleColorMode() {
    var next = COLOR_MODES[(COLOR_MODES.indexOf(currentColorMode()) + 1) % COLOR_MODES.length];
    writeColorModeCookie(next);
    applyColorMode(next);
  }

  /* ------------------------------------------------------------ 初始化 */
  ready(function () {
    highlightDatatables();

    // 全選核取方塊（#selectall / #selectallusers / #selectallgroups / #selectallpaths）
    each(document.querySelectorAll('input[type="checkbox"][id^="selectall"]'), function (box) {
      box.addEventListener('change', function () {
        selectAll(box);
      });
    });

    // 語系選擇
    each(document.querySelectorAll('select#locale-selector, select[data-locale-selector]'), function (select) {
      select.addEventListener('change', function () {
        ChangeLocale(select.value);
      });
    });

    // 深色模式
    applyColorMode(currentColorMode());
    each(document.querySelectorAll('[data-color-mode-toggle]'), function (btn) {
      btn.addEventListener('click', cycleColorMode);
    });

    // 行動版導航開合
    each(document.querySelectorAll('[data-nav-toggle]'), function (btn) {
      btn.addEventListener('click', function () {
        var nav = document.getElementById(btn.getAttribute('data-nav-toggle') || 'global-navigation');
        if (!nav) {
          return;
        }
        nav.classList.toggle('is-open');
        btn.setAttribute('aria-expanded', String(nav.classList.contains('is-open')));
      });
    });

    // 依 select 值顯示對應區塊（repositorycreate 的預設結構說明）
    each(document.querySelectorAll('select[data-toggle-targets]'), function (select) {
      var group = select.getAttribute('data-toggle-targets');
      var sync = function () {
        each(document.querySelectorAll('[data-toggle-group="' + group + '"]'), function (el) {
          var showFor = el.getAttribute('data-toggle-when');
          el.style.display = showFor === select.value ? '' : 'none';
        });
      };
      select.addEventListener('change', sync);
      sync();
    });

    // flash 訊息閃爍一次
    each(document.querySelectorAll('.flash'), flashElement);
  });

  window.filterDataTable = filterDataTable;
  window.selectAll = selectAll;
  window.deletionPrompt = deletionPrompt;
  window.highlightDatatables = highlightDatatables;
  window.flashElement = flashElement;
  window.ChangeLocale = ChangeLocale;
})();

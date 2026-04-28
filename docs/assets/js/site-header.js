(function () {
  var menuToggle = document.getElementById("site-header-menu-toggle");
  if (menuToggle) {
    menuToggle.addEventListener("click", function () {
      var ctrl = document.getElementById("menu-control");
      if (ctrl) ctrl.click();
    });
  }
})();

(function () {
  var toggle = document.querySelector("[data-theme-toggle]");
  if (!toggle) return;
  toggle.addEventListener("click", function () {
    var saved = null;
    try { saved = localStorage.getItem("site-theme"); } catch (_) {}

    var prefersDark = window.matchMedia &&
      window.matchMedia("(prefers-color-scheme: dark)").matches;
    var osTheme = prefersDark ? "dark" : "light";
    var visible = saved || osTheme;
    var nextVisible = visible === "dark" ? "light" : "dark";

    /* If the next visible theme matches the OS preference, store no
     * explicit choice (auto). Otherwise pin it explicitly. This way every
     * click changes the visible theme, and "follow OS" is reachable again
     * without clearing storage manually. */
    if (nextVisible === osTheme) {
      document.documentElement.removeAttribute("data-theme");
      try { localStorage.removeItem("site-theme"); } catch (_) {}
    } else {
      document.documentElement.setAttribute("data-theme", nextVisible);
      try { localStorage.setItem("site-theme", nextVisible); } catch (_) {}
    }
  });
})();

(function () {
  var link = document.querySelector(".site-header__repo[data-repo]");
  if (!link) return;

  var repo = link.getAttribute("data-repo");
  if (!repo) return;

  var TTL = 60 * 60 * 1000;
  var key = "site-header:v1:" + repo;

  function fmt(n) {
    if (n == null) return null;
    if (n >= 10000) return Math.round(n / 1000) + "k";
    if (n >= 1000) {
      var s = (n / 1000).toFixed(1);
      return s.replace(/\.0$/, "") + "k";
    }
    return String(n);
  }

  function apply(data) {
    if (!data) return;
    function set(sel, val) {
      var li = link.querySelector(sel);
      if (!li || val == null) return;
      var span = li.querySelector(".site-header__fact-value");
      if (span) span.textContent = val;
      li.hidden = false;
    }
    set(".site-header__fact--version", data.version);
    set(".site-header__fact--stars", fmt(data.stars));
    set(".site-header__fact--forks", fmt(data.forks));
  }

  try {
    var raw = localStorage.getItem(key);
    if (raw) {
      var cached = JSON.parse(raw);
      if (cached && Date.now() - cached.t < TTL) {
        apply(cached.d);
        return;
      }
    }
  } catch (_) {}

  function getJSON(url) {
    return fetch(url, { headers: { Accept: "application/vnd.github+json" } })
      .then(function (r) { return r.ok ? r.json() : null; })
      .catch(function () { return null; });
  }

  Promise.all([
    getJSON("https://api.github.com/repos/" + repo),
    getJSON("https://api.github.com/repos/" + repo + "/releases/latest"),
  ]).then(function (results) {
    var info = results[0];
    var release = results[1];
    var data = {
      stars: info ? info.stargazers_count : null,
      forks: info ? info.forks_count : null,
      version: release ? release.tag_name || release.name : null,
    };
    apply(data);
    try {
      localStorage.setItem(key, JSON.stringify({ t: Date.now(), d: data }));
    } catch (_) {}
  });
})();

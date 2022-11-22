function initBreadcrumb() {

  const breadcrumb = document.getElementById("breadcrumb");
  if (!breadcrumb) return;

  const page = location.pathname.substring(location.pathname.lastIndexOf("/") + 1);

  const crumbs = [{
    name: "Quick Start Guide",
    url: location.pathname.substring(0, location.pathname.lastIndexOf("/") + 1)
  }];

  let url = "";
  for (let current of page.split("_")) {
    url = !url ? current : `${url}_${current}`;
    crumbs.push({ name: current, url: `${url}.html` });
  }

  let html = "";
  for (let i = 0; i < crumbs.length - 1; i++) {
    html += `<a href="${crumbs[i].url}" class="bc-crumb">${crumbs[i].name}</a><span class="bc-seperator">/</span>`;
  }
  html += `<span class="bc-current">${crumbs[crumbs.length - 1].name.replace(".html", "")}</span>`;

  breadcrumb.innerHTML = html;
}

function highlightCode() {
  const codes = document.getElementsByTagName("code");
  for (let code of codes) {
    const lines = code.innerHTML.split("\n");
    for (let i = 0; i < lines.length; i++) {
      switch (true) {
        case lines[i].trimStart().startsWith("#"):
          lines[i] = `<span class="code-comment">${lines[i]}</span>`;
          break
        case lines[i].trimStart().startsWith("&gt;"):
          lines[i] = `<span class="code-command">${lines[i]}</span>`;
          break;
      }
    }
    code.innerHTML = lines.join("\n").trim();
  }
}

function initManPage() {
  initBreadcrumb();
  highlightCode();
}
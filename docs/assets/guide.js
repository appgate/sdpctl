function initBreadcrumb() {

  const breadcrumb = document.getElementById("breadcrumb");
  if (!breadcrumb) return;

  const page = location.pathname.substring(location.pathname.lastIndexOf("/") + 1);

  const crumbs = [{
    name: "Quickstart Guide",
    url: location.pathname.substring(0, location.pathname.lastIndexOf("/") + 1)
  }];

  let url = "";
  for (let current of page.split("_")) {
    url = !url ? current : `${url}_${current}`;
    crumbs.push({ name: current, url: `${url}.html` });
  }
  console.log(crumbs);
  let html = "";
  for (let i = 0; i < crumbs.length - 1; i++) {
    html += `<a href="${crumbs[i].url}" class="bc-crumb">${crumbs[i].name}</a><span class="bc-seperator">/</span>`;
  }
  html += `<span class="bc-current">${crumbs[crumbs.length - 1].name.replace(".html", "")}</span>`;

  breadcrumb.innerHTML = html;
}
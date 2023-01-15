export function getQueryParams() {
  return new URLSearchParams(window.location.search);
}

export function setQueryParams(params: any) {
  const url = new URL(window.location.href);
  url.search = params.toString();
  window.history.replaceState(null, document.title, url.toString());
}

/* eslint-disable no-restricted-globals */
// eslint-disable-next-line import/prefer-default-export
export function getScroll() {
  if (window.pageYOffset !== undefined) {
    return [window.pageXOffset, window.pageYOffset];
  }
  const r = document.documentElement;
  const b = document.body;
  const sx = r.scrollLeft || b.scrollLeft || 0;
  const sy = r.scrollTop || b.scrollTop || 0;
  return [sx, sy];
}

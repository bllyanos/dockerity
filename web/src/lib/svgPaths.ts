/** Returns the d attribute of every path in an SVG document, in order. */
export function svgPaths(svg: string): string[] {
  return [...svg.matchAll(/<path[^>]*\sd="([^"]+)"/g)].map((m) => m[1])
}

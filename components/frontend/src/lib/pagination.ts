/**
 * Computes which page numbers (and ellipsis markers) to render in a
 * pagination control.
 *
 * Always shows the first page, last page, and up to `siblingCount` pages
 * on each side of the current page. Gaps are represented by the string
 * `'ellipsis'`.
 *
 * Example with currentPage=5, totalPages=10, siblingCount=1:
 *   [1, 'ellipsis', 4, 5, 6, 'ellipsis', 10]
 */
export function getPageNumbers(
  currentPage: number,
  totalPages: number,
  siblingCount = 1
): (number | 'ellipsis')[] {
  // If total pages is small enough, show them all
  const totalSlots = siblingCount * 2 + 5; // siblings + first + last + current + 2 ellipses
  if (totalPages <= totalSlots) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }

  const leftSiblingIndex = Math.max(currentPage - siblingCount, 1);
  const rightSiblingIndex = Math.min(currentPage + siblingCount, totalPages);

  const showLeftEllipsis = leftSiblingIndex > 2;
  const showRightEllipsis = rightSiblingIndex < totalPages - 1;

  const pages: (number | 'ellipsis')[] = [];

  // Always include first page
  pages.push(1);

  if (showLeftEllipsis) {
    pages.push('ellipsis');
  } else {
    // Fill in pages between 1 and leftSiblingIndex
    for (let i = 2; i < leftSiblingIndex; i++) {
      pages.push(i);
    }
  }

  // Sibling pages and current page
  for (let i = leftSiblingIndex; i <= rightSiblingIndex; i++) {
    if (i !== 1 && i !== totalPages) {
      pages.push(i);
    }
  }

  if (showRightEllipsis) {
    pages.push('ellipsis');
  } else {
    // Fill in pages between rightSiblingIndex and totalPages
    for (let i = rightSiblingIndex + 1; i < totalPages; i++) {
      pages.push(i);
    }
  }

  // Always include last page
  if (totalPages > 1) {
    pages.push(totalPages);
  }

  return pages;
}

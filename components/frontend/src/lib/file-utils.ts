/**
 * Common binary file extensions that should not be rendered as text
 */
const BINARY_EXTENSIONS = new Set([
  // Archives
  'zip', 'tar', 'gz', 'bz2', 'xz', '7z', 'rar',
  // Images
  'png', 'jpg', 'jpeg', 'gif', 'bmp', 'ico', 'webp', 'svg', 'tiff', 'tif',
  // Documents
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods', 'odp',
  // Media
  'mp3', 'mp4', 'avi', 'mov', 'wmv', 'flv', 'webm', 'wav', 'ogg', 'flac',
  // Executables & compiled
  'exe', 'dll', 'so', 'dylib', 'bin', 'o', 'a', 'class', 'pyc', 'pyo',
  // Fonts
  'ttf', 'otf', 'woff', 'woff2', 'eot',
  // Other binary
  'sqlite', 'db', 'iso', 'dmg', 'deb', 'rpm',
]);

/**
 * Check if a file path represents a binary file based on its extension
 */
export function isBinaryFile(filePath: string): boolean {
  const ext = filePath.split('.').pop()?.toLowerCase();
  return ext ? BINARY_EXTENSIONS.has(ext) : false;
}

/**
 * Check if content appears to be binary (contains null bytes or high ratio of non-printable characters)
 */
export function contentAppearsBinary(content: string): boolean {
  // Check for null bytes (common in binary files that were forced to text)
  if (content.includes('\0')) {
    return true;
  }

  // Check first 1000 characters for a high ratio of non-printable characters
  const sample = content.slice(0, 1000);
  let nonPrintable = 0;

  for (let i = 0; i < sample.length; i++) {
    const code = sample.charCodeAt(i);
    // Count characters that are neither printable ASCII nor common whitespace
    if (code < 32 && code !== 9 && code !== 10 && code !== 13) {
      nonPrintable++;
    }
  }

  // If more than 10% non-printable, likely binary
  return sample.length > 0 && nonPrintable / sample.length > 0.1;
}

import { describe, it, expect } from 'vitest'
import { isBinaryFile, contentAppearsBinary } from '../file-utils'

describe('isBinaryFile', () => {
  it('returns true for common binary extensions', () => {
    const binaryFiles = [
      'archive.zip',
      'image.png',
      'photo.jpg',
      'document.pdf',
      'video.mp4',
      'font.woff2',
      'compiled.exe',
      'library.dll',
      'database.sqlite',
    ]

    for (const file of binaryFiles) {
      expect(isBinaryFile(file)).toBe(true)
    }
  })

  it('returns false for text-based extensions', () => {
    const textFiles = [
      'script.js',
      'styles.css',
      'document.txt',
      'config.json',
      'markup.html',
      'code.ts',
      'readme.md',
      'data.xml',
      'code.py',
      'code.go',
    ]

    for (const file of textFiles) {
      expect(isBinaryFile(file)).toBe(false)
    }
  })

  it('handles paths with directories', () => {
    expect(isBinaryFile('/path/to/file.png')).toBe(true)
    expect(isBinaryFile('relative/path/file.js')).toBe(false)
  })

  it('handles case insensitivity', () => {
    expect(isBinaryFile('IMAGE.PNG')).toBe(true)
    expect(isBinaryFile('Archive.ZIP')).toBe(true)
    expect(isBinaryFile('Photo.JPG')).toBe(true)
  })

  it('returns false for files without extension', () => {
    expect(isBinaryFile('Makefile')).toBe(false)
    expect(isBinaryFile('Dockerfile')).toBe(false)
    expect(isBinaryFile('.gitignore')).toBe(false)
  })

  it('handles files with multiple dots', () => {
    expect(isBinaryFile('file.test.png')).toBe(true)
    expect(isBinaryFile('archive.tar.gz')).toBe(true)
    expect(isBinaryFile('component.test.ts')).toBe(false)
  })
})

describe('contentAppearsBinary', () => {
  it('returns true for content with null bytes', () => {
    const binaryContent = 'some text\0with null bytes'
    expect(contentAppearsBinary(binaryContent)).toBe(true)
  })

  it('returns false for normal text content', () => {
    const textContent = 'This is normal text content.\nWith newlines and tabs.\t'
    expect(contentAppearsBinary(textContent)).toBe(false)
  })

  it('returns false for code content', () => {
    const codeContent = `
function hello() {
  console.log("Hello, world!");
  return 42;
}
`
    expect(contentAppearsBinary(codeContent)).toBe(false)
  })

  it('returns true for content with many non-printable characters', () => {
    // Create content with >10% non-printable characters (control chars 0-31 except tab, newline, CR)
    const nonPrintableChars = String.fromCharCode(1, 2, 3, 4, 5, 6, 7, 8)
    const padding = 'a'.repeat(50) // 50 'a' characters
    const binaryContent = nonPrintableChars + padding // 8 non-printable out of 58 = ~14%
    expect(contentAppearsBinary(binaryContent)).toBe(true)
  })

  it('returns false for content with few non-printable characters', () => {
    // Create content with <10% non-printable characters
    const nonPrintableChars = String.fromCharCode(1)
    const padding = 'a'.repeat(100) // 100 'a' characters
    const mixedContent = nonPrintableChars + padding // 1 non-printable out of 101 = ~1%
    expect(contentAppearsBinary(mixedContent)).toBe(false)
  })

  it('returns false for empty content', () => {
    expect(contentAppearsBinary('')).toBe(false)
  })

  it('handles content with common whitespace correctly', () => {
    // Tab (9), newline (10), carriage return (13) should NOT be counted as non-printable
    const whitespaceContent = 'line1\nline2\rline3\tindented'
    expect(contentAppearsBinary(whitespaceContent)).toBe(false)
  })

  it('samples first 1000 characters for non-printable ratio check', () => {
    // Non-printable chars (not null bytes) after first 1000 chars should not affect the ratio
    const normalText = 'a'.repeat(1000)
    // Use control chars that aren't null bytes (which are checked separately for the whole content)
    const binaryAfter = String.fromCharCode(1, 2, 3, 4, 5).repeat(100)
    const content = normalText + binaryAfter
    expect(contentAppearsBinary(content)).toBe(false)
  })

  it('checks for null bytes in entire content', () => {
    // Null bytes anywhere in content should be detected
    const normalText = 'a'.repeat(2000)
    const contentWithNullAtEnd = normalText + '\0'
    expect(contentAppearsBinary(contentWithNullAtEnd)).toBe(true)
  })
})

import { buildForwardHeadersAsync } from '@/lib/auth';
import { BACKEND_URL } from '@/lib/config';
import { NextRequest } from 'next/server';
import { fileTypeFromBuffer } from 'file-type';

// Maximum file sizes based on type
// SDK has 1MB JSON limit, base64 adds ~33% overhead, plus JSON structure overhead
// Conservative compression target: 350KB raw → ~467KB base64 → ~490KB total (safe margin)
// Text files don't get base64 encoded, so they can be larger (700KB safe limit)
// These limits are configurable via environment variables to allow different values per environment
const MAX_DOCUMENT_SIZE = parseInt(process.env.MAX_UPLOAD_SIZE_DOCUMENTS || '716800'); // Default 700KB for documents
const MAX_IMAGE_SIZE = parseInt(process.env.MAX_UPLOAD_SIZE_IMAGES || '3145728'); // Default 3MB upload limit
const IMAGE_COMPRESSION_TARGET = parseInt(process.env.IMAGE_COMPRESSION_TARGET || '358400'); // Default 350KB target

// Determine if a file is an image based on content type
const isImageFile = (contentType: string): boolean => {
  return contentType.startsWith('image/');
};

// Get the appropriate max file size based on content type
const getMaxFileSize = (contentType: string): number => {
  return isImageFile(contentType) ? MAX_IMAGE_SIZE : MAX_DOCUMENT_SIZE;
};

// Format size limit for error messages
const formatSizeLimit = (contentType: string): string => {
  const maxSize = getMaxFileSize(contentType);
  const sizeInKB = Math.round(maxSize / 1024);
  const fileType = isImageFile(contentType) ? 'images' : 'documents';
  return `${sizeInKB}KB for ${fileType}`;
};

// Sanitize filename to prevent path traversal and malicious characters
// Removes path separators (/, \, ..), null bytes, and limits length
function sanitizeFilename(filename: string): string {
  // Remove path separators and null bytes
  return filename.replace(/[\/\\\0]/g, '_').substring(0, 255);
}

// Validate URL to prevent SSRF attacks
// Returns true if URL is safe to fetch, false otherwise
function isValidUrl(urlString: string): boolean {
  try {
    const url = new URL(urlString);

    // Only allow http and https protocols
    if (!['http:', 'https:'].includes(url.protocol)) {
      return false;
    }

    // Block private IP ranges and localhost
    const hostname = url.hostname.toLowerCase();

    // Block localhost
    if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1') {
      return false;
    }

    // Block private IPv4 ranges (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
    const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
    const ipv4Match = hostname.match(ipv4Regex);
    if (ipv4Match) {
      const [, a, b, c, d] = ipv4Match.map(Number);
      // Check if any octet is invalid
      if (a > 255 || b > 255 || c > 255 || d > 255) {
        return false;
      }
      // Block private ranges
      if (a === 10 || (a === 172 && b >= 16 && b <= 31) || (a === 192 && b === 168)) {
        return false;
      }
      // Block link-local (169.254.0.0/16)
      if (a === 169 && b === 254) {
        return false;
      }
    }

    // Block link-local IPv6 (fe80::/10)
    if (hostname.startsWith('fe80:') || hostname.startsWith('[fe80:')) {
      return false;
    }

    return true;
  } catch {
    return false;
  }
}

// Validate file content type via magic bytes
// Returns the actual MIME type detected from file content, or null if detection fails
// This prevents Content-Type header spoofing attacks
async function validateFileType(buffer: ArrayBuffer, claimedType: string): Promise<string> {
  try {
    // Convert ArrayBuffer to Uint8Array for file-type library
    const uint8Array = new Uint8Array(buffer);

    // Detect actual file type from magic bytes
    const detected = await fileTypeFromBuffer(uint8Array);

    if (!detected) {
      // If detection fails, treat as plain text/binary
      // Allow common text-based types that don't have magic bytes (JSON, XML, CSV, YAML, JavaScript, etc.)
      const textTypes = [
        'text/',
        'application/json',
        'application/xml',
        'application/javascript',
        'application/yaml',
        'application/x-yaml',
        'text/yaml',
        'text/csv',
        'application/csv',
        'application/octet-stream',
      ];

      if (textTypes.some(t => claimedType.startsWith(t)) || claimedType === 'application/octet-stream') {
        return claimedType;
      }
      // For other types, reject if we can't verify
      throw new Error('Unable to verify file type. File may be corrupted or unsupported.');
    }

    // Normalize both types for comparison (remove parameters like charset)
    const normalizedClaimed = claimedType.split(';')[0].trim().toLowerCase();
    const normalizedDetected = detected.mime.toLowerCase();

    // Check if types match
    if (normalizedClaimed !== normalizedDetected) {
      // Special case: allow jpeg/jpg variations
      const isJpegVariant = (type: string) => type === 'image/jpeg' || type === 'image/jpg';
      if (isJpegVariant(normalizedClaimed) && isJpegVariant(normalizedDetected)) {
        return detected.mime;
      }

      // Types don't match - reject
      throw new Error(
        `Content-Type mismatch: claimed '${normalizedClaimed}' but detected '${normalizedDetected}'. ` +
        `This may indicate a malicious file or incorrect Content-Type header.`
      );
    }

    // Use the detected type (more trustworthy than header)
    return detected.mime;
  } catch (error) {
    // Re-throw validation errors
    if (error instanceof Error) {
      throw error;
    }
    throw new Error('File type validation failed');
  }
}

// Compress image if it exceeds size limit
// Returns compressed image buffer or original if already small enough, along with compression metadata
// IMPORTANT: Compression preserves original format (PNG stays PNG, JPEG stays JPEG, etc.)
async function compressImageIfNeeded(
  buffer: ArrayBuffer,
  contentType: string,
  maxSize: number
): Promise<{ buffer: ArrayBuffer; compressed: boolean; originalSize: number; finalSize: number; contentType: string }> {
  const originalSize = buffer.byteLength;

  // Only compress actual images (not SVGs or other formats that won't benefit)
  if (!contentType.match(/^image\/(jpeg|jpg|png|webp)$/i)) {
    return { buffer, compressed: false, originalSize, finalSize: originalSize, contentType };
  }

  // If already under limit, return as-is
  if (buffer.byteLength <= maxSize) {
    return { buffer, compressed: false, originalSize, finalSize: originalSize, contentType };
  }

  // Use sharp library for server-side image processing
  try {
    const sharp = (await import('sharp')).default;

    // Determine output format and compression options based on original type
    const isJpeg = contentType.match(/^image\/(jpeg|jpg)$/i);
    const isPng = contentType.match(/^image\/png$/i);
    const isWebp = contentType.match(/^image\/webp$/i);

    let compressed: Buffer;
    let quality = 80;

    // Compress with format-specific settings
    if (isJpeg) {
      // JPEG: use quality compression
      compressed = await sharp(Buffer.from(buffer))
        .jpeg({ quality, mozjpeg: true })
        .toBuffer();

      // Reduce quality iteratively if still too large
      while (compressed.byteLength > maxSize && quality > 20) {
        quality -= 10;
        compressed = await sharp(Buffer.from(buffer))
          .jpeg({ quality, mozjpeg: true })
          .toBuffer();
      }
    } else if (isPng) {
      // PNG: use compression level (lossless) and quality (lossy palette reduction)
      compressed = await sharp(Buffer.from(buffer))
        .png({ quality, compressionLevel: 9, palette: true })
        .toBuffer();

      // Reduce quality iteratively if still too large
      while (compressed.byteLength > maxSize && quality > 20) {
        quality -= 10;
        compressed = await sharp(Buffer.from(buffer))
          .png({ quality, compressionLevel: 9, palette: true })
          .toBuffer();
      }
    } else if (isWebp) {
      // WebP: use quality compression
      compressed = await sharp(Buffer.from(buffer))
        .webp({ quality })
        .toBuffer();

      // Reduce quality iteratively if still too large
      while (compressed.byteLength > maxSize && quality > 20) {
        quality -= 10;
        compressed = await sharp(Buffer.from(buffer))
          .webp({ quality })
          .toBuffer();
      }
    } else {
      // Fallback: shouldn't reach here due to earlier check
      throw new Error(`Unsupported image format: ${contentType}`);
    }

    // If still too large after quality reduction, resize dimensions
    if (compressed.byteLength > maxSize) {
      const metadata = await sharp(Buffer.from(buffer)).metadata();
      const width = metadata.width || 1920;
      const height = metadata.height || 1080;

      // Reduce by 25% iteratively
      let scale = 0.75;
      while (compressed.byteLength > maxSize && scale > 0.25) {
        const sharpInstance = sharp(Buffer.from(buffer))
          .resize(Math.floor(width * scale), Math.floor(height * scale), {
            fit: 'inside',
            withoutEnlargement: true,
          });

        // Apply format-specific compression after resize
        if (isJpeg) {
          compressed = await sharpInstance.jpeg({ quality: 70, mozjpeg: true }).toBuffer();
        } else if (isPng) {
          compressed = await sharpInstance.png({ quality: 70, compressionLevel: 9, palette: true }).toBuffer();
        } else if (isWebp) {
          compressed = await sharpInstance.webp({ quality: 70 }).toBuffer();
        }

        scale -= 0.1;
      }
    }

    const finalSize = compressed.byteLength;

    // Convert Node.js Buffer to ArrayBuffer by creating a new ArrayBuffer and copying data
    const arrayBuffer = new ArrayBuffer(finalSize);
    const view = new Uint8Array(arrayBuffer);
    view.set(compressed);
    // Return with original contentType preserved
    return { buffer: arrayBuffer, compressed: true, originalSize, finalSize, contentType };
  } catch (error) {
    console.error('Failed to compress image:', error);
    // If compression fails, throw error rather than uploading oversized file
    throw new Error('Image too large and compression failed');
  }
}

// Helper function to compress and validate file buffer
// Handles both images (with compression) and non-images (size validation only)
async function compressAndValidate(
  buffer: ArrayBuffer,
  contentType: string
): Promise<{ buffer: ArrayBuffer; contentType: string; compressionInfo: { compressed: boolean; originalSize: number; finalSize: number } }> {
  const maxSize = getMaxFileSize(contentType);

  // For images, compress if needed instead of rejecting
  if (isImageFile(contentType)) {
    try {
      const result = await compressImageIfNeeded(buffer, contentType, IMAGE_COMPRESSION_TARGET);
      return {
        buffer: result.buffer,
        contentType: result.contentType,
        compressionInfo: {
          compressed: result.compressed,
          originalSize: result.originalSize,
          finalSize: result.finalSize,
        },
      };
    } catch (error) {
      console.error('Image compression failed:', error);
      throw new Error(`Image too large and could not be compressed. Please reduce image size and try again.`);
    }
  } else {
    // For non-images, enforce strict size limit
    if (buffer.byteLength > maxSize) {
      throw new Error(`File too large. Maximum size is ${formatSizeLimit(contentType)}`);
    }
    // No compression needed for non-images
    const compressionInfo = {
      compressed: false,
      originalSize: buffer.byteLength,
      finalSize: buffer.byteLength,
    };
    return { buffer, contentType, compressionInfo };
  }
}

// Helper function to upload file to workspace with retry logic
// Handles 202 Accepted responses (content service starting) with retries
async function uploadFileToWorkspace(
  buffer: ArrayBuffer,
  filename: string,
  contentType: string,
  headers: HeadersInit,
  name: string,
  sessionName: string
): Promise<Response> {
  const maxRetries = 3;
  const retryDelay = 2000; // 2 seconds

  for (let retries = 0; retries < maxRetries; retries++) {
    const resp = await fetch(
      `${BACKEND_URL}/projects/${encodeURIComponent(name)}/agentic-sessions/${encodeURIComponent(sessionName)}/workspace/file-uploads/${encodeURIComponent(filename)}`,
      {
        method: 'PUT',
        headers: {
          ...headers,
          'Content-Type': contentType,
        },
        body: buffer,
      }
    );

    // If 202 Accepted (content service starting), wait and retry
    if (resp.status === 202) {
      if (retries < maxRetries - 1) {
        await new Promise((resolve) => setTimeout(resolve, retryDelay));
        continue;
      }
    }

    return resp;
  }

  // Should never reach here, but TypeScript needs a return
  throw new Error('Upload failed after all retries');
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ name: string; sessionName: string }> },
) {
  const { name, sessionName } = await params;
  const headers = await buildForwardHeadersAsync(request);

  try {
    const formData = await request.formData();
    const uploadType = formData.get('type') as string;

    if (uploadType === 'local') {
      // Handle local file upload
      const file = formData.get('file') as File;
      if (!file) {
        return new Response(JSON.stringify({ error: 'No file provided' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      // Sanitize filename to prevent path traversal attacks
      const rawFilename = (formData.get('filename') as string) || file.name;
      const filename = sanitizeFilename(rawFilename);
      const claimedContentType = file.type || 'application/octet-stream';
      const fileArrayBuffer = await file.arrayBuffer();

      // Validate file type via magic bytes to prevent malicious file uploads
      let validatedContentType: string;
      try {
        validatedContentType = await validateFileType(fileArrayBuffer, claimedContentType);
      } catch (error) {
        console.error('File type validation failed:', error);
        return new Response(
          JSON.stringify({
            error: 'File type validation failed'
          }),
          {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }

      // Compress and validate file size
      let fileBuffer: ArrayBuffer;
      let finalContentType: string;
      let compressionInfo: { compressed: boolean; originalSize: number; finalSize: number };

      try {
        const result = await compressAndValidate(fileArrayBuffer, validatedContentType);
        fileBuffer = result.buffer;
        finalContentType = result.contentType;
        compressionInfo = result.compressionInfo;
      } catch (error) {
        return new Response(
          JSON.stringify({ error: error instanceof Error ? error.message : 'File validation failed' }),
          {
            status: 413, // Payload Too Large
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }

      // Upload to workspace with retry logic
      const resp = await uploadFileToWorkspace(fileBuffer, filename, finalContentType, headers, name, sessionName);

      if (!resp.ok) {
        const errorText = await resp.text();
        console.error('Upload failed:', errorText);
        return new Response(JSON.stringify({ error: 'Failed to upload file' }), {
          status: resp.status,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      return new Response(
        JSON.stringify({
          success: true,
          filename,
          compressed: compressionInfo.compressed,
          originalSize: compressionInfo.originalSize,
          finalSize: compressionInfo.finalSize,
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      );
    } else if (uploadType === 'url') {
      // Handle URL-based file upload
      const fileUrl = formData.get('url') as string;
      const rawFilename = formData.get('filename') as string;

      if (!fileUrl || !rawFilename) {
        return new Response(JSON.stringify({ error: 'URL and filename are required' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      // Sanitize filename to prevent path traversal attacks
      const filename = sanitizeFilename(rawFilename);

      // Validate URL to prevent SSRF attacks
      if (!isValidUrl(fileUrl)) {
        return new Response(
          JSON.stringify({
            error: 'Invalid URL: only http/https protocols are allowed and private IPs are blocked'
          }),
          {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }

      // Download the file from URL
      const fileResp = await fetch(fileUrl);
      if (!fileResp.ok) {
        return new Response(JSON.stringify({ error: 'Failed to download file from URL' }), {
          status: 400,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      const claimedContentType = fileResp.headers.get('content-type') || 'application/octet-stream';
      const fileArrayBuffer = await fileResp.arrayBuffer();

      // Validate file type via magic bytes to prevent Content-Type spoofing
      let validatedContentType: string;
      try {
        validatedContentType = await validateFileType(fileArrayBuffer, claimedContentType);
      } catch (error) {
        console.error('File type validation failed:', error);
        return new Response(
          JSON.stringify({
            error: 'File type validation failed'
          }),
          {
            status: 400,
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }

      // Compress and validate file size
      let fileBuffer: ArrayBuffer;
      let finalContentType: string;
      let compressionInfo: { compressed: boolean; originalSize: number; finalSize: number };

      try {
        const result = await compressAndValidate(fileArrayBuffer, validatedContentType);
        fileBuffer = result.buffer;
        finalContentType = result.contentType;
        compressionInfo = result.compressionInfo;
      } catch (error) {
        return new Response(
          JSON.stringify({ error: error instanceof Error ? error.message : 'File validation failed' }),
          {
            status: 413, // Payload Too Large
            headers: { 'Content-Type': 'application/json' },
          }
        );
      }

      // Upload to workspace with retry logic
      const resp = await uploadFileToWorkspace(fileBuffer, filename, finalContentType, headers, name, sessionName);

      if (!resp.ok) {
        const errorText = await resp.text();
        console.error('Upload failed:', errorText);
        return new Response(JSON.stringify({ error: 'Failed to upload file' }), {
          status: resp.status,
          headers: { 'Content-Type': 'application/json' },
        });
      }

      return new Response(
        JSON.stringify({
          success: true,
          filename,
          compressed: compressionInfo.compressed,
          originalSize: compressionInfo.originalSize,
          finalSize: compressionInfo.finalSize,
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      );
    } else {
      return new Response(JSON.stringify({ error: 'Invalid upload type' }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }
  } catch (error) {
    console.error('File upload error:', error);
    return new Response(JSON.stringify({ error: 'Internal server error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
